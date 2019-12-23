package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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

	devices := make([]YandexHomeDevice, 0, len(yxhDevices))
	for _, v := range yxhDevices {
		devices = append(devices, v.yandex())
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
	var req YandexHomeQueryRequest
	if !parseJsonRequest(&req, w, r) {
		return
	}

	devices := make([]YandexHomeDeviceState, 0, len(req.Devices))
	for _, v := range req.Devices {
		if di, ok := yxhDevices[v.Id]; ok {
			devices = append(devices, di.query())
		} else {
			devices = append(devices, YandexHomeDeviceState{Id: v.Id, ErrorCode: yhDeviceErrorNotFound})
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(YandexHomeResponse{
		RequestId: r.Header.Get("X-Request-Id"),
		Payload: YandexHomeDevicesState{
			Devices: devices,
		},
	})
}

func yandexHomeAction(w http.ResponseWriter, r *http.Request) {
	var req YandexHomeActionRequest
	if !parseJsonRequest(&req, w, r) {
		return
	}

	devices := make([]YandexHomeDeviceActionResult, 0, len(req.Payload.Devices))
	for _, v := range req.Payload.Devices {
		if di, ok := yxhDevices[v.Id]; ok {
			devices = append(devices, di.action(v.Capabilities))
		} else {
			devices = append(devices, YandexHomeDeviceActionResult{
				Id: v.Id,
				ActionResult: YandexHomeActionResult{
					Status:    yhDeviceStatusError,
					ErrorCode: yhDeviceErrorNotFound,
				},
			})
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(YandexHomeResponse{
		RequestId: r.Header.Get("X-Request-Id"),
		Payload: YandexHomeDevicesActionResult{
			Devices: devices,
		},
	})
}
