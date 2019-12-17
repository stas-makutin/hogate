package main

import "net/http"

func addOAuthRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeOAuthAuthorize, http.HandlerFunc(oauthAuthorize))
	handleDedicatedRoute(router, routeOAuthToken, http.HandlerFunc(oauthToken))
}

func oauthAuthorize(w http.ResponseWriter, r *http.Request) {
}

func oauthToken(w http.ResponseWriter, r *http.Request) {
}
