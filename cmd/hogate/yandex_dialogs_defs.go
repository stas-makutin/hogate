package main

// request

type YandexDialogsRequestEnvelope struct {
	Meta    YandexDialogsMeta           `json:"meta"`
	Request YandexDialogsRequest        `json:"request"`
	Session YandexDialogsRequestSession `json:"session"`
	Version string                      `json:"version"`
}

type YandexDialogsMeta struct {
	Locale     string                 `json:"locale"`
	Timezone   string                 `json:"timezone"`
	ClientId   string                 `json:"client_id,omitempty"`
	Interfaces map[string]interface{} `json:"interfaces,omitempty"`
}

type YandexDialogsRequestSession struct {
	New       bool   `json:"new"`
	MessageId int    `json:"message_id"`
	SessionId string `json:"session_id"`
	SkillId   string `json:"skill_id"`
	UserId    string `json:"user_id"`
}

type YandexDialogsRequest struct {
	Command           string               `json:"command"`
	OriginalUtterance string               `json:"original_utterance"`
	Type              string               `json:"type"`
	Markup            *YandexDialogsMarkup `json:"markup,omitempty"`
	Payload           interface{}          `json:"payload,omitempty"`
	Nlu               *YandexDialogsNlu    `json:"nlu,omitempty"`
}

type YandexDialogsMarkup struct {
	DangerousContext bool `json:"dangerous_context,omitempty"`
}

type YandexDialogsNlu struct {
	Tokens   []string              `json:"tokens"`
	Entities []YandexDialogsEntity `json:"entities"`
}

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

type YandexDialogsEntityTokens struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type YandexDialogsEntityFio struct {
	FirstName      string `json:"first_name,omitempty"`
	PatronymicName string `json:"patronymic_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
}

type YandexDialogsEntityGeo struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
	Street  string `json:"street,omitempty"`
}

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

type YandexDialogsResponseEnvelope struct {
	Response YandexDialogsResponse        `json:"response"`
	Session  YandexDialogsResponseSession `json:"session"`
	Version  string                       `json:"version"`
}

type YandexDialogsResponseSession struct {
	SessionId string `json:"session_id"`
	MessageId int    `json:"message_id"`
	UserId    string `json:"user_id"`
}

type YandexDialogsResponse struct {
	Text       string                `json:"text"`
	TTS        string                `json:"tts,omitempty"`
	Card       interface{}           `json:"card,omitempty"`
	Buttons    []YandexDialogsButton `json:"buttons,omitempty"`
	EndSession bool                  `json:"end_session"`
}

type YandexDialogsButton struct {
	Title   string      `json:"title"`
	Url     string      `json:"url,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
	Hide    bool        `json:"hide,omitempty"`
}
