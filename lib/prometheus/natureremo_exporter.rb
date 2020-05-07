require 'faraday'
require 'faraday_middleware'

class NatureRemoExporter
  def initialize(registry, logger:, **config)
    @config = config.merge(
      api_endpoint: 'https://api.nature.global',
    ).freeze
    @logger = logger
    @conn = Faraday.new(url: @config.fetch(:api_endpoint)) do |conn|
      conn.authorization :Bearer, @config.fetch(:api_token)
      conn.headers['User-Agent'] = "natureremo_exporter (+https://github.com/hanazuki/natureremo_exporter) Faraday/#{Faraday::VERSION}"

      conn.response :json, content_type: 'application/json'
      conn.response :logger, @logger, log_level: :debug, headers: {response: true}

      conn.adapter :net_http
    end

    device_labels = %i[remoid name serial]
    appliance_labels = %i[id remoid name]

    @registry = registry
    @metrics = {
      sensor_temperature: @registry.gauge(:natureremo_sensor_temperature, docstring: 'Measured temperature', labels: device_labels),
      sensor_humidity: @registry.gauge(:natureremo_sensor_humidity, docstring: 'Measured humidity', labels: device_labels),
      sensor_illuminance: @registry.gauge(:natureremo_sensor_illuminance, docstring: 'Measured illuminance', labels: device_labels),
      sensor_motion: @registry.gauge(:natureremo_sensor_motion, docstring: 'Measured motion', labels: device_labels),
      sensor_offset_temperature: @registry.gauge(:natureremo_sensor_offset_temperature, docstring: 'Temperature offset', labels: device_labels),
      sensor_offset_humidity: @registry.gauge(:natureremo_sensor_offset_humidity, docstring: 'Humidity offset', labels: device_labels),
      ac_on: @registry.gauge(:natureremo_ac_on, docstring: 'Wheather air-conditioning is turned on', labels: appliance_labels),
      ac_mode: @registry.gauge(:natureremo_ac_mode, docstring: 'Air-conditioning mode setting', labels: appliance_labels + %i[mode]),
      ac_temperature: @registry.gauge(:natureremo_ac_temperature, docstring: 'Air-conditioning temperature setting', labels: appliance_labels + %i[mode unit]),
      light_on: @registry.gauge(:natureremo_light_on, docstring: 'Wheather light is turned on', labels: appliance_labels),
      light_brightness: @registry.gauge(:natureremo_light_brightness, docstring: 'Light brightness setting', labels: appliance_labels),
      smart_meter_fwd_energy: @registry.gauge(:natureremo_smart_meter_forward_energy_kilowatthours, docstring: 'Cumulative imported energy', labels: appliance_labels),
      smart_meter_bwd_energy: @registry.gauge(:natureremo_smart_meter_backward_energy_kilowatthours, docstring: 'Cumulative exported energy', labels: appliance_labels),
      smart_meter_power: @registry.gauge(:natureremo_smart_meter_instantaneous_power_watts, docstring: 'Instantaneous power', labels: appliance_labels),
    }
  end

  def update
    begin
      update_devices
    rescue => e
      @logger.warn(__method__) { e }
    end

    begin
      update_appliances
    rescue => e
      @logger.warn(__method__) { e }
    end
  end

  private

  def fetch_appliances
    resp = @conn.get('1/appliances')
    unless resp.success?
      fail "API Failure 1/appliances (#{resp.status})"
    end
    resp.body
  end

  def fetch_devices
    resp = @conn.get('1/devices')
    unless resp.success?
      fail "API Failure 1/devices (#{resp.status})"
    end
    resp.body
  end

  SENSORS = {
    'te' => :sensor_temperature,
    'hu' => :sensor_humidity,
    'il' => :sensor_illuminance,
    'mo' => :sensor_motion,
  }.freeze

  def update_devices
    devices = fetch_devices
    @logger.info(__method__) { "Found #{devices.size} devices" }

    devices.each do |device|
      @logger.debug(__method__) { "Device ID: #{device['id']}" }
      labels = {
        remoid: device['id'],
        name: device['name'],
        serial: device['serial_number'],
      }

      if offset = device['temperature_offset']
        @metrics[:sensor_offset_temperature].set(offset, labels: labels)
      end
      if offset = device['temperature_humidity']
        @metrics[:sensor_offset_humidity].set(offset, labels: labels)
      end

      if events = device['newest_events']
        SENSORS.each do |event_name, metric_key|
          if value = events.dig(event_name, 'val')
            @metrics[metric_key].set(value, labels: labels)
          end
        end
      end
    rescue => e
      @logger.error(__method__) { e }
    end
  end

  def update_appliances
    appliances = fetch_appliances
    @logger.info(__method__) { "Found #{appliances.size} appliances" }

    appliances.each do |appliance|
      @logger.debug(__method__) { "Appliance ID: #{appliance['id']}, Type: #{appliance['type']}" }
      labels = {
        id: appliance['id'],
        remoid: appliance.dig('device', 'id'),
        name: appliance['nickname'],
      }

      case appliance['type']
      when 'AC'
        update_ac(appliance, labels: labels)
      when 'LIGHT'
        update_light(appliance, labels: labels)
      when 'SMART_METER'
        update_smart_meter(appliance, labels: labels)
      end
    rescue => e
      @logger.error(__method__) { e }
    end
  end

  AC_MODES = %w[cool warm dry blow auto]

  def update_ac(appliance, labels:)
    settings = appliance.fetch('settings')
    aircon = appliance.fetch('aircon')

    button = settings['button']
    @metrics[:ac_on].set(button != 'power-off' ? 1 : 0, labels: labels)

    mode = settings['mode']
    AC_MODES.each do |m|
      @metrics[:ac_mode].set(mode == m ? 1 : 0, labels: labels.merge(mode: m))
    end

    temp = settings['temp']
    temp_unit = aircon['tempUnit']
    @metrics[:ac_temperature].set(temp.to_f, labels: labels.merge(mode: mode, unit: temp_unit))
  end

  def update_light(appliance, labels:)
    state = appliance.dig('light', 'state')

    power = state['power']
    unless power.empty?
      @metrics[:light_on].set(power == 'on' ? 1 : 0, labels: labels)
    end

    brightness = state['brightness']
    unless brightness.empty?
      @metrics[:light_brightness].set(brightness.to_f, labels: labels)
    end
  end

  def update_smart_meter(appliance, labels:)
    echonetlite_properties = appliance.dig('smart_meter', 'echonetlite_properties')
    props = ->(epc) { echonetlite_properties.find {|prop| prop['epc'] == epc }&.[]('val') }

    coeff = props[211]&.to_i || 1
    unit = sm_unit(props[225].to_i)

    if fwd_energy = props[224]
      @metrics[:smart_meter_fwd_energy].set(fwd_energy.to_i * coeff * unit, labels: labels)
    end
    if bwd_energy = props[227]
      @metrics[:smart_meter_bwd_energy].set(bwd_energy.to_i * coeff * unit, labels: labels)
    end

    if power = props[231]
      @metrics[:smart_meter_power].set(power.to_i, labels: labels)
    end
  end

  def sm_unit(val)
    if val < 0x0A
      10 ** -val
    else
      10 ** (val - 0x09)
    end
  end

end
