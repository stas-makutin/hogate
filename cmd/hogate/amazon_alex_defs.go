package main

// AlexaRequestEnvelope struct
type AlexaRequestEnvelope struct {
	Version string        `json:"version,omitempty"`
	Session *AlexaSession `json:"session,omitempty"`
	Context *AlexaContext `json:"context,omitempty"`
	Request *AlexaRequest `json:"request,omitempty"`
}

// AlexaSession struct
type AlexaSession struct {
}

// AlexaContext struct
type AlexaContext struct {
}

// AlexaRequest struct
type AlexaRequest struct {
	Type      string `json:"type,omitempty"`
	RequestId string `json:"requestId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Locale    string `json:"locale,omitempty"`
}

// AlexaResponseEnvelope struct
type AlexaResponseEnvelope struct {
	Version  string         `json:"version,omitempty"`
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
