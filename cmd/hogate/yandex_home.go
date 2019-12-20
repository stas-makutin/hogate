package main

import "net/http"

func addYandexHomeRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeYandexHomeHealth, http.HandlerFunc(yandexHomeHealth))
	handleDedicatedRoute(router, routeYandexHomeUnlink, authorizationMiddleware(scopeYandexHome)(http.HandlerFunc(yandexHomeUnlink)))
	handleDedicatedRoute(router, routeYandexHomeDevices, authorizationMiddleware(scopeYandexHome)(http.HandlerFunc(yandexHomeDevices)))
	handleDedicatedRoute(router, routeYandexHomeQuery, authorizationMiddleware(scopeYandexHome)(http.HandlerFunc(yandexHomeQuery)))
	handleDedicatedRoute(router, routeYandexHomeAction, authorizationMiddleware(scopeYandexHome)(http.HandlerFunc(yandexHomeAction)))
}

func yandexHomeHealth(w http.ResponseWriter, r *http.Request) {
	// expects HTTP/1.1 200 OK back
}

func yandexHomeUnlink(w http.ResponseWriter, r *http.Request) {
}

func yandexHomeDevices(w http.ResponseWriter, r *http.Request) {
}

func yandexHomeQuery(w http.ResponseWriter, r *http.Request) {
}

func yandexHomeAction(w http.ResponseWriter, r *http.Request) {
}
