package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"
)

const (
	routeOAuthAuthorize = iota
	routeOAuthToken
)

type routeInfo struct {
	routeType   int
	rateLimit   float64
	rateBurst   int
	maxBodySize int64
	methods     []string
}

var specialRoutes = map[int]*routeInfo{
	routeOAuthAuthorize: {
		routeType:   routeOAuthAuthorize,
		rateLimit:   5,
		rateBurst:   3,
		maxBodySize: 4096,
		methods:     []string{"GET", "POST"},
	},
	routeOAuthToken: {
		routeType:   routeOAuthToken,
		rateLimit:   10,
		rateBurst:   4,
		maxBodySize: 8196,
		methods:     []string{"GET", "POST"},
	},
}

func validateRouteConfig(cfgError configError) {
	for i, route := range config.Routes {
		var ri routeInfo
		var err error
		valid := true

		ri.routeType, err = parseRouteType(route.Type)
		if err != nil {
			cfgError(fmt.Sprintf("routes, route %v has unknown type '%v'.", i, route.Type))
			valid = false
		}

		hasRateLimit := false
		if route.RateLimit != "" {
			hasRateLimit = true
			ri.rateLimit, ri.rateBurst, err = parseRateLimit(route.RateLimit)
			if err != nil {
				cfgError(fmt.Sprintf("routes, route %v has incorrect rateLimit value '%v': %v", i, route.RateLimit, err))
				valid = false
			}
		}

		hasMaxBodySize := false
		if route.MaxBodySize != "" {
			hasMaxBodySize = true
			ri.maxBodySize, err = parseSizeString(route.MaxBodySize)
			if err == nil && ri.maxBodySize < 0 {
				err = fmt.Errorf("negative value not allowed.")
			}
			if err != nil {
				cfgError(fmt.Sprintf("routes, route %v has incorrect maxBodySize value '%v': %v", i, route.MaxBodySize, err))
				valid = false
			}
		}

		hasMethods := false
		if route.Methods != "" {
			hasMethods = true
			ri.methods, err = parseRouteMethods(route.Methods)
			if err != nil {
				cfgError(fmt.Sprintf("routes, route %v has incorrect methods value '%v': %v", i, route.Methods, err))
				valid = false
			}
		}

		if valid {
			current := specialRoutes[ri.routeType]
			if hasRateLimit {
				current.rateLimit = ri.rateLimit
				current.rateBurst = ri.rateBurst
			}
			if hasMaxBodySize {
				current.maxBodySize = ri.maxBodySize
			}
			if hasMethods {
				current.methods = ri.methods
			}
		}
	}
}

func parseRouteType(t string) (int, error) {
	switch t {
	case "oauth-authorize":
		return routeOAuthAuthorize, nil
	case "oauth-token":
		return routeOAuthToken, nil
	}
	return 0, fmt.Errorf("Unrecognized route type.")
}

func parseRateLimit(rateLimit string) (float64, int, error) {
	rateParts := strings.FieldsFunc(rateLimit, func(r rune) bool { return r == ',' || r == ';' })
	ratePartsLen := len(rateParts)
	if ratePartsLen < 0 || ratePartsLen > 2 {
		return 0, 0, fmt.Errorf("expected limit and optional burst value, comma or semicolon separated.")
	}
	limitStr := strings.TrimSpace(rateParts[0])
	limit, err := strconv.ParseFloat(limitStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid limit value '%v': %v", limitStr, err)
	}
	burst := int(1)
	if ratePartsLen > 1 {
		burstStr := strings.TrimSpace(rateParts[1])
		burst, err = strconv.Atoi(burstStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit value '%v': %v", burstStr, err)
		}
	}
	return limit, burst, nil
}

func parseRouteMethods(methods string) ([]string, error) {
	var rv []string
	for _, method := range strings.FieldsFunc(methods, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		if method != "" {
			rv = append(rv, method)
		}
	}
	if len(rv) <= 0 {
		return []string{}, fmt.Errorf("at least one method must present")
	}
	return rv, nil
}

func routeMiddleware(ri *routeInfo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
