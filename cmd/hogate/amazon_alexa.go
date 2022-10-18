package main

import (
	"encoding/json"
	"net/http"
)

func addAmazonAlexaRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeAmazonAlexaWhistles, http.HandlerFunc(amazonAlexaWhistles))
}

func amazonAlexaWhistles(w http.ResponseWriter, r *http.Request) {
	if !validateAlexaRequest(r) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(AlexaResponseEnvelope{
		Response: &AlexaResponse{
			OutputSpeeech: &AlexaOutputSpeeech{
				Type: "PlainText",
				Text: "I shall elucidate",
			},
		},
	})
}
