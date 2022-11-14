package main

import "net/http"

func addLoginRoute(router *http.ServeMux) {
	handleDedicatedRoute(router, routeLogin, http.HandlerFunc(login))
}

func login(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}
