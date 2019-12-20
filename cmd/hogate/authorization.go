package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var authTokenSecret []byte
var codeTokenLifeTime = time.Minute * 3
var accessTokenLifeTime = time.Hour * 1
var refreshTokenLifeTime = time.Hour * 24 * 90

const generatedAuthTokenSecretSize = 32

var generatedAuthTokenSecretAlphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-")

type AuthTokenClaims struct {
	Type      byte        `json:"t,omitempty"`
	ClientId  string      `json:"c,omitempty"`
	UserName  string      `json:"u,omitempty"`
	Scope     []scopeType `json:"s,omitempty"`
	ExpiresAt int64       `json:"e,omitempty"`
}

const (
	authTokenCode = byte(iota)
	authTokenAccess
	authTokenRefresh
)

func (c AuthTokenClaims) Valid() error {
	if c.Type != authTokenCode && c.Type != authTokenAccess && c.Type != authTokenRefresh {
		return fmt.Errorf("Unknown Type")
	}
	if c.ExpiresAt > 0 && c.ExpiresAt < time.Now().UTC().Unix() {
		return fmt.Errorf("Expired")
	}
	return nil
}

func validateAuthorizationConfig(cfgError configError) {
	if config.Authorization == nil {
		return
	}

	if config.Authorization.TokenSecret != "" {
		authTokenSecret = []byte(config.Authorization.TokenSecret)
	} else {
		authTokenSecret = []byte(randomString(generatedAuthTokenSecretSize, generatedAuthTokenSecretAlphabet))
	}

	if config.Authorization.LifeTime != nil {
		parseLifeTime := func(src, name string) (time.Duration, bool) {
			if src != "" {
				duration, err := parseTimeDuration(src)
				if err == nil && duration < 0 {
					err = fmt.Errorf("negative value not allowed.")
				}
				if err == nil {
					return duration, true
				}
				cfgError(fmt.Sprintf("%v is not valid: %v", name, err))
			}
			return 0, false
		}

		if duration, ok := parseLifeTime(config.Authorization.LifeTime.CodeToken, "authorization.lifeTime.codeToken"); ok {
			codeTokenLifeTime = duration
		}
		if duration, ok := parseLifeTime(config.Authorization.LifeTime.AccessToken, "authorization.lifeTime.accessToken"); ok {
			accessTokenLifeTime = duration
		}
		if duration, ok := parseLifeTime(config.Authorization.LifeTime.RefreshToken, "authorization.lifeTime.refreshToken"); ok {
			refreshTokenLifeTime = duration
		}
	}
}

func createAuthToken(tokenType byte, clientId, userName string, scope scopeSet) (string, error) {
	var duration time.Duration
	switch tokenType {
	case authTokenCode:
		duration = codeTokenLifeTime
	case authTokenAccess:
		duration = accessTokenLifeTime
	case authTokenRefresh:
		duration = refreshTokenLifeTime
	default:
		return "", fmt.Errorf("Unknown token type %v.", tokenType)
	}
	expiresAt := time.Now().UTC().Add(duration).Unix()

	claims := AuthTokenClaims{
		Type:      tokenType,
		ClientId:  clientId,
		UserName:  userName,
		ExpiresAt: expiresAt,
	}

	for k, _ := range scope {
		claims.Scope = append(claims.Scope, k)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(authTokenSecret)
}

func parseAuthToken(tokenString string) (*AuthTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return authTokenSecret, nil
	})
	if err == nil {
		if claims, ok := token.Claims.(*AuthTokenClaims); ok && token.Valid {
			return claims, nil
		}
		err = fmt.Errorf("invalid token")
	}
	return nil, err
}

func authorizationMiddleware(scope ...scopeType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			next.ServeHTTP(w, r)
		})
	}
}