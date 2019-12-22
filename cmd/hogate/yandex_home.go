package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var yhDevices map[string]YandexHomeDevice

func validateYandexHomeConfig(cfgError configError) {
	yhDevices = make(map[string]YandexHomeDevice)
	if config.YandexHome == nil {
		return
	}

	for i, device := range config.YandexHome.Devices {
		deviceError := func(msg string) {
			cfgError(fmt.Sprintf("yandexHome.devices, device %v: %v", i, msg))
		}

		var yhDevice YandexHomeDevice
		var ok bool

		yhDevice.Id = device.Id
		if device.Id == "" {
			deviceError("id cannot be empty.")
		}
		if _, ok = yhDevices[device.Id]; ok {
			deviceError(fmt.Sprintf("id '%v' is in use already.", device.Id))
		}
		yhDevice.Name = device.Name
		yhDevice.Description = device.Description
		yhDevice.Room = device.Room
		if yhDevice.Type, ok = parseDeviceType(device.Type); !ok {
			deviceError(fmt.Sprintf("invalid type '%v'.", device.Type))
		}
		yhDevice.CustomData = YandexHomeZwData{device.ZwId}
		if device.ZwId == 0 {
			deviceError("zwid is required and cannot be 0.")
		}
		if len(device.Capabilities) <= 0 {
			yhDevice.Capabilities = []YandexHomeCapability{
				{Type: yhDeviceCapOnOff, Retrievable: true},
			}
		} else {
			for j, cap := range device.Capabilities {
				capError := func(msg string) {
					deviceError(fmt.Sprintf("capabilities, capability %v: %v", j, msg))
				}

				yhc := YandexHomeCapability{}
				yhc.Retrievable = cap.Retrievable

				switch cap.Parameters.(type) {
				case YandexHomeParametersOnOff:
					yhc.Type = yhDeviceCapOnOff
				case YandexHomeParametersModeConfig:
					p, err := parseCapabilityMode(cap.Parameters.(YandexHomeParametersModeConfig))
					if err != nil {
						capError(err.Error())
					}
					yhc.Type = yhDeviceCapMode
					yhc.Parameters = *p
				case YandexHomeParametersRangeConfig:
					p, err := parseCapabilityRange(cap.Parameters.(YandexHomeParametersRangeConfig))
					if err != nil {
						capError(err.Error())
					}
					yhc.Type = yhDeviceCapRange
					yhc.Parameters = *p
				default:
					capError("invalid or unsupported.")
				}
				yhDevice.Capabilities = append(yhDevice.Capabilities, yhc)
			}
		}

		yhDevices[yhDevice.Id] = yhDevice
	}
}

func parseDeviceType(t string) (string, bool) {
	switch strings.ToLower(t) {
	case "light":
		return yhDeviceTypeLight, true
	case "socket":
		return yhDeviceTypeSwitch, true
	case "switch":
		return yhDeviceTypeSwitch, true
	case "thermostat":
		return yhDeviceTypeThermostat, true
	case "thermostat-ac":
		return yhDeviceTypeThermostatAC, true
	}
	return "", false
}

func parseCapabilityMode(m YandexHomeParametersModeConfig) (*YandexHomeCapabilityMode, error) {
	rv := YandexHomeCapabilityMode{
		Instance: strings.ToLower(m.Instance),
	}

	var values *map[string]struct{}
	switch rv.Instance {
	case yhCapModeInstanceThermostat:
		values = &map[string]struct{}{
			yhModeThermostatAuto:    struct{}{},
			yhModeThermostatCool:    struct{}{},
			yhModeThermostatDry:     struct{}{},
			yhModeThermostatEco:     struct{}{},
			yhModeThermostatFanOnly: struct{}{},
			yhModeThermostatHeat:    struct{}{},
		}
	case yhCapModeInstanceFanSpeed:
		values = &map[string]struct{}{
			yhModeFanSpeedAuto:   struct{}{},
			yhModeFanSpeedHigh:   struct{}{},
			yhModeFanSpeedLow:    struct{}{},
			yhModeFanSpeedMedium: struct{}{},
			yhModeFanSpeedQuiet:  struct{}{},
			yhModeFanSpeedTurbo:  struct{}{},
		}
	default:
		return nil, fmt.Errorf("unknown or not supported instance '%v'", m.Instance)
	}

	for _, v := range m.Values {
		lv := strings.ToLower(v)
		if _, ok := (*values)[lv]; !ok {
			return nil, fmt.Errorf("unknown mode value '%v'", v)
		}
		rv.Modes = append(rv.Modes, YandexHomeCapabilityModeValue{Value: lv})
	}

	return &rv, nil
}

func parseCapabilityRange(r YandexHomeParametersRangeConfig) (*YandexHomeCapabilityRange, error) {
	rv := YandexHomeCapabilityRange{
		Instance: strings.ToLower(r.Instance),
	}
	rv.RandomAccess = r.RandomAccess == nil || *r.RandomAccess
	rv.Range.Min = r.Min
	rv.Range.Max = r.Max
	rv.Range.Precision = r.Precision

	switch rv.Instance {
	case yhCapRangeInstanceTemperature:
		switch strings.ToLower(r.Unit) {
		case "celsius":
			rv.Unit = yhRangeTemperatureUnitCelsius
		case "kelvin":
			rv.Unit = yhRangeTemperatureUnitKelvin
		default:
			return nil, fmt.Errorf("Invalid unit '%v'.", r.Unit)
		}

	default:
		return nil, fmt.Errorf("unknown or not supported instance '%v'", r.Instance)
	}

	return &rv, nil
}

func addYandexHomeRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeYandexHomeHealth, http.HandlerFunc(yandexHomeHealth))
	handleDedicatedRoute(router, routeYandexHomeUnlink, authorizationHandler(scopeYandexHome)(http.HandlerFunc(yandexHomeUnlink)))
	handleDedicatedRoute(router, routeYandexHomeDevices, authorizationHandler(scopeYandexHome)(http.HandlerFunc(yandexHomeDevices)))
	handleDedicatedRoute(router, routeYandexHomeQuery, authorizationHandler(scopeYandexHome)(http.HandlerFunc(yandexHomeQuery)))
	handleDedicatedRoute(router, routeYandexHomeAction, authorizationHandler(scopeYandexHome)(http.HandlerFunc(yandexHomeAction)))
}

func yandexHomeHealth(w http.ResponseWriter, r *http.Request) {
	// expects HTTP/1.1 200 OK back
}

func yandexHomeUnlink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"request_id":"%v"}`, jsonEscape(r.Header.Get("X-Request-Id")))
}

func yandexHomeDevices(w http.ResponseWriter, r *http.Request) {
	claim := httpAuthorization(r)

	devices := make([]YandexHomeDevice, 0, len(yhDevices))
	for _, v := range yhDevices {
		devices = append(devices, v)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(YandexHomeResponse{
		RequestId: r.Header.Get("X-Request-Id"),
		Payload: YandexHomeDevices{
			UserId:  claim.UserName,
			Devices: devices,
		},
	})
}

func yandexHomeQuery(w http.ResponseWriter, r *http.Request) {
}

func yandexHomeAction(w http.ResponseWriter, r *http.Request) {
}
