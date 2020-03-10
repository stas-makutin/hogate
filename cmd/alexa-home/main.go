package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

// temporary mapping

var devicesFriendlyNames = map[string]string{
	"L-1":   "Office Light",
	"L-2":   "Over the Table Light",
	"L-3":   "Over the Sink Light",
	"L-4":   "Island Light",
	"L-5":   "Entrance Light",
	"L-6":   "Garage Light",
	"L-7":   "Garbage Light",
	"L-6a":  "Gate Light",
	"L-8":   "Window Light",
	"L-9":   "Fireplace Light",
	"L-10":  "Upper Light",
	"L-11":  "Backyard Light",
	"L-12":  "Shed Light",
	"L-13":  "Driveway Light",
	"L-13a": "Frontyard Light",
	"L-13b": "Main Door Light",
	"L-14":  "Corridor Light",
	"L-15":  "Landry Light",
}

// Alexa Smart Home definitions

type InputEvent struct {
	Directive Directive `json:"directive"`
}

type Directive struct {
	Header   Header          `json:"header"`
	Payload  json.RawMessage `json:"payload"`
	Endpoint *Endpoint       `json:"endpoint"`
}

type Header struct {
	Namespace        string `json:"namespace"`
	Name             string `json:"name"`
	PayloadVersion   string `json:"payloadVersion"`
	MessageID        string `json:"messageId"`
	CorrelationToken string `json:"correlationToken"`
}

type Endpoint struct {
	Scope      Scope   `json:"scope"`
	EndpointID string  `json:"endpointId"`
	Cookie     *Cookie `json:"cookie,omitempty"`
}

type Scope struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type Cookie struct {
	Detail1 *string `json:"detail1,omitempty"`
	Detail2 *string `json:"detail2,omitempty"`
}

type Response struct {
	Event   Event       `json:"event"`
	Context interface{} `json:"context,omitempty"`
}

