package main

import (
	"fmt"
	"strings"
)

type yxhDeviceType int

type yxhCapabilityType int

type yxhInstanceType int

type yxhUnitsType int

const (
	yxhDeviceTypeLight = yxhDeviceType(iota)
	yxhDeviceTypeSocket
	yxhDeviceTypeSwitch
	yxhDeviceTypeThermostatAC
)

const (
	yxhCapabilityOnOff = yxhCapabilityType(iota)
	yxhCapabilityMode
	yxhCapabilityRange
)

const (
	yxhModeThermostat = yxhInstanceType(iota)
	yxhModeFanSpeed
)

const (
	yxhModeThermostatAuto = byte(iota)
	yxhModeThermostatCool
	yxhModeThermostatDry
	yxhModeThermostatEco
	yxhModeThermostatFanOnly
	yxhModeThermostatHeat
)

const (
	yxhModeFanSpeedAuto = byte(iota)
	yxhModeFanSpeedHigh
	yxhModeFanSpeedLow
	yxhModeFanSpeedMedium
	yxhModeFanSpeedQuiet
	yxhModeFanSpeedTurbo
)

const (
	yxhRangeTemperature = yxhInstanceType(iota)
)

const (
	yxhUnitsCelsius = yxhUnitsType(iota)
	yxhUnitsKelvin
)

type yxhDevice struct {
	id           string
	name         string
	description  string
	room         string
	devType      yxhDeviceType
	zwID         byte
	capabilities []yxhCapability
}

type yxhCapability struct {
	capType     yxhCapabilityType
	retrievable bool
	parameters  interface{}
}

type yxhParamMode struct {
	instance yxhInstanceType
	values   []byte
}

type yxhParamRange struct {
	instance     yxhInstanceType
	randomAccess bool
	units        yxhUnitsType
	min          float64
	max          float64
	precision    float64
}

var yxhDevices map[string]yxhDevice

func validateYandexHomeConfig(cfgError configError) {
	yxhDevices = make(map[string]yxhDevice)
	if config.YandexHome == nil {
		return
	}

	for i, d := range config.YandexHome.Devices {
		deviceError := func(msg string) {
			cfgError(fmt.Sprintf("yandexHome.devices, device %v: %v", i, msg))
		}

		device := parseDevice(d, deviceError)
		if device.id != "" {
			if _, ok := yxhDevices[device.id]; ok {
				deviceError(fmt.Sprintf("id '%v' is in use already.", d.ID))
			}
			yxhDevices[device.id] = device
		}
	}
}

func parseDevice(d YandexHomeDeviceConfig, cfgError configError) yxhDevice {
	var rv yxhDevice
	var ok bool
	if d.ID == "" {
		cfgError("id cannot be empty.")
	}
	rv.id = d.ID
	rv.name = d.Name
	rv.description = d.Description
	rv.room = d.Room
	if rv.devType, ok = parseDeviceType(d.Type); !ok {
		cfgError(fmt.Sprintf("invalid type '%v'.", d.Type))
	}
	if d.ZwID == 0 {
		cfgError("zwid is required and cannot be 0.")
	}
	rv.zwID = d.ZwID
	if len(d.Capabilities) <= 0 {
		rv.capabilities = append(rv.capabilities, yxhCapability{capType: yxhCapabilityOnOff, retrievable: true})
	} else {
		for i, cap := range d.Capabilities {
			capError := func(msg string) {
				cfgError(fmt.Sprintf("capabilities, capability %v: %v", i, msg))
			}
			c := yxhCapability{retrievable: cap.Retrievable}
			switch cap.Parameters.(type) {
			case YandexHomeParametersOnOff:
				c.capType = yxhCapabilityOnOff
			case YandexHomeParametersModeConfig:
				c.capType = yxhCapabilityMode
				c.parameters = parseCapabilityMode(cap.Parameters.(YandexHomeParametersModeConfig), capError)
			case YandexHomeParametersRangeConfig:
				c.capType = yxhCapabilityRange
				c.parameters = parseCapabilityRange(cap.Parameters.(YandexHomeParametersRangeConfig), capError)
			default:
				cfgError("invalid or unsupported.")
			}
			rv.capabilities = append(rv.capabilities, c)
		}
	}
	return rv
}

func parseDeviceType(t string) (yxhDeviceType, bool) {
	switch strings.ToLower(t) {
	case "light":
		return yxhDeviceTypeLight, true
	case "socket":
		return yxhDeviceTypeSocket, true
	case "switch":
		return yxhDeviceTypeSwitch, true
	case "thermostat-ac":
		return yxhDeviceTypeThermostatAC, true
	}
	return 0, false
}

