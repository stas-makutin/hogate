package main

import (
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

				switch cap.Parameters.(type) {
				case YandexHomeParametersOnOff:
					yhDevice.Capabilities = append(yhDevice.Capabilities, YandexHomeCapability{Type: yhDeviceCapOnOff, Retrievable: cap.Retrievable})
				case YandexHomeParametersModeConfig:
					yhc := YandexHomeCapability{
						Type:        yhDeviceCapMode,
						Retrievable: cap.Retrievable,
						Parameters: YandexHomeCapabilityMode{
							Instance: cap.Parameters.(YandexHomeParametersModeConfig).Instance,
						},
					}
					yhDevice.Capabilities = append(yhDevice.Capabilities, yhc)
				case YandexHomeParametersRangeConfig:
				default:
					capError("invalid or unsupported.")
				}

			}
		}
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
}

func yandexHomeQuery(w http.ResponseWriter, r *http.Request) {
}

func yandexHomeAction(w http.ResponseWriter, r *http.Request) {
}
