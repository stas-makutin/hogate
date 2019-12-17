package main

import (
	"fmt"
	"strings"
	"unicode"
)

var credentials credentialsContainer

type userInfo struct {
	name     string
	password string
	scope    []uint32
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
	redirectUri string
	options     uint32
	scope       []uint32
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
			userError(fmt.Sprintf("name '%v' already exists.", user.Name))
		}

		if user.Password == "" {
			userError("password cannot be empty.")
		}

		credentials.users[user.Name] = userInfo{name: user.Name, password: user.Password}
	}

	for i, client := range config.Credentials.Clients {
		clientError := func(msg string) {
			cfgError(fmt.Sprintf("credentials.clients, client %v: %v", i, msg))
		}

		if client.Id == "" {
			clientError("id cannot be empty")
		} else if _, ok := credentials.clients[client.Id]; ok {
			clientError(fmt.Sprintf("id '%v' already exists.", client.Id))
		}

		clientName := client.Name
		if clientName == "" {
			clientName = client.Id
		}

		if client.Secret == "" {
			clientError("secret cannot be empty.")
		}

		options, err := parseClientOptions(client.Options)
		if err == nil && options == 0 {
			err = fmt.Errorf("at least one option must be specified.")
		}
		if err != nil {
			clientError(fmt.Sprintf("invalid options: %v", err))
		}

		if options&coAuthorizationCode != 0 && client.RedirectUri == "" {
			clientError("redirectUri cannot be empty if authorizationCode option is set.")
		}

		credentials.clients[client.Id] = clientInfo{
			id:          client.Id,
			name:        clientName,
			secret:      client.Secret,
			redirectUri: client.RedirectUri,
			options:     options,
		}
	}
}

func parseClientOptions(options string) (uint32, error) {
	rv := uint32(0)
	for _, word := range strings.FieldsFunc(options, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		if word != "" {
			switch word {
			case "authorizationCode":
				rv |= coAuthorizationCode
			case "clientCredentials":
				rv |= coClientCredentials
			case "refreshToken":
				rv |= coRefreshToken
			default:
				return 0, fmt.Errorf("unknown option '%v'", word)
			}
		}
	}
	return rv, nil
}
