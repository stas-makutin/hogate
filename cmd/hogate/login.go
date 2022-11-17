package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const DefaultMaxAge = 60 * 60 * 24 * 30 // one month
const DefaultLoginTitle = "Home Gateway Login"
const DefaultLoginHeader = `<img src="data:image/png;base64,
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
AElFTkSuQmCC">`

var RememberMeMaxAge int = DefaultMaxAge

func addLoginRoute(router *http.ServeMux) {
	handleDedicatedRoute(router, routeLogin, http.HandlerFunc(login))
}

func validateLoginConfig(cfgError configError) {
	if config.Login != nil && config.Login.RememberMaxAge != "" {
		duration, err := parseTimeDuration(config.Login.RememberMaxAge)
		if err == nil && duration < 0 {
			err = fmt.Errorf("negative value not allowed")
		}
		if err == nil {
			RememberMeMaxAge = int(duration / time.Second)
		} else {
			cfgError(fmt.Sprintf("login.rememberMaxAge is not valid: %v", err))
		}
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	scope := r.URL.Query().Get("scope")

	if redirectURI == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	message := ""
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		userName := r.PostForm.Get("username")
		password := r.PostForm.Get("password")
		remember := r.PostForm.Get("remember")
		if ui, ok := credentials.verifyUser(userName, password); ok {
			if parsedScope := parseScope(scope); ui.scope.test(parsedScope, true) {
				accessToken, err := createAuthToken(authTokenAccess, "", userName, parsedScope)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				cookie := &http.Cookie{
					Name:     authTokenCookie,
					Value:    accessToken,
					HttpOnly: true,
				}
				if remember == "on" && RememberMeMaxAge > 0 {
					cookie.MaxAge = RememberMeMaxAge
				}

				http.SetCookie(w, cookie)
				w.Header().Set("Location", redirectURI)
				w.WriteHeader(http.StatusFound)
				return
			}
		}

		message = "User Name and/or password are not valid"
	}

	action := fmt.Sprintf("?redirect_uri=%s&scope=%s", url.QueryEscape(redirectURI), url.QueryEscape(scope))

	title := DefaultLoginTitle
	header := DefaultLoginHeader
	if config.Login != nil {
		if config.Login.Title != "" {
			title = config.Login.Title
		}
		if config.Login.Header != "" {
			header = config.Login.Header
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, login_html(action, title, header, message, RememberMeMaxAge > 0))
}

func login_html(action, title, header, message string, addRememberMe bool) string {
	if message != "" {
		message = fmt.Sprintf(`<div id="hgl-m">%s</div>`, message)
	}

	rememberMe := ""
	if addRememberMe {
		rememberMe = `<div id="hgl-r">
					<input type="checkbox" name="remember">
					<label for="remember">Remember me on this device</label>
				</div>`
	}

	return fmt.Sprintf(
		`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>%s</title>
	<style>
		html,body{width:100%%;height:100%%;margin:0;}
		body{display:flex;justify-content:center;align-items:center;}
		#hgl-b{padding:1em 2em;border:1px solid activeborder;box-shadow:2px 2px 3px 1px lightgray;text-align:center;}
		#hgl-t,#hgl-s{display:inline-block;margin:1em;}
		.hgl-l,#hgl-u,#hgl-p,#hgl-r{display:block;text-align:left;}
		#hgl-u,#hgl-p{margin:0 0 0.5em 0;min-width: 25em;}
		#hgl-m {color:red;margin:0.5em 0;}
	</style>
</head>
<body>
		<div id="hgl-b">
			<div id="hgl-t">%s</div>
			<form action="%s" method="POST">
				<label class="hgl-l" for="username">User Name</label>
				<input id="hgl-u" type="text" name="username" placeholder="Please enter your user name" autofocus>
				<label class="hgl-l" for="password">Password</label>
				<input id="hgl-p" type="password" name="password" placeholder="Please enter your password">
				%s%s
				<button id="hgl-s" type="submit">Login</button>
			</form>
		</div>
</body>
</html>`,
		title, header, action, message, rememberMe,
	)
}
