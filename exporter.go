package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hanazuki/prometheus-natureremo-exporter/natureremo"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "natureremo"

var (
	deviceLabels = []string{"remoid", "name", "serial"}

	sensorTemperature = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "temperature"),
		"Measured temperature",
		deviceLabels, nil,
	)
	sensorHumidity = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "humidity"),
		"Measured humidity",
		deviceLabels, nil,
	)
	sensorIlluminance = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "illuminance"),
		"Measured illuminance",
		deviceLabels, nil,
	)
	sensorMotion = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "motion"),
		"Measured motion",
		deviceLabels, nil,
	)
	sensorOffsetTemperature = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor_offset", "temperature"),
		"Temperature offset setting",
		deviceLabels, nil,
	)
	sensorOffsetHumidity = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor_offset", "humidity"),
		"Humidity offset setting",
		deviceLabels, nil,
	)

	applianceLabels = []string{"id", "remoid", "name"}
	acOn            = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ac", "on"),
		"Wheather air-conditioning is turned on",
		applianceLabels, nil,
	)
	acMode = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ac", "mode"),
		"Air-conditioning mode setting",
		append([]string{"mode"}, applianceLabels...), nil,
	)
	acTemperatureC = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ac", "temperature_celsius"),
		"Air-conditioning temperature setting in degrees Celsius",
		append([]string{"mode"}, applianceLabels...), nil,
	)
	acTemperatureF = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "ac", "temperature_fahrenheit"),
		"Air-conditioning temperature setting in degrees Fahrenheit",
		append([]string{"mode"}, applianceLabels...), nil,
	)
	lightOn = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "light", "on"),
		"Wheather light is turned on",
		applianceLabels, nil,
	)
	lightBrightness = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "light", "brightness"),
		"Light brightness setting",
		applianceLabels, nil,
	)
	smartMeterFwdEnergy = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "smart_meter", "forward_energy_kilowatthours"),
		"Cumulative forward energy",
		applianceLabels, nil,
	)
	smartMeterBwdEnergy = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "smart_meter", "backward_energy_kilowatthours"),
		"Cumulative backward energy",
		applianceLabels, nil,
	)
	smartMeterInstantaneousPower = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "smart_meter", "instantaneous_power_watts"),
		"Measured instantaneous power",
		applianceLabels, nil,
	)
)

type Exporter struct {
	client natureremo.Client
	logger log.Logger
}

func NewExporter(client natureremo.Client, logger log.Logger) *Exporter {
	return &Exporter{
		client: client,
		logger: logger,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- sensorTemperature
	ch <- sensorHumidity
	ch <- sensorIlluminance
	ch <- sensorMotion
	ch <- sensorOffsetTemperature
	ch <- sensorOffsetHumidity
	ch <- acOn
	ch <- acMode
	ch <- acTemperatureC
	ch <- acTemperatureF
	ch <- lightOn
	ch <- lightBrightness
	ch <- smartMeterFwdEnergy
	ch <- smartMeterBwdEnergy
	ch <- smartMeterInstantaneousPower
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(2)
	go func() {
		defer wg.Done()
		e.collectDeviceMetrics(ch, ctx)
	}()
	go func() {
		defer wg.Done()
		e.collectApplianceMetrics(ch, ctx)
	}()
}

func (e *Exporter) collectDeviceMetrics(ch chan<- prometheus.Metric, ctx context.Context) {
	devices, err := e.client.FetchDevices(ctx)
	if err != nil {
		level.Warn(e.logger).Log("msg", err.Error())
		return
	}

	level.Info(e.logger).Log("msg", fmt.Sprintf("Fetched %d devices", len(devices)))

	for _, device := range devices {
		deviceMetricsEmitter{Device: device, logger: e.logger}.emit(ch)
	}
}

type deviceMetricsEmitter struct {
	natureremo.Device
	logger log.Logger
}

func (em deviceMetricsEmitter) labels() []string {
	return []string{
		em.Id,
		em.Name,
		em.SerialNumber,
	}
}

func (em deviceMetricsEmitter) emit(ch chan<- prometheus.Metric) {
	labels := em.labels()

	ch <- prometheus.MustNewConstMetric(sensorOffsetTemperature, prometheus.GaugeValue,
		em.TemperatureOffset, labels...)
	ch <- prometheus.MustNewConstMetric(sensorOffsetHumidity, prometheus.GaugeValue,
		em.HumidityOffset, labels...)

	if ev := em.Events.Temperature; ev != nil {
		ch <- prometheus.MustNewConstMetric(sensorTemperature, prometheus.GaugeValue, ev.Value, labels...)
	}
	if ev := em.Events.Humidity; ev != nil {
		ch <- prometheus.MustNewConstMetric(sensorHumidity, prometheus.GaugeValue, ev.Value, labels...)
	}
	if ev := em.Events.Illuminance; ev != nil {
		ch <- prometheus.MustNewConstMetric(sensorIlluminance, prometheus.GaugeValue, ev.Value, labels...)
	}
	if ev := em.Events.Motion; ev != nil {
		ch <- prometheus.MustNewConstMetric(sensorMotion, prometheus.GaugeValue, ev.Value, labels...)
	}
}

func (e *Exporter) collectApplianceMetrics(ch chan<- prometheus.Metric, ctx context.Context) {
	appliances, err := e.client.FetchAppliances(ctx)
	if err != nil {
		level.Warn(e.logger).Log("msg", err.Error())
	}

	level.Info(e.logger).Log("msg", fmt.Sprintf("Fetched %d appliances", len(appliances)))

	for _, appliance := range appliances {
		applianceMetricsEmitter{Appliance: appliance, logger: e.logger}.emit(ch)
	}
}

type applianceMetricsEmitter struct {
	natureremo.Appliance
	logger log.Logger
}

func (em applianceMetricsEmitter) labels() []string {
	return []string{
		em.Id,
		em.Device.Id,
		em.Nickname,
	}
}

func (em applianceMetricsEmitter) acLabels(mode string) []string {
	return append([]string{mode}, em.labels()...)
}

func (em applianceMetricsEmitter) emit(ch chan<- prometheus.Metric) {
	switch em.Type {
	case natureremo.APPLIANCE_AC:
		em.emitAcMetrics(ch)
	case natureremo.APPLIANCE_LIGHT:
		em.emitLightMetrics(ch)
	case natureremo.APPLIANCE_EL_SMART_METER:
		em.emitSmartMeterMetrics(ch)
	}
}

func (em applianceMetricsEmitter) emitAcMetrics(ch chan<- prometheus.Metric) {
	aircon := em.Aircon
	if aircon == nil {
		level.Warn(em.logger).Log("msg", fmt.Sprintf("aircon is null for appliance with type=AC"))
		return
	}

	settings := em.AirconSettings
	if settings == nil {
		level.Warn(em.logger).Log("msg", fmt.Sprintf("settings is null for appliance with type=AC"))
		return
	}

	on := 1.0
	if settings.Button == "power-off" {
		on = 0.0
	}
	ch <- prometheus.MustNewConstMetric(acOn, prometheus.GaugeValue, on, em.labels()...)

	mode := settings.Mode
	if mode != "" {
		ch <- prometheus.MustNewConstMetric(acMode, prometheus.GaugeValue, on, em.acLabels(mode)...)
	}

	switch aircon.TemperatureUnit {
	case "c":
		val, err := strconv.ParseFloat(settings.Temperature, 64)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(acTemperatureC, prometheus.GaugeValue,
				val, em.acLabels(mode)...)
		}
	case "f":
		val, err := strconv.ParseFloat(settings.Temperature, 64)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(acTemperatureF, prometheus.GaugeValue,
				val, em.acLabels(mode)...)
		}
	}
}

