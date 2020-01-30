package main

import "net/http"

func addYandexDialogsRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeYandexDialogsTales, http.HandlerFunc(yandexDialogsTales))
}
