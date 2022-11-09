package main

import "encoding/json"

// AlexaRequestEnvelope struct
type AlexaRequestEnvelope struct {
	Version string        `json:"version,omitempty"`
	Session *AlexaSession `json:"session,omitempty"`
	Context *AlexaContext `json:"context,omitempty"`
	Request AlexaRequest  `json:"request,omitempty"`
}

func (c *AlexaRequestEnvelope) UnmarshalJSON(data []byte) error {
	var env struct {
		Version string          `json:"version,omitempty"`
		Session *AlexaSession   `json:"session,omitempty"`
		Context *AlexaContext   `json:"context,omitempty"`
		Request json.RawMessage `json:"request,omitempty"`
	}
	err := json.Unmarshal(data, &env)
	if err != nil {
		return err
	}

	var baseRequest *AlexaBaseRequest
	if err := json.Unmarshal(data, &baseRequest); err != nil {
		return err
	}

	switch c.Request.Type() {
	case "LaunchRequest":
		c.Request = &AlexaLaunchRequest{AlexaBaseRequest: *baseRequest}
	case "CanFulfillIntentRequest":
		var canFulfillIntentRequest *AlexaCanFulfillIntentRequest
		if err := json.Unmarshal(data, &canFulfillIntentRequest); err != nil {
			return err
		}
		c.Request = canFulfillIntentRequest
	case "IntentRequest":
		var intentRequest *AlexaIntentRequest
		if err := json.Unmarshal(data, &intentRequest); err != nil {
			return err
		}
		c.Request = intentRequest
	case "SessionEndedRequest":
		var sessionEndedRequest *AlexaSessionEndedRequest
		if err := json.Unmarshal(data, &sessionEndedRequest); err != nil {
			return err
		}
		c.Request = sessionEndedRequest
	default:
		c.Request = baseRequest
	}

	c.Version = env.Version
	c.Session = env.Session
	c.Context = env.Context
	return nil
}

