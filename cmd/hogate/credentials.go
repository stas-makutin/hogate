package main

import (
	"fmt"
	"strings"
	"unicode"
)

var credentials credentialsContainer

type scopeType uint16

const (
	scopeYandexHome = scopeType(iota)
	scopeYandexDialogs
)

type scopeSet map[scopeType]struct{}

var scopeNames = map[scopeType]string{
	scopeYandexHome:    "yandex-home",
	scopeYandexDialogs: "yandex-dialogs",
}

var scopeDisplayNames = map[scopeType]string{
	scopeYandexHome:    "Yandex Home",
	scopeYandexDialogs: "Yandex Dialogs",
}

type userInfo struct {
	name     string
	password string
	scope    scopeSet
}

const (
	coAuthorizationCode = uint32(1 << iota)
	coClientCredentials
	coRefreshToken
)

type clientInfo struct {
	id          string
	name        string
	secret      string
	redirectURI []string
	options     uint32
	scope       scopeSet
}

type credentialsContainer struct {
	users   map[string]userInfo   // user name -> user ingo
	clients map[string]clientInfo // client id -> client ifo
}

func validateCredentialsConfig(cfgError configError) {
	credentials.users = make(map[string]userInfo)
	credentials.clients = make(map[string]clientInfo)

	for i, user := range config.Credentials.Users {
		userError := func(msg string) {
			cfgError(fmt.Sprintf("credentials.users, user %v: %v", i, msg))
		}

		if user.Name == "" {
			userError("name cannot be empty.")
		} else if _, ok := credentials.users[user.Name]; ok {
			userError(fmt.Sprintf("name '%v' already exists", user.Name))
		}

		if user.Password == "" {
			userError("password cannot be empty")
		}

		scope, err := parseScope(user.Scope)
		if err != nil {
			userError(err.Error())
		}

		credentials.users[user.Name] = userInfo{name: user.Name, password: user.Password, scope: scope}
	}

	for i, client := range config.Credentials.Clients {
		clientError := func(msg string) {
			cfgError(fmt.Sprintf("credentials.clients, client %v: %v", i, msg))
		}

		if client.ID == "" {
			clientError("id cannot be empty")
		} else if _, ok := credentials.clients[client.ID]; ok {
			clientError(fmt.Sprintf("id '%v' already exists", client.ID))
		}

		clientName := client.Name
		if clientName == "" {
			clientName = client.ID
		}

		if client.Secret == "" {
			clientError("secret cannot be empty")
		}

		options, err := parseClientOptions(client.Options)
		if err == nil && options == 0 {
			err = fmt.Errorf("at least one option must be specified")
		}
		if err != nil {
			clientError(fmt.Sprintf("invalid options: %v", err))
		}

		if options&coAuthorizationCode != 0 {
			count := 0
			for _, v := range client.RedirectURI {
				if v != "" {
					count++
				}
			}
			if count <= 0 {
				clientError("At least one non-empty redirectUri must present if authorizationCode option is set.")
			}
		}

		scope, err := parseScope(client.Scope)
		if err != nil {
			clientError(err.Error())
		}

		credentials.clients[client.ID] = clientInfo{
			id:          client.ID,
			name:        clientName,
			secret:      client.Secret,
			redirectURI: client.RedirectURI,
			options:     options,
			scope:       scope,
		}
	}
}

func (s scopeType) String() string {
	if name, ok := scopeNames[s]; ok {
		return name
	}
	return ""
}

func (s scopeType) displayName() string {
	if name, ok := scopeDisplayNames[s]; ok {
		return name
	}
	return ""
}

func (s scopeSet) test(scope scopeSet, allowEmpty bool) bool {
	empty := true
	for k := range scope {
		if _, ok := s[k]; !ok {
			return false
		}
		empty = false
	}
	if empty {
		return allowEmpty
	}
	return true
}

func (s scopeSet) same(scope scopeSet) bool {
	if len(s) == len(scope) {
		for k := range scope {
			if _, ok := s[k]; !ok {
				return false
			}
		}
		for k := range s {
			if _, ok := scope[k]; !ok {
				return false
			}
		}
		return true
	}
	return false
}

func (s scopeSet) String() string {
	var sb strings.Builder
	for k := range s {
		if name := k.String(); name != "" {
			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(name)
		}
	}
	return sb.String()
}

func (ci clientInfo) matchRedirectURI(redirectURI string) bool {
	for _, v := range ci.redirectURI {
		if v == redirectURI {
			return true
		}
	}
	return false
}

func (c credentialsContainer) client(clientID string) (*clientInfo, bool) {
	ci, ok := c.clients[clientID]
	return &ci, ok
}

func (c credentialsContainer) user(userName string) (*userInfo, bool) {
	ui, ok := c.users[userName]
	return &ui, ok
}

func (c credentialsContainer) verifyUser(userName, password string) (*userInfo, bool) {
	if ui, ok := c.user(userName); ok && ui.password == password {
		return ui, true
	}
	return nil, false
}

func newScopeSet(scope ...scopeType) scopeSet {
	rv := make(scopeSet)
	for _, s := range scope {
		rv[s] = struct{}{}
	}
	return rv
}

func parseScope(scope string) (scopeSet, error) {
	rv := make(scopeSet)
	for _, word := range strings.FieldsFunc(scope, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		if word != "" {
			found := false
			for k, v := range scopeNames {
				if strings.EqualFold(word, v) {
					rv[k] = struct{}{}
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown scope '%v'", word)
			}
		}
	}
	return rv, nil
}

func parseClientOptions(options string) (uint32, error) {
	rv := uint32(0)
	for _, word := range strings.FieldsFunc(options, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		if word != "" {
			switch strings.ToLower(word) {
			case "authorizationcode":
				rv |= coAuthorizationCode
			case "clientcredentials":
				rv |= coClientCredentials
			case "refreshtoken":
				rv |= coRefreshToken
			default:
				return 0, fmt.Errorf("unknown option '%v'", word)
			}
		}
	}
	return rv, nil
}
