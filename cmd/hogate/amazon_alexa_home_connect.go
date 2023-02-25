package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type axhcConfig struct {
	valid        bool
	AudioArchive *axhcAudioArchive `yaml:"audioArchive,omitempty"`
	PhotoArchive *axhcPhotoArchive `yaml:"photoArchive,omitempty"`
}

type axhcAudioArchive struct {
	Directories []*axhcAudioDirectory `yaml:"directories,omitempty"`
	Playlists   []*axhcAudioPlaylist  `yaml:"playlists,omitempty"`
}

type axhcAudioDirectory struct {
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

type axhcAudioPlaylist struct {
	Name  string                   `yaml:"name"`
	Files []*axhcAudioPlaylistFile `yaml:"files"`
}

type axhcAudioPlaylistFile struct {
	URL      string `yaml:"url"`
	Title    string `yaml:"title,omitempty"`
	Subtitle string `yaml:"subtitle,omitempty"`
	Art      string `yaml:"art,omitempty"`
}

type axhcPhotoArchive struct {
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

var ahConfig axhcConfig

func validateAlexaHomeConnectConfig(cfgError configError) {
	if config.AlexaHomeConnect == nil || config.AlexaHomeConnect.Config == "" {
		return
	}
	if err := loadSubConfig(config.AlexaHomeConnect.Config, &ahConfig); err != nil {
		cfgError(fmt.Sprintf("alexaHomeConnect.config, unable to load configuration file '%v': %v", config.AlexaHomeConnect.Config, err))
		return
	}

	if ahConfig.AudioArchive != nil {
		for i, directory := range ahConfig.AudioArchive.Directories {
			if directory == nil {
				continue
			}
			dirError := func(msg string) {
				cfgError(fmt.Sprintf("alexaHomeConnect.audioArchive.directories, directory %v: %v", i, msg))
			}
			if _, err := url.Parse(directory.URL); err != nil {
				dirError(fmt.Sprintf("invalid base URL, error: %v", err))
			}

		}

/*
		for i, playlist := range ahConfig.AudioArchive.Playlists {
			if playlist == nil {
				continue
			}
			plError := func(msg string) {
				cfgError(fmt.Sprintf("alexaHomeConnect.audioArchive.playlists, playlist %v: %v", i, msg))
			}
		}
*/
	}

	if ahConfig.PhotoArchive != nil {


		
	}
}

func addAmazonAlexaRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeAmazonAlexaHomeConnect, http.HandlerFunc(routeAmazonAlexaHomeConnectHandle))
}

func routeAmazonAlexaHomeConnectHandle(w http.ResponseWriter, r *http.Request) {
	// parse
	request := acceptAlexaRequest(r)
	if request == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	// authenticate
	accessToken := ""
	if request.Context != nil && request.Context.System != nil && request.Context.System.User != nil {
		accessToken = request.Context.System.User.AccessToken
	}
	if valid, _ := verifyAuthToken(accessToken, scopeYandexHome); !valid {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	response := AlexaResponseEnvelope{
		Version: "1.0",
	}

	switch r := request.Request.(type) {
	case *AlexaLaunchRequest:
		response.Response = &AlexaResponse{
			OutputSpeeech: &AlexaOutputSpeeech{
				Type: Alexa_OutputSpeech_PlainText,
				Text: "What would you like to access?",
			},
			Reprompt: &AlexaReprompt{
				OutputSpeeech: &AlexaOutputSpeeech{
					Type: Alexa_OutputSpeech_PlainText,
					Text: "You could choose either audio or photo archive.",
				},
			},
			Directives: []interface{}{
				&AlexaDirectiveRenderDocument{
					AlexaDirective: AlexaDirective{Type: ADT_APL_RenderDocument},
					Token:          "123456",
					Document: json.RawMessage(`{
						"type": "APL",
						"version": "2022.1",
						"mainTemplate": {
							"item": [
								{
									"source": "https://makutin.linkpc.net/tteesstt/media/photo.jpg",
									"scale": "best-fit",
									"type": "Image",
									"width": "100vw",
									"height": "100vh"
								}
							]
						}
					}`),
				},
			},
			ShouldEndSession: false,
		}
	case *AlexaIntentRequest:
		switch r.Intent.Name {
		case "AudioPlaylistIntent":
			response.Response = &AlexaResponse{
				Directives: []interface{}{
					&AlexaDirectiveAudioPlayerPlay{
						AlexaDirective: AlexaDirective{Type: ADT_AudioPlayer_Play},
						PlayBehavior:   ADT_AudioPlayerPlay_ReplaceAll,
						AudioItem: &AlexaAudioItem{
							Stream: &AlexaAudioItemStream{
								URL:   "https://makutin.linkpc.net/tteesstt/media/music.mp3",
								Token: "123456789ee12",
							},
						},
					},
				},
			}
		case "AMAZON.PauseIntent":
			response.Response = &AlexaResponse{
				Directives: []interface{}{
					&AlexaDirectiveAudioPlayerPlay{
						AlexaDirective: AlexaDirective{Type: ADT_AudioPlayer_Stop},
					},
				},
			}
		}
	case *AlexaSessionEndedRequest:
	case *AlexaAudioPlayerPlaybackRequest:
	case *AlexaAudioPlayerPlaybackFailedRequest:
		response.Response = &AlexaResponse{}
	default:
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}
