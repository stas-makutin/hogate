package main

// request

// YandexDialogsRequestEnvelope struct
type YandexDialogsRequestEnvelope struct {
	Meta           *YandexDialogsMeta          `json:"meta,omitempty"`
	Request        *YandexDialogsRequest       `json:"request,omitempty"`
	AccountLinking *struct{}                   `json:"account_linking_complete_event,omitempty"`
	Session        YandexDialogsRequestSession `json:"session"`
	State          *YandexDialogsRequestState  `json:"state"`
	Version        string                      `json:"version"`
}

// YandexDialogsMeta struct
type YandexDialogsMeta struct {
	Locale     string                 `json:"locale"`
	Timezone   string                 `json:"timezone"`
	ClientID   string                 `json:"client_id,omitempty"`
	Interfaces map[string]interface{} `json:"interfaces,omitempty"`
}

// YandexDialogsRequestSession struct
type YandexDialogsRequestSession struct {
	New       bool   `json:"new"`
	MessageID int    `json:"message_id"`
	SessionID string `json:"session_id"`
	SkillID   string `json:"skill_id"`
	UserID    string `json:"user_id"`
}

// YandexDialogsRequestState struct
type YandexDialogsRequestState struct {
	Session map[string]string `json:"session,omitempty"`
	User    interface{}       `json:"user,omitempty"`
}

// YandexDialogsRequest struct
type YandexDialogsRequest struct {
	Command           string               `json:"command"`
	OriginalUtterance string               `json:"original_utterance"`
	Type              string               `json:"type"`
	Markup            *YandexDialogsMarkup `json:"markup,omitempty"`
	Payload           interface{}          `json:"payload,omitempty"`
	Nlu               *YandexDialogsNlu    `json:"nlu,omitempty"`
}

// YandexDialogsMarkup struct
type YandexDialogsMarkup struct {
	DangerousContext bool `json:"dangerous_context,omitempty"`
}

// YandexDialogsNlu struct
type YandexDialogsNlu struct {
	Tokens   []string              `json:"tokens"`
	Entities []YandexDialogsEntity `json:"entities"`
}

// YandexDialogsEntity struct
type YandexDialogsEntity struct {
	Tokens YandexDialogsEntityTokens `json:"tokens"`
	Type   string                    `json:"type"`
	Value  interface{}               `json:"value"`
}

// Yandex Dialogs Entity types
const (
	ydYandexDateTime = "YANDEX.DATETIME"
	ydYandexFio      = "YANDEX.FIO"
	ydYandexGeo      = "YANDEX.GEO"
	ydYandexNumber   = "YANDEX.NUMBER"
)

// YandexDialogsEntityTokens struct
type YandexDialogsEntityTokens struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// YandexDialogsEntityFio struct
type YandexDialogsEntityFio struct {
	FirstName      string `json:"first_name,omitempty"`
	PatronymicName string `json:"patronymic_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
}

// YandexDialogsEntityGeo struct
type YandexDialogsEntityGeo struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
	Street  string `json:"street,omitempty"`
}

// YandexDialogsEntityDateTime struct
type YandexDialogsEntityDateTime struct {
	Year             int  `json:"year,omitempty"`
	YearIsRelative   bool `json:"year_is_relative ,omitempty"`
	Month            int  `json:"month,omitempty"`
	MonthIsRelative  bool `json:"month_is_relative,omitempty"`
	Day              int  `json:"day,omitempty"`
	DayIsRelative    bool `json:"day_is_relative,omitempty"`
	Hour             int  `json:"hour,omitempty"`
	HourIsRelative   bool `json:"hour_is_relative,omitempty"`
	Minute           int  `json:"minute,omitempty"`
	MinuteIsRelative bool `json:"minute_is_relative,omitempty"`
}

// response

// YandexDialogsResponseEnvelope struct
type YandexDialogsResponseEnvelope struct {
	Response       *YandexDialogsResponse       `json:"response,omitempty"`
	AccountLinking *struct{}                    `json:"start_account_linking,omitempty"`
	Session        YandexDialogsResponseSession `json:"session"`
	SessionState   interface{}                  `json:"session_state,omitempty"`
	UserState      interface{}                  `json:"user_state_update,omitempty"`
	Version        string                       `json:"version"`
}

// YandexDialogsResponseSession struct
type YandexDialogsResponseSession struct {
	SessionID string `json:"session_id"`
	MessageID int    `json:"message_id"`
	UserID    string `json:"user_id"`
}

// YandexDialogsResponse struct
type YandexDialogsResponse struct {
	Text       string                `json:"text"`
	TTS        string                `json:"tts,omitempty"`
	Card       interface{}           `json:"card,omitempty"`
	Buttons    []YandexDialogsButton `json:"buttons,omitempty"`
	EndSession bool                  `json:"end_session"`
}

// YandexDialogsButton struct
type YandexDialogsButton struct {
	Title   string      `json:"title"`
	URL     string      `json:"url,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
	Hide    bool        `json:"hide,omitempty"`
}
