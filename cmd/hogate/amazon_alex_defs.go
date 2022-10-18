package main

// AlexaResponseEnvelope struct
type AlexaResponseEnvelope struct {
	Response *AlexaResponse `json:"response,omitempty"`
}

// AlexaResponse struct
type AlexaResponse struct {
	OutputSpeeech *AlexaOutputSpeeech `json:"outputSpeech,omitempty"`
}

// AlexaOutputSpeeech struct
type AlexaOutputSpeeech struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}
