package natureremo

import (
	"fmt"
	"strconv"
)

type DeviceCore struct {
	Name         string `json:"name"`
	Id           string `json:"id"`
	SerialNumber string `json:"serial_number"`

	TemperatureOffset float64 `json:"temperature_offset"`
	HumidityOffset    float64 `json:"humidity_offset"`
}

type Device struct {
	DeviceCore
	Events SensorValues `json:"newest_events"`
}

type SensorValues struct {
	Temperature *SensorValue `json:"te"`
	Humidity    *SensorValue `json:"hu"`
	Illuminance *SensorValue `json:"il"`
	Motion      *SensorValue `json:"mo"`
}

type SensorValue struct {
	Value float64 `json:"val"`
}

type Appliance struct {
	Id             string          `json:"id"`
	Device         DeviceCore      `json:"device"`
	Type           string          `json:"type"`
	Nickname       string          `json:"nickname"`
	AirconSettings *AirconSettings `json:"settings"`
	Aircon         *Aircon         `json:"aircon"`
	Light          *Light          `json:"light"`
	SmartMeter     *SmartMeter     `json:"smart_meter"`
}

const (
	APPLIANCE_IR             = "IR"
	APPLIANCE_AC             = "AC"
	APPLIANCE_TV             = "TV"
	APPLIANCE_LIGHT          = "LIGHT"
	APPLIANCE_EL_SMART_METER = "EL_SMART_METER"
)

type AirconSettings struct {
	Temperature string `json:"temp"`
	Mode        string `json:"mode"`
	Volume      string `json:"vol"`
	Direction   string `json:"dir"`
	Button      string `json:"button"`
}

type Aircon struct {
	TemperatureUnit string `json:"tempUnit"`
}

type Light struct {
	Settings LightSettings `json:"state"`
}

type LightSettings struct {
	Brightness string `json:"brightness"`
	Power      string `json:"power"`
}

type SmartMeter struct {
	Properties []ELProperty `json:"echonetlite_properties"`
}

type ELProperty struct {
	EPC   int    `json:"epc"`
	Value string `json:"val"`
}

func (sm *SmartMeter) FindProperty(epc int) *string {
	for _, property := range sm.Properties {
		if property.EPC == epc {
			return &property.Value
		}
	}
	return nil
}

func (sm *SmartMeter) FindIntProperty(epc int) (*int, error) {
	raw_val := sm.FindProperty(epc)
	if raw_val == nil {
		return nil, nil
	}

	val, err := strconv.Atoi(*raw_val)
	if err != nil {
		return nil, fmt.Errorf("Unparseable value `%s' for EPC %d: %s", *raw_val, epc, err.Error())
	}

	return &val, nil
}
