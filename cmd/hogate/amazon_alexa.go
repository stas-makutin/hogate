package main

import (
	"encoding/json"
	"net/http"
)

func addAmazonAlexaRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeAmazonAlexaWhistles, authorizationHandler(scopeYandexHome)(http.HandlerFunc(amazonAlexaWhistles)))
}

func amazonAlexaWhistles(w http.ResponseWriter, r *http.Request) {
	request := acceptAlexaRequest(r)
	if request == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(AlexaResponseEnvelope{
		Version: "1.0",
		Response: &AlexaResponse{
			OutputSpeeech: &AlexaOutputSpeeech{
				Type: "PlainText",
				Text: "I shall elucidate",
			},
		},
	})
}