func parseCapabilityMode(m YandexHomeParametersModeConfig, cfgError configError) yxhParamMode {
	rv := yxhParamMode{}

	var values *map[string]byte
	switch strings.ToLower(m.Instance) {
	case yhCapModeInstanceThermostat:
		rv.instance = yxhModeThermostat
		values = &map[string]byte{
			yhModeThermostatAuto:    yxhModeThermostatAuto,
			yhModeThermostatCool:    yxhModeThermostatCool,
			yhModeThermostatDry:     yxhModeThermostatDry,
			yhModeThermostatEco:     yxhModeThermostatEco,
			yhModeThermostatFanOnly: yxhModeThermostatFanOnly,
			yhModeThermostatHeat:    yxhModeThermostatHeat,
		}
	case yhCapModeInstanceFanSpeed:
		rv.instance = yxhModeFanSpeed
		values = &map[string]byte{
			yhModeFanSpeedAuto:   yxhModeFanSpeedAuto,
			yhModeFanSpeedHigh:   yxhModeFanSpeedHigh,
			yhModeFanSpeedLow:    yxhModeFanSpeedLow,
			yhModeFanSpeedMedium: yxhModeFanSpeedMedium,
			yhModeFanSpeedQuiet:  yxhModeFanSpeedQuiet,
			yhModeFanSpeedTurbo:  yxhModeFanSpeedTurbo,
		}
	default:
		cfgError(fmt.Sprintf("unknown or not supported instance '%v'", m.Instance))
	}

	if values != nil {
		for _, v := range m.Values {
			lv := strings.ToLower(v)
			if m, ok := (*values)[lv]; ok {
				rv.values = append(rv.values, m)
			} else {
				cfgError(fmt.Sprintf("unknown mode value '%v'", v))
			}
		}
	}

	return rv
}

func parseCapabilityRange(r YandexHomeParametersRangeConfig, cfgError configError) yxhParamRange {
	rv := yxhParamRange{
		randomAccess: r.RandomAccess == nil || *r.RandomAccess,
		min:          r.Min,
		max:          r.Max,
		precision:    r.Precision,
	}

	switch strings.ToLower(r.Instance) {
	case yhCapRangeInstanceTemperature:
		rv.instance = yxhRangeTemperature
		switch strings.ToLower(r.Units) {
		case "", "celsius":
			rv.units = yxhUnitsCelsius
		case "kelvin":
			rv.units = yxhUnitsKelvin
		default:
			cfgError(fmt.Sprintf("Invalid units '%v'.", r.Units))
		}

	default:
		cfgError(fmt.Sprintf("unknown or not supported instance '%v'", r.Instance))
	}

	return rv
}

func (d yxhDevice) yandex() YandexHomeDevice {
	devType := yhDeviceTypeLight
	switch d.devType {
	case yxhDeviceTypeSocket:
		devType = yhDeviceTypeSocket
	case yxhDeviceTypeSwitch:
		devType = yhDeviceTypeSwitch
	case yxhDeviceTypeThermostatAC:
		devType = yhDeviceTypeThermostatAC
	}

	var capabilities []YandexHomeCapability
	for _, v := range d.capabilities {
		capabilities = append(capabilities, v.yandex())
	}

	rv := YandexHomeDevice{
		ID:           d.id,
		Name:         d.name,
		Description:  d.description,
		Room:         d.room,
		Type:         devType,
		Capabilities: capabilities,
	}
	return rv
}

func (c yxhCapability) yandex() YandexHomeCapability {
	rv := YandexHomeCapability{
		Type:        yhDeviceCapOnOff,
		Retrievable: c.retrievable,
	}
	switch c.capType {
	case yxhCapabilityMode:
		if p, ok := c.parameters.(yxhParamMode); ok {
			rv.Type = yhDeviceCapMode
			rv.Parameters = p.yandex()
		}
	case yxhCapabilityRange:
		if p, ok := c.parameters.(yxhParamRange); ok {
			rv.Type = yhDeviceCapRange
			rv.Parameters = p.yandex()
		}
	}
	return rv
}