// AlexaSession struct
type AlexaSession struct {
	New         bool              `json:"new"`
	SessionId   string            `json:"sessionId"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Application *AlexaApplication `json:"application,omitempty"`
	User        *AlexaUser        `json:"user,omitempty"`
}

// AlexaContext struct
type AlexaContext struct {
	AlexaPresentationAPL *AlexaPresentationAPL `json:"Alexa.Presentation.APL,omitempty"`
	AudioPlayer          *AlexaAudioPlayer     `json:"AudioPlayer,omitempty"`
	System               *AlexaSystem          `json:"System,omitempty"`
	Viewport             *AlexaViewport        `json:"Viewport,omitempty"`
	Viewports            []*AlexaViewportInfo  `json:"Viewports,omitempty"`
}

type AlexaApplication struct {
	ApplicationId string `json:"applicationId"`
}

type AlexaUser struct {
	UserId      string `json:"userId"`
	AccessToken string `json:"accessToken,omitempty"`
}

type AlexaPresentationAPL struct {
}

type AlexaAudioPlayer struct {
	Token                string `json:"token,omitempty"`
	OffsetInMilliseconds int64  `json:"offsetInMilliseconds,omitempty"`
	PlayerActivity       string `json:"playerActivity"`
}

type AlexaSystem struct {
	ApiAccessToken string            `json:"apiAccessToken,omitempty"`
	ApiEndpoint    string            `json:"apiEndpoint,omitempty"`
	Application    *AlexaApplication `json:"application,omitempty"`
	Device         *AlexaDevice      `json:"device,omitempty"`
	Unit           *AlexaUnit        `json:"unit,omitempty"`
	Person         *AlexaPerson      `json:"person,omitempty"`
	User           *AlexaUser        `json:"user,omitempty"`
}

type AlexaDevice struct {
	DeviceId             string                 `json:"deviceId"`
	SupportedInterfaces  map[string]interface{} `json:"supportedInterfaces,omitempty"`
	PersistentEndpointId string                 `json:"persistentEndpointId,omitempty"`
}

type AlexaUnit struct {
	UnitId           string `json:"unitId"`
	PersistentUnitId string `json:"persistentUnitId,omitempty"`
}

type AlexaPerson struct {
	PersonId    string `json:"personId"`
	AccessToken string `json:"accessToken,omitempty"`
}

type AlexaViewport struct {
	Experiences        []*AlexaViewportExperience `json:"experiences,omitempty"`
	Mode               string                     `json:"mode,omitempty"`
	Shape              string                     `json:"shape,omitempty"`
	PixelHeight        uint32                     `json:"pixelHeight,omitempty"`
	PixelWidth         uint32                     `json:"pixelWidth,omitempty"`
	CurrentPixelWidth  uint32                     `json:"currentPixelWidth,omitempty"`
	CurrentPixelHeight uint32                     `json:"currentPixelHeight,omitempty"`
	DPI                uint32                     `json:"dpi,omitempty"`
	Touch              []string                   `json:"touch,omitempty"`
	Keyboard           []string                   `json:"keyboard,omitempty"`
	Video              *AlexaViewportVideo        `json:"video,omitempty"`
}

type AlexaViewportInfo struct {
	ID                string               `json:"id"`
	Format            string               `json:"format,omitempty"`
	LineCount         uint32               `json:"lineCount,omitempty"`
	LineLength        uint32               `json:"lineLength,omitempty"`
	Type              string               `json:"type"`
	SupportedProfiles []string             `json:"supportedProfiles,omitempty"`
	InterSegments     []*AlexaInterSegment `json:"interSegments,omitempty"`
}

type AlexaViewportExperience struct {
	CanRotate bool `json:"canRotate,omitempty"`
	CanResize bool `json:"canResize,omitempty"`
}

type AlexaViewportVideo struct {
	Codecs []string `json:"codecs,omitempty"`
}

type AlexaInterSegment struct {
	X          uint32 `json:"x"`
	Y          uint32 `json:"y"`
	Characters string `json:"characters"`
}

type AlexaRequest interface {
	Type() string
	RequestID() string
	Timestamp() string
	Locale() string
}

type AlexaBaseRequest struct {
	SrcType      string `json:"type,omitempty"`
	SrcRequestID string `json:"requestId,omitempty"`
	SrcTimestamp string `json:"timestamp,omitempty"`
	SrcLocale    string `json:"locale,omitempty"`
}

func (r *AlexaBaseRequest) Type() string {
	return r.SrcType
}
func (r *AlexaBaseRequest) RequestID() string {
	return r.SrcRequestID
}
func (r *AlexaBaseRequest) Timestamp() string {
	return r.SrcTimestamp
}
func (r *AlexaBaseRequest) Locale() string {
	return r.SrcLocale
}

type AlexaLaunchRequest struct {
	AlexaBaseRequest
}

// AlexaCanFulfillIntentRequest - TODO: published skills only
type AlexaCanFulfillIntentRequest struct {
	AlexaBaseRequest
}

type AlexaIntentRequest struct {
	AlexaBaseRequest
	DialogState string       `json:"dialogState,omitempty"`
	Intent      *AlexaIntent `json:"intent"`
}

type AlexaSessionEndedRequest struct {
	AlexaBaseRequest
	Reason string                  `json:"reason"`
	Error  *AlexaSessionEndedError `json:"error,omitempty"`
}

type AlexaIntent struct {
	Name               string                `json:"name"`
	ConfirmationStatus string                `json:"confirmationStatus,omitempty"`
	Slots              map[string]*AlexaSlot `json:"slots,omitempty"`
}

type AlexaSlot struct {
	Name               string            `json:"name"`
	ConfirmationStatus string            `json:"confirmationStatus,omitempty"`
	Source             string            `json:"source,omitempty"`
	Value              string            `json:"value,omitempty"`
	Resolutions        *AlexaResolutions `json:"resolutions,omitempty"`
	SlotValue          *AlexaSlotValue   `json:"slotValue,omitempty"`
}

type AlexaResolutions struct {
	ResolutionsPerAuthority []*AlexaResolution `json:"resolutionsPerAuthority"`
}

type AlexaResolution struct {
	Authority string                       `json:"authority,omitempty"`
	Status    *AlexaEntityResolutionStatus `json:"status,omitempty"`
	Values    []*AlexaResolvedSlotValue    `json:"values,omitempty"`
}

type AlexaEntityResolutionStatus struct {
	Code string `json:"code"`
}

type AlexaResolvedSlotValue struct {
	Value AlexaResolvedSlotValueContent `json:"value,omitempty"`
}

type AlexaResolvedSlotValueContent struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type AlexaSingleSlotValue struct {
	Type        string            `json:"type,omitempty"`
	Value       string            `json:"value,omitempty"`
	Resolutions *AlexaResolutions `json:"resolutions,omitempty"`
}

type AlexaSlotValue struct {
	AlexaSingleSlotValue
	Values []*AlexaSingleSlotValue `json:"values,omitempty"`
}

type AlexaSessionEndedError struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
}

// AlexaResponseEnvelope struct
type AlexaResponseEnvelope struct {
	Version           string            `json:"version,omitempty"`
	SessionAttributes map[string]string `json:"sessionAttributes,omitempty"`
	Response          *AlexaResponse    `json:"response,omitempty"`
}

// AlexaResponse struct
type AlexaResponse struct {
	OutputSpeeech    *AlexaOutputSpeeech `json:"outputSpeech,omitempty"`
	Card             *AlexaCard          `json:"card,omitempty"`
	Reprompt         *AlexaReprompt      `json:"reprompt,omitempty"`
	ShouldEndSession interface{}         `json:"shouldEndSession,omitempty"`
	Directives       []interface{}       `json:"directives,omitempty"`
}

// AlexaOutputSpeeech struct
type AlexaOutputSpeeech struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	SSML         string `json:"ssml,omitempty"`
	PlayBehavior string `json:"playBehavior,omitempty"`
}

type AlexaReprompt struct {
	OutputSpeeech *AlexaOutputSpeeech `json:"outputSpeech"`
}

type AlexaCard struct {
	Type    string          `json:"type"`
	Title   string          `json:"title,omitempty"`
	Content string          `json:"content,omitempty"`
	Text    string          `json:"text,omitempty"`
	Image   *AlexaCardImage `json:"image,omitempty"`
}

type AlexaCardImage struct {
	SmallImageUrl string `json:"smallImageUrl,omitempty"`
	LargeImageUrl string `json:"largeImageUrl,omitempty"`
}
