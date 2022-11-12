package main

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func addOAuthRoutes(router *http.ServeMux) {
	handleDedicatedRoute(router, routeOAuthAuthorize, http.HandlerFunc(oauthAuthorize))
	handleDedicatedRoute(router, routeOAuthToken, http.HandlerFunc(oauthToken))
}

func oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	responseType := r.URL.Query().Get("response_type")
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	// general validation
	if responseType != "code" || clientID == "" || redirectURI == "" || state == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	parsedScope := parseScope(scope)

	// validate clientID, redirectUrl, and scope
	ci, ok := credentials.client(clientID)
	if !ok || ci.options&coAuthorizationCode == 0 || !ci.matchRedirectURI(redirectURI) || !ci.scope.test(parsedScope, false) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// authorize
	message := ""
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		targetURL := redirectURI
		if strings.Contains(targetURL, "?") {
			targetURL += "&"
		} else {
			targetURL += "?"
		}

		action := r.URL.Query().Get("action")
		if action == "deny" {
			targetURL += fmt.Sprintf(
				"error=access_denied&error_description=The+request+denied.&state=%v", url.QueryEscape(state),
			)
			w.Header().Set("Location", targetURL)
			w.WriteHeader(http.StatusFound)
			return
		}

		username := r.PostForm.Get("username")
		password := r.PostForm.Get("password")

		if ui, ok := credentials.verifyUser(username, password); ok && ui.scope.test(parsedScope, false) {
			code, err := createAuthToken(authTokenCode, clientID, ui.name, parsedScope)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			targetURL += fmt.Sprintf(
				"code=%v&client_id=%v&scope=%v&state=%v", url.QueryEscape(code), url.QueryEscape(clientID), url.QueryEscape(scope), url.QueryEscape(state),
			)
			w.Header().Set("Location", targetURL)
			w.WriteHeader(http.StatusFound)
			return
		}

		message = "User unknown or has no permission."
	}

	actionURL := fmt.Sprintf(
		"?response_type=%v&client_id=%v&redirect_uri=%v&scope=%v&state=%v&action=",
		url.QueryEscape(responseType), url.QueryEscape(clientID), url.QueryEscape(redirectURI), url.QueryEscape(scope), url.QueryEscape(state),
	)

	var scopeList strings.Builder
	for k := range parsedScope {
		name := scopeDisplayName(k)
		scopeList.WriteString("<li>")
		scopeList.WriteString(html.EscapeString(name))
		scopeList.WriteString("</li>")
	}

	if message != "" {
		message = fmt.Sprintf(`<tr><td colspan="2" align="center"><font color="red">%v</font></td></tr>`, html.EscapeString(message))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w,
		`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Home Access Authorization</title>
</head>
<body>
	<form action="%v" method="POST">
		<h1>Authorize</h1>
		The %v would like to access:
		<ul>%v</ul>
		<table cellpadding="3" cellspacing="0">
			<tr>
				<td><label for="username">User Name</label></td>
				<td><input type="text" name="username" placeholder="Please enter your user name"></td>
			</tr>
			<tr>
				<td><label for="password">Password</label></td>
				<td><input type="password" name="password" placeholder="Please enter your password"></td>
			</tr>
			%v
			<tr>
				<td colspan="2" align="center">
					<button formaction="%v" type="submit">Allow</button>
					<button type="submit">Deny</button>
				</td>
			</tr>
		</table>
	</form>
</body>
</html>`,
		actionURL+"deny", html.EscapeString(ci.name), scopeList.String(), message, actionURL+"allow",
	)
}

func oauthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	successfulResponse := func(clientID, userName string, scope scopeSet, setRefreshToken bool) {
		accessToken, err := createAuthToken(authTokenAccess, clientID, userName, scope)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		refreshToken := ""
		if setRefreshToken {
			refreshToken, err = createAuthToken(authTokenRefresh, clientID, userName, scope)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			refreshToken = `,"refresh_token":"` + jsonEscape(refreshToken) + `"`
		}

		fmt.Fprintf(w,
			`{"access_token":"%v","token_type":"bearer","expires_in":%v%v,"scope":"%v"}`,
			jsonEscape(accessToken), int64(accessTokenLifeTime/time.Second), refreshToken, jsonEscape(scope.String()),
		)
	}
	basicAuthPair := func(first, second string) (string, string) {
		if f, s, ok := r.BasicAuth(); ok {
			return f, s
		}
		return first, second
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	errorStatus := http.StatusBadRequest
	errorCode := "unsupported_grant_type"
	grantType := r.Form.Get("grant_type")
	switch grantType {
	case "authorization_code":
		code := r.Form.Get("code")
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		redirectURI := r.Form.Get("redirect_uri")

		if clientSecret == "" {
			clientID, clientSecret = basicAuthPair(clientID, clientSecret)
		}

		if code == "" || clientID == "" || clientSecret == "" || redirectURI == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientID); !ok || clientSecret != ci.secret || !ci.matchRedirectURI(redirectURI) {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if claims, err := parseAuthToken(code); err != nil || claims.Type != authTokenCode || clientID != claims.ClientID {
			errorCode = "invalid_grant"
		} else {
			successfulResponse(clientID, claims.UserName, newScopeSet(claims.Scope...), ci.options&coRefreshToken != 0)
			return
		}

	case "client_credentials":
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scope := r.Form.Get("scope")

		if clientSecret == "" {
			clientID, clientSecret = basicAuthPair(clientID, clientSecret)
		}

		if clientID == "" || clientSecret == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientID); !ok || clientSecret != ci.secret {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if ci.options&coClientCredentials == 0 {
			errorCode = "unauthorized_client"
		} else if parsedScope := parseScope(scope); !ci.scope.test(parsedScope, false) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse(clientID, "", parsedScope, ci.options&coRefreshToken != 0)
			return
		}

	case "refresh_token":
		refreshToken := r.Form.Get("refresh_token")
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scope := r.Form.Get("scope")

		if clientSecret == "" {
			clientID, clientSecret = basicAuthPair(clientID, clientSecret)
		}

		if refreshToken == "" || clientID == "" || clientSecret == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientID); !ok || clientSecret != ci.secret {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if ci.options&coRefreshToken == 0 {
			errorCode = "unauthorized_client"
		} else if claims, err := parseAuthToken(refreshToken); err != nil || claims.Type != authTokenRefresh || claims.ClientID != clientID {
			errorCode = "invalid_grant"
		} else if originScope := newScopeSet(claims.Scope...); scope == "" {
			successfulResponse(clientID, claims.UserName, originScope, true)
			return
		} else if parsedScope := parseScope(scope); !ci.scope.test(parsedScope, false) || !parsedScope.same(originScope) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse(clientID, claims.UserName, originScope, true)
			return
		}

	case "user_credentials":
		userName := r.Form.Get("user")
		password := r.Form.Get("password")
		scope := r.Form.Get("scope")

		if password == "" {
			userName, password = basicAuthPair(userName, password)
		}

		if userName == "" || password == "" {
			errorCode = "invalid_request"
		} else if ui, ok := credentials.verifyUser(userName, password); !ok {
			errorCode = "invalid_user"
			errorStatus = http.StatusUnauthorized
		} else if parsedScope := parseScope(scope); !ui.scope.test(parsedScope, true) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse("", userName, parsedScope, false)
			return
		}
	}
	w.WriteHeader(errorStatus)
	fmt.Fprintf(w, `{"error":"%v"}`, jsonEscape(errorCode))
}
