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
	clientId := r.URL.Query().Get("client_id")
	redirectUri := r.URL.Query().Get("redirect_uri")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	// general validation
	if responseType != "code" || clientId == "" || redirectUri == "" || state == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	parsedScope, err := parseScope(scope)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// validate clientId, redirectUrl, and scope
	ci, ok := credentials.client(clientId)
	if !ok || ci.options&coAuthorizationCode == 0 || redirectUri != ci.redirectUri || !ci.scope.test(parsedScope, false) {
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

		if strings.Contains(redirectUri, "?") {
			redirectUri += "&"
		} else {
			redirectUri += "?"
		}

		action := r.URL.Query().Get("action")
		if action == "deny" {
			redirectUri += fmt.Sprintf(
				"error=access_denied&error_description=The+request+denied.&state=%v", url.QueryEscape(state),
			)
			w.Header().Set("Location", redirectUri)
			w.WriteHeader(http.StatusFound)
			return
		}

		username := r.PostForm.Get("username")
		password := r.PostForm.Get("password")

		if ui, ok := credentials.verifyUser(username, password); ok && ui.scope.test(parsedScope, false) {
			code, err := createAuthToken(authTokenCode, clientId, ui.name, parsedScope)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			redirectUri += fmt.Sprintf(
				"code=%v&client_id=%v&scope=%v&state=%v", url.QueryEscape(code), url.QueryEscape(clientId), url.QueryEscape(scope), url.QueryEscape(state),
			)
			w.Header().Set("Location", redirectUri)
			w.WriteHeader(http.StatusFound)
			return
		} else {
			message = "User unknown or has no permission."
		}
	}

	actionUrl := r.URL.RawQuery
	if actionUrl != "" {
		actionUrl += "&action="
	}

	var scopeList strings.Builder
	for k, _ := range parsedScope {
		name := k.displayName()
		if name == "" {
			name = fmt.Sprintf("Code %v", k)
		}
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
		actionUrl+"deny", html.EscapeString(ci.name), scopeList, message, actionUrl+"allow",
	)
}

func oauthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	successfulResponse := func(clientId, userName string, scope scopeSet, setRefreshToken bool) {
		accessToken, err := createAuthToken(authTokenAccess, clientId, userName, ss)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		refreshToken := ""
		if setRefreshToken {
			refreshToken, err = createAuthToken(authTokenRefresh, clientId, userName, ss)
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
		clientId := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		redirectUri := r.Form.Get("redirect_uri")

		if code == "" || clientId == "" || clientSecret == "" || redirectUri == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientId); !ok || clientSecret != ci.secret || redirectUri != ci.redirectUri {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if claims, err := parseAuthToken(code); err != nil || claims.Type != authTokenCode || clientId != claims.ClientId {
			errorCode = "invalid_grant"
		} else {
			successfulResponse(clientId, claims.UserName, newScopeSet(claims.Scope...), ci.options&coRefreshToken != 0)
			return
		}

	case "client_credentials":
		clientId := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scope := r.Form.Get("scope")

		if clientId == "" || clientSecret == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientId); !ok || clientSecret != ci.secret {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if ci.options&coClientCredentials == 0 {
			errorCode = "unauthorized_client"
		} else if parsedScope, err := parseScope(scope); err != nil || !ci.scope.test(parsedScope, false) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse(clientId, "", parsedScope, ci.options&coRefreshToken != 0)
			return
		}

	case "refresh_token":
		refreshToken := r.Form.Get("refresh_token")
		clientId := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scope := r.Form.Get("scope")

		if refreshToken == "" || clientId == "" || clientSecret == "" {
			errorCode = "invalid_request"
		} else if ci, ok := credentials.client(clientId); !ok || clientSecret != ci.secret {
			errorCode = "invalid_client"
			errorStatus = http.StatusUnauthorized
		} else if ci.options&coRefreshToken == 0 {
			errorCode = "unauthorized_client"
		} else if claims, err := parseAuthToken(refreshToken); err != nil || claims.Type != authTokenRefresh || claims.ClientId != clientId {
			errorCode = "invalid_grant"
		} else if parsedScope, err := parseScope(scope); err != nil || !ci.scope.test(parsedScope, false) || !parsedScope.same(newScopeSet(claims.Scope...)) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse(clientId, claims.UserName, parsedScope, true)
			return
		}

	case "user_credentials":
		userName := r.Form.Get("user")
		password := r.Form.Get("password")
		scope := r.Form.Get("scope")

		if userName == "" || password == "" {
			errorCode = "invalid_request"
		} else if ui, ok := credentials.verifyUser(userName, password); !ok {
			errorCode = "invalid_user"
			errorStatus = http.StatusUnauthorized
		} else if parsedScope, err := parseScope(scope); err != nil || !ui.scope.test(parsedScope, true) {
			errorCode = "invalid_scope"
		} else {
			successfulResponse("", userName, parsedScope, false)
			return
		}
	}
	w.WriteHeader(errorStatus)
	fmt.Fprintf(w, `{"error":"%v"}`, jsonEscape(errorCode))
}