func (em applianceMetricsEmitter) emitLightMetrics(ch chan<- prometheus.Metric) {
	light := em.Light
	if light == nil {
		level.Warn(em.logger).Log("msg", fmt.Sprintf("light is null for appliance with type=LIGHT"))
		return
	}

	settings := light.Settings

	on := 0.0
	if settings.Power == "on" {
		on = 1.0
	}
	ch <- prometheus.MustNewConstMetric(lightOn, prometheus.GaugeValue, on, em.labels()...)

	brightness, err := strconv.ParseFloat(settings.Brightness, 64)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(lightBrightness, prometheus.GaugeValue, brightness, em.labels()...)
	}
}

func (em applianceMetricsEmitter) emitSmartMeterMetrics(ch chan<- prometheus.Metric) {
	sm := em.SmartMeter
	if sm == nil {
		level.Warn(em.logger).Log("msg", fmt.Sprintf("smart_meter is null for appliance with type=EL_SMART_METER"))
		return
	}

	val, err := sm.FindIntProperty(231)
	if err != nil {
		level.Warn(em.logger).Log("msg", err.Error())
	} else if val != nil {
		ch <- prometheus.MustNewConstMetric(smartMeterInstantaneousPower, prometheus.GaugeValue,
			float64(*val), em.labels()...)
	}

	coeff := 1
	unit := 1.0

	val, err = sm.FindIntProperty(211)
	if err != nil {
		level.Warn(em.logger).Log("msg", err.Error())
		return
	} else if val != nil {
		coeff = *val
	}

	val, err = sm.FindIntProperty(225)
	if err != nil {
		level.Warn(em.logger).Log("msg", err.Error())
		return
	} else if val != nil {
		unit = smUnit(*val)
	}

	val, err = sm.FindIntProperty(224)
	if err != nil {
		level.Warn(em.logger).Log("msg", err.Error())
	} else if val != nil {
		ch <- prometheus.MustNewConstMetric(smartMeterFwdEnergy, prometheus.GaugeValue,
			float64(*val)*float64(coeff)*unit, em.labels()...)
	}

	val, err = sm.FindIntProperty(227)
	if err != nil {
		level.Warn(em.logger).Log("msg", err.Error())
	} else if val != nil {
		ch <- prometheus.MustNewConstMetric(smartMeterBwdEnergy, prometheus.GaugeValue,
			float64(*val)*float64(coeff)*unit, em.labels()...)
	}
}

func smUnit(unit int) float64 {
	if unit < 0x0A {
		return math.Pow10(-unit)
	} else {
		return math.Pow10(unit - 0x09)
	}
}