func (c yxhParamMode) yandex() YandexHomeCapabilityMode {
	rv := YandexHomeCapabilityMode{}

	rv.Instance = yhCapModeInstanceThermostat
	switch c.instance {
	case yxhModeThermostat:
		for _, m := range c.values {
			mode := yhModeThermostatAuto
			switch m {
			case yxhModeThermostatCool:
				mode = yhModeThermostatCool
			case yxhModeThermostatDry:
				mode = yhModeThermostatDry
			case yxhModeThermostatEco:
				mode = yhModeThermostatEco
			case yxhModeThermostatFanOnly:
				mode = yhModeThermostatFanOnly
			case yxhModeThermostatHeat:
				mode = yhModeThermostatHeat
			}
			rv.Modes = append(rv.Modes, YandexHomeCapabilityModeValue{Value: mode})
		}
	case yxhModeFanSpeed:
		rv.Instance = yhCapModeInstanceFanSpeed
		for _, m := range c.values {
			mode := yhModeFanSpeedAuto
			switch m {
			case yxhModeFanSpeedHigh:
				mode = yhModeFanSpeedHigh
			case yxhModeFanSpeedLow:
				mode = yhModeFanSpeedLow
			case yxhModeFanSpeedMedium:
				mode = yhModeFanSpeedMedium
			case yxhModeFanSpeedQuiet:
				mode = yhModeFanSpeedQuiet
			case yxhModeFanSpeedTurbo:
				mode = yhModeFanSpeedTurbo
			}
			rv.Modes = append(rv.Modes, YandexHomeCapabilityModeValue{Value: mode})
		}
	}

	return rv
}

func (c yxhParamRange) yandex() YandexHomeCapabilityRange {
	rv := YandexHomeCapabilityRange{
		Instance:     yhCapRangeInstanceTemperature,
		RandomAccess: c.randomAccess,
		Range: YandexHomeCapabilityRangeValue{
			Min:       c.min,
			Max:       c.max,
			Precision: c.precision,
		},
	}

	switch c.instance {
	case yxhRangeTemperature:
		rv.Unit = yhRangeTemperatureUnitCelsius
		switch c.units {
		case yxhUnitsKelvin:
			rv.Unit = yhRangeTemperatureUnitKelvin
		}
	}

	return rv
}

func (d yxhDevice) query() (rv YandexHomeDeviceState) {
	rv.ID = d.id

	for _, c := range d.capabilities {
		if !c.retrievable {
			continue
		}

		var capState *YandexHomeCapabilityState = nil
		errorCode := ""

		switch c.capType {
		case yxhCapabilityOnOff:
			switch d.devType {
			case yxhDeviceTypeLight, yxhDeviceTypeSocket, yxhDeviceTypeSwitch:
				capState, errorCode = yxhQueryBasicOnOff(d.zwID)
			}
		}

		if errorCode != "" || capState == nil {
			if errorCode == "" {
				rv.ErrorCode = yhDeviceErrorInternal
			} else {
				rv.ErrorCode = errorCode
			}
			rv.Capabilities = []YandexHomeCapabilityState{}
			break
		} else {
			rv.Capabilities = append(rv.Capabilities, *capState)
		}
	}

	return
}

func (d yxhDevice) action(capabilities []YandexHomeCapabilityAction) (rv YandexHomeDeviceActionResult) {
	rv.ID = d.id

	for _, cap := range capabilities {
		c := YandexHomeCapabilityActionResult{
			Type: cap.Type,
			State: YandexHomeInstanceResult{
				Instance: cap.State.Instance,
				ActionResult: YandexHomeActionResult{
					Status:    yhDeviceStatusError,
					ErrorCode: yhDeviceErrorInvalidAction,
				},
			},
		}

		if capType, instance, ok := parseActionCapability(cap.Type, cap.State.Instance); ok && d.validate(capType, instance) {
			switch capType {
			case yxhCapabilityOnOff:
				if value, ok := cap.State.Value.(bool); ok {
					c.State.ActionResult.ErrorCode = yxhActionBasicOnOff(d.zwID, value)
				}
			}
		}

		if c.State.ActionResult.ErrorCode == "" {
			c.State.ActionResult.Status = yhDeviceStatusDone
		}

		rv.Capabilities = append(rv.Capabilities, c)
	}

	return
}

func (d yxhDevice) validate(t yxhCapabilityType, i yxhInstanceType) bool {
	for _, c := range d.capabilities {
		if c.capType == t {
			switch t {
			case yxhCapabilityOnOff:
				return true
			case yxhCapabilityMode:
				if p, ok := c.parameters.(yxhParamMode); ok && p.instance == i {
					return true
				}
			case yxhCapabilityRange:
				if p, ok := c.parameters.(yxhParamRange); ok && p.instance == i {
					return true
				}
			}
		}
	}
	return false
}

func parseActionCapability(t string, i string) (yxhCapabilityType, yxhInstanceType, bool) {
	switch t {
	case yhDeviceCapOnOff:
		if i == "on" {
			return yxhCapabilityOnOff, 0, true
		}
	case yhDeviceCapMode:
		return yxhCapabilityMode, 0, true
	case yhDeviceCapRange:
		return yxhCapabilityRange, 0, true
	}
	return 0, 0, false
}