type Event struct {
	Header   Header      `json:"header,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
	Endpoint *Endpoint   `json:"endpoint,omitempty"`
}

type ErrorPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type DiscoveryPayload struct {
	Endpoints []DiscoveryEndpoint `json:"endpoints"`
}

type DiscoveryEndpoint struct {
	EndpointID        string       `json:"endpointId"`
	ManufacturerName  string       `json:"manufacturerName"`
	FriendlyName      string       `json:"friendlyName"`
	Description       string       `json:"description"`
	DisplayCategories []string     `json:"displayCategories"`
	Cookie            *Cookie      `json:"cookie,omitempty"`
	Capabilities      []Capability `json:"capabilities"`
}

type Capability struct {
	Type                       string                      `json:"type"`
	Interface                  string                      `json:"interface"`
	Version                    string                      `json:"version"`
	Properties                 *Properties                 `json:"properties,omitempty"`
	SupportsDeactivation       *bool                       `json:"supportsDeactivation,omitempty"`
	ProactivelyReported        *bool                       `json:"proactivelyReported,omitempty"`
	CameraStreamConfigurations []CameraStreamConfiguration `json:"cameraStreamConfigurations,omitempty"`
}

type CameraStreamConfiguration struct {
	Protocols          []string     `json:"protocols"`
	Resolutions        []Resolution `json:"resolutions"`
	AuthorizationTypes []string     `json:"authorizationTypes"`
	VideoCodecs        []string     `json:"videoCodecs"`
	AudioCodecs        []string     `json:"audioCodecs"`
}

type Resolution struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type Properties struct {
	Supported           []Supported `json:"supported"`
	ProactivelyReported bool        `json:"proactivelyReported"`
	Retrievable         bool        `json:"retrievable"`
}

type Supported struct {
	Name string `json:"name"`
}

// Home endpoint
var devicesUrl, actionUrl string

func init() {
	hostPrefix := os.Getenv("TARGET_HOST_URL_PREFIX")
	devicesUrl = hostPrefix + "/yandex/home/v1.0/user/devices"
	actionUrl = hostPrefix + "/yandex/home/v1.0/user/devices/action"
}

func internalError() (namespace, name string, payload, context interface{}) {
	return "Alexa", "ErrorResponse", ErrorPayload{"INTERNAL_ERROR", "Internal error"}, nil
}

func unavailableError() (namespace, name string, payload, context interface{}) {
	return "Alexa", "ErrorResponse", ErrorPayload{"BRIDGE_UNREACHABLE", "Unable to get list of devices"}, nil
}

func invalidAuthorizationError() (namespace, name string, payload, context interface{}) {
	return "Alexa", "ErrorResponse", ErrorPayload{"INVALID_AUTHORIZATION_CREDENTIAL", "OAuth2 token is not provided"}, nil
}

func expiredCredentialsError() (namespace, name string, payload, context interface{}) {
	return "Alexa", "ErrorResponse", ErrorPayload{"EXPIRED_AUTHORIZATION_CREDENTIAL", "OAuth2 token is expired"}, nil
}

func discovery(event InputEvent) (namespace, name string, payload, context interface{}) {
	if event.Directive.Payload == nil {
		return internalError()
	}
	var endpoint Endpoint
	if err := json.Unmarshal(event.Directive.Payload, &endpoint); err != nil {
		return internalError()
	}
	if endpoint.Scope.Token == "" {
		return invalidAuthorizationError()
	}
	client := http.Client{Timeout: time.Second * 2}
	request, error := http.NewRequest("GET", devicesUrl, nil)
	if error != nil {
		return unavailableError()
	}
	request.Header.Add("Authorization", "Bearer "+endpoint.Scope.Token)
	response, error := client.Do(request)
	if error != nil {
		return unavailableError()
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 403 {
			return expiredCredentialsError()
		}
		return unavailableError()
	}

	var data map[string]interface{}
	if error := json.NewDecoder(response.Body).Decode(&data); error != nil {
		return unavailableError()
	}
	pi, ok := data["payload"]
	if !ok {
		return unavailableError()
	}
	p, ok := pi.(map[string]interface{})
	if !ok {
		return unavailableError()
	}
	di, ok := p["devices"]
	if !ok {
		return unavailableError()
	}
	devices, ok := di.([]interface{})
	if !ok {
		return unavailableError()
	}

	pl := DiscoveryPayload{}
	for _, vi := range devices {
		v, ok := vi.(map[string]interface{})
		if !ok {
			return unavailableError()
		}

		deviceId, _ := v["id"].(string)
		deviceType, _ := v["type"].(string)

		if deviceId == "" || !(deviceType == "devices.types.light" || deviceType == "devices.types.switch") {
			continue
		}
		friendlyName, ok := devicesFriendlyNames[deviceId]
		if !ok {
			continue
		}

		pl.Endpoints = append(pl.Endpoints, DiscoveryEndpoint{
			EndpointID:        deviceId,
			ManufacturerName:  "DIY",
			Description:       "DIY Switch",
			FriendlyName:      friendlyName,
			DisplayCategories: []string{"SWITCH", "LIGHT"},
			Capabilities: []Capability{
				Capability{
					Type:      "AlexaInterface",
					Interface: "Alexa.PowerController",
					Version:   "3",
				},
				Capability{
					Type:      "AlexaInterface",
					Interface: "Alexa",
					Version:   "3",
				},
			},
		})
	}

	return "Alexa.Discovery", "Discover.Response", pl, nil
}

func powerController(event InputEvent) (namespace, name string, payload, context interface{}) {
	if event.Directive.Endpoint == nil {
		return internalError()
	}
	if event.Directive.Endpoint.Scope.Token == "" {
		return invalidAuthorizationError()
	}
	action := event.Directive.Header.Name == "TurnOn"
	client := http.Client{Timeout: time.Second * 3}
	request, error := http.NewRequest("POST", actionUrl, bytes.NewBuffer([]byte(
		fmt.Sprintf(
			`{"payload":{"devices":[{"id":"%v","capabilities":[{"type":"devices.capabilities.on_off","state":{"instance":"on","value":%t}}]}]}}`,
			event.Directive.Endpoint.EndpointID,
			action,
		),
	)))
	if error != nil {
		return unavailableError()
	}
	request.Header.Add("Authorization", "Bearer "+event.Directive.Endpoint.Scope.Token)
	request.Header.Set("Content-Type", "application/json")

	response, error := client.Do(request)
	if error != nil {
		return unavailableError()
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 403 {
			return expiredCredentialsError()
		}
		return unavailableError()
	}

	value := "OFF"
	if action {
		value = "ON"
	}

	return "Alexa", "Response", struct{}{}, map[string]interface{}{
		"properties": []map[string]interface{}{map[string]interface{}{
			"namespace":                 "Alexa.PowerController",
			"name":                      "powerState",
			"value":                     value,
			"timeOfSample":              time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
			"uncertaintyInMilliseconds": 1000,
		}},
	}
}

func QueryHandler(ctx context.Context, event InputEvent) (*Response, error) {
	var namespace, name string
	var payload, context interface{}

	/*
		if ev, err := json.Marshal(&event); err == nil {
			fmt.Println(string(ev))
		}
	*/

	switch event.Directive.Header.Namespace {
	case "Alexa.Discovery":
		namespace, name, payload, context = discovery(event)
	case "Alexa.PowerController":
		namespace, name, payload, context = powerController(event)
	default:
		namespace = "Alexa"
		name = "ErrorResponse"
		payload = ErrorPayload{"INVALID_DIRECTIVE", "Directive is not supported"}
	}

	return &Response{
		Event: Event{
			Header: Header{
				Namespace:        namespace,
				Name:             name,
				CorrelationToken: event.Directive.Header.CorrelationToken,
				PayloadVersion:   "3",
				MessageID:        uuid.New().String(),
			},
			Payload:  payload,
			Endpoint: event.Directive.Endpoint,
		},
		Context: context,
	}, nil
}

func main() {
	lambda.Start(QueryHandler)
}
