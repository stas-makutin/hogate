package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var authTokenSecret []byte
var codeTokenLifeTime = time.Minute * 3
var accessTokenLifeTime = time.Hour * 1
var refreshTokenLifeTime = time.Hour * 24 * 90

const generatedAuthTokenSecretSize = 32

var generatedAuthTokenSecretAlphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-")

var authTokenCookie = "hogoken"

type httpAuthorizationKey struct{}

// AuthTokenClaims type
type AuthTokenClaims struct {
	Type      byte     `json:"t,omitempty"`
	ClientID  string   `json:"c,omitempty"`
	UserName  string   `json:"u,omitempty"`
	Scope     []string `json:"s,omitempty"`
	ExpiresAt int64    `json:"e,omitempty"`
}

const (
	authTokenCode = byte(iota)
	authTokenAccess
	authTokenRefresh
)

// Valid method
func (c AuthTokenClaims) Valid() error {
	if c.Type != authTokenCode && c.Type != authTokenAccess && c.Type != authTokenRefresh {
		return fmt.Errorf("unknown Type")
	}
	if c.ExpiresAt > 0 && c.ExpiresAt < time.Now().UTC().Unix() {
		return fmt.Errorf("expired")
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
					err = fmt.Errorf("negative value not allowed")
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

func createAuthToken(tokenType byte, clientID, userName string, scope scopeSet) (string, error) {
	var duration time.Duration
	switch tokenType {
	case authTokenCode:
		duration = codeTokenLifeTime
	case authTokenAccess:
		duration = accessTokenLifeTime
	case authTokenRefresh:
		duration = refreshTokenLifeTime
	default:
		return "", fmt.Errorf("unknown token type %v", tokenType)
	}
	expiresAt := time.Now().UTC().Add(duration).Unix()

	claims := AuthTokenClaims{
		Type:      tokenType,
		ClientID:  clientID,
		UserName:  userName,
		ExpiresAt: expiresAt,
	}

	for k := range scope {
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

func verifyAuthToken(token string, scope ...string) (bool, *AuthTokenClaims) {
	claim, err := parseAuthToken(token)
	if err != nil || claim.Type != authTokenAccess {
		return false, nil
	}

	valid := true
	if len(scope) > 0 {
		var ss scopeSet
		if len(claim.Scope) > 0 {
			ss = newScopeSet(claim.Scope...)
		} else if claim.UserName != "" && claim.ClientID == "" {
			if ui, ok := credentials.user(claim.UserName); ok {
				ss = ui.scope
			} else {
				valid = false
			}
		} else {
			valid = false
		}
		if valid {
			for _, v := range scope {
				if _, ok := ss[v]; !ok {
					valid = false
					break
				}
			}
		}
	}

	if valid {
		return true, claim
	}
	return false, nil
}

func testAuthorization(r *http.Request, scope ...string) (int, *AuthTokenClaims) {
	token := ""
	authorization := r.Header.Get("Authorization")
	if strings.HasPrefix(authorization, "Bearer ") {
		token = authorization[7:]
	}
	if token == "" {
		if cookie, err := r.Cookie(authTokenCookie); err == nil {
			token = cookie.Value
		}
	}
	if token == "" {
		return http.StatusForbidden, nil
	}

	if valid, claim := verifyAuthToken(token, scope...); valid {
		httpSetLogBulkData(r, logData{
			"auth": {
				"u": claim.UserName,
				"c": claim.ClientID,
			},
		})
		return http.StatusOK, claim
	}

	return http.StatusForbidden, nil
}

func httpAuthorization(r *http.Request) *AuthTokenClaims {
	if claim, ok := r.Context().Value(httpAuthorizationKey{}).(*AuthTokenClaims); ok {
		return claim
	}
	return nil
}

func authorizationHandler(scope ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			status, claim := testAuthorization(r, scope...)
			if status == http.StatusOK {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httpAuthorizationKey{}, claim)))
				return
			}
			http.Error(w, http.StatusText(status), status)
		})
	}
}
