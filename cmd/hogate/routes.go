package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/time/rate"
)

const (
	routeOAuthAuthorize = iota
	routeOAuthToken

	routeLogin

	routeYandexHomeHealth
	routeYandexHomeUnlink
	routeYandexHomeDevices
	routeYandexHomeQuery
	routeYandexHomeAction

	routeYandexDialogsTales

	routeAmazonAlexaWhistles
)

type routeBase struct {
	rateLimit        float64
	rateBurst        int
	maxBodySize      int64
	methods          []string
	originAny        bool
	originIncludes   []*regexp.Regexp
	originExcludes   []*regexp.Regexp
	headers          string
	allowCredentials bool
}

type routeInfo struct {
	routeBase
	path string
}

var dedicatedRoutes = map[int]*routeInfo{
	routeOAuthAuthorize: {
		path: "/authorize",
		routeBase: routeBase{
			rateLimit:   10,
			rateBurst:   3,
			maxBodySize: 4096,
			methods:     []string{"GET", "POST", "OPTIONS"},
		},
	},
	routeOAuthToken: {
		path: "/token",
		routeBase: routeBase{
			rateLimit:   20,
			rateBurst:   5,
			maxBodySize: 8196,
			methods:     []string{"GET", "POST", "OPTIONS"},
		},
	},
	routeLogin: {
		path: "/login",
		routeBase: routeBase{
			rateLimit:   5,
			rateBurst:   2,
			maxBodySize: 8196,
			methods:     []string{"GET", "POST", "OPTIONS"},
		},
	},

	routeYandexHomeHealth: {
		path: "/yandex/home/v1.0",
		routeBase: routeBase{
			rateLimit:   50,
			rateBurst:   10,
			maxBodySize: 256,
			methods:     []string{"GET", "OPTIONS"},
		},
	},
	routeYandexHomeUnlink: {
		path: "/yandex/home/v1.0/user/unlink",
		routeBase: routeBase{
			rateLimit:   10,
			rateBurst:   3,
			maxBodySize: 256,
			methods:     []string{"GET", "POST", "OPTIONS"},
		},
	},
	routeYandexHomeDevices: {
		path: "/yandex/home/v1.0/user/devices",
		routeBase: routeBase{
			rateLimit:   20,
			rateBurst:   5,
			maxBodySize: 256,
			methods:     []string{"GET", "POST", "OPTIONS"},
		},
	},
	routeYandexHomeQuery: {
		path: "/yandex/home/v1.0/user/devices/query",
		routeBase: routeBase{
			rateLimit:   0,
			rateBurst:   0,
			maxBodySize: 102400,
			methods:     []string{"POST", "OPTIONS"},
		},
	},
	routeYandexHomeAction: {
		path: "/yandex/home/v1.0/user/devices/action",
		routeBase: routeBase{
			rateLimit:   0,
			rateBurst:   0,
			maxBodySize: 512000,
			methods:     []string{"POST", "OPTIONS"},
		},
	},

	routeYandexDialogsTales: {
		path: "/yandex/dialogs/tales",
		routeBase: routeBase{
			rateLimit:   1000,
			rateBurst:   300,
			maxBodySize: 102400,
			methods:     []string{"POST", "OPTIONS"},
		},
	},

	routeAmazonAlexaWhistles: {
		path: "/amazon/alexa/whistles",
		routeBase: routeBase{
			rateLimit:   0,
			rateBurst:   0,
			maxBodySize: 102400,
			methods:     []string{"POST", "OPTIONS"},
		},
	},
}

func validateRoutePropertiesConfig(src RouteProperties, dest *routeBase, reportError func(msg string)) {
	if src.PropRateLimit() != "" {
		if rateLimit, rateBurst, err := parseRateLimit(src.PropRateLimit()); err != nil {
			reportError(fmt.Sprintf("invalid rateLimit value '%v': %v", src.PropRateLimit(), err))
		} else {
			dest.rateLimit = rateLimit
			dest.rateBurst = rateBurst
		}
	}

	if src.PropMaxBodySize() != "" {
		maxBodySize, err := parseSizeString(src.PropMaxBodySize())
		if err == nil && maxBodySize < 0 {
			err = fmt.Errorf("negative value not allowed")
		}
		if err != nil {
			reportError(fmt.Sprintf("invalid maxBodySize value '%v': %v", src.PropMaxBodySize(), err))
		} else {
			dest.maxBodySize = maxBodySize
		}
	}

	if src.PropMethods() != "" {
		if methods, err := parseRouteMethods(src.PropMethods()); err != nil {
			reportError(fmt.Sprintf("invalid methods value '%v': %v", src.PropMethods(), err))
		} else {
			dest.methods = methods
		}
	}

	dest.originAny = false
	if len(src.PropOriginExcludes()) == 0 && len(src.PropOriginIncludes()) == 1 {
		dest.originAny = src.PropOriginIncludes()[0] == "*"
	}
	if !dest.originAny {
		dest.originIncludes = make([]*regexp.Regexp, len(src.PropOriginIncludes()))
		dest.originExcludes = make([]*regexp.Regexp, len(src.PropOriginExcludes()))
		for i, val := range src.PropOriginIncludes() {
			re, err := regexp.Compile(val)
			if err != nil {
				reportError(fmt.Sprintf("invalid origin include regex '%v': %v", i, err))
			}
			dest.originIncludes[i] = re
		}
		for i, val := range src.PropOriginExcludes() {
			re, err := regexp.Compile(val)
			if err != nil {
				reportError(fmt.Sprintf("invalid origin exclude regex '%v': %v", i, err))
			}
			dest.originExcludes[i] = re
		}
	}

	dest.headers = src.PropHeaders()
	dest.allowCredentials = src.PropAllowCredentials()
}

func validateRouteConfig(cfgError configError) {
	if config.Routes == nil {
		return
	}

	for i, route := range *config.Routes {
		routeError := func(msg string) {
			cfgError(fmt.Sprintf("routes, route %v: %v", i, msg))
		}

		routeType, err := parseRouteType(route.Type)
		if err != nil {
			routeError(fmt.Sprintf("unknown type '%v'.", route.Type))
			continue
		}

		ri, ok := dedicatedRoutes[routeType]
		if !ok {
			routeError(fmt.Sprintf("internal error - dedicated type %v is not set.", route.Type))
			continue
		}

		if route.Path != "" {
			if path, err := parseRoutePath(route.Path); err != nil {
				routeError(fmt.Sprintf("invalid path '%v': %v", route.Path, err))
			} else {
				ri.path = path
			}
		}

		validateRoutePropertiesConfig(&route, &ri.routeBase, routeError)
	}
}

func parseRouteType(t string) (int, error) {
	switch strings.ToLower(t) {
	case "oauth-authorize":
		return routeOAuthAuthorize, nil
	case "oauth-token":
		return routeOAuthToken, nil
	case "yandex-home-health":
		return routeYandexHomeHealth, nil
	case "yandex-home-unlink":
		return routeYandexHomeUnlink, nil
	case "yandex-home-devices":
		return routeYandexHomeDevices, nil
	case "yandex-home-query":
		return routeYandexHomeQuery, nil
	case "yandex-home-action":
		return routeYandexHomeAction, nil
	}
	return 0, fmt.Errorf("unrecognized route type")
}

func parseRoutePath(path string) (string, error) {
	if path == "" {
		return "/", nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	if url.Path != path {
		return "", fmt.Errorf("path must contain URI path only")
	}
	return path, nil
}

func parseRateLimit(rateLimit string) (float64, int, error) {
	rateParts := strings.FieldsFunc(rateLimit, func(r rune) bool { return r == ',' || r == ';' })
	ratePartsLen := len(rateParts)
	if ratePartsLen < 0 || ratePartsLen > 2 {
		return 0, 0, fmt.Errorf("expected limit and optional burst value, comma or semicolon separated")
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

func dedicatedRoutePaths() map[string]struct{} {
	routes := make(map[string]struct{})
	for _, ri := range dedicatedRoutes {
		routes[ri.path] = struct{}{}
	}
	return routes
}

func handleDedicatedRoute(router *http.ServeMux, routeType int, handler http.Handler) {
	ri, ok := dedicatedRoutes[routeType]
	if !ok {
		panic(fmt.Sprintf("Unknown route type %v.", routeType))
	}

	handleRoute(router, ri, handler)
}

func maxBodySizeHandler(maxBodySize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitHandler(rateLimit float64, rateBurst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(rateLimit), rateBurst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func limitMethodsHandler(methods []string) func(http.Handler) http.Handler {
	methodsValue := strings.Join(methods, ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			notFound := true
			for _, method := range methods {
				if method == r.Method {
					notFound = false
					break
				}
			}
			if notFound {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			w.Header().Add("Access-Control-Allow-Methods", methodsValue)
			next.ServeHTTP(w, r)
		})
	}
}

func originMethodHandler(originAny bool, originIncludes, originExcludes []*regexp.Regexp) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if originAny || !skipByPatterns(
					originIncludes, originExcludes,
					[]string{origin},
					func(pattern *regexp.Regexp, value string) bool {
						return pattern.MatchString(value)
					},
				) {
					w.Header().Add("Access-Control-Allow-Origin", origin)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowsHeadersHandler(headers string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Headers", headers)
			next.ServeHTTP(w, r)
		})
	}
}

func allowCredentialsHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			next.ServeHTTP(w, r)
		})
	}
}

func optionsMethodHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rb *routeBase) applyHandlers(handler http.Handler) http.Handler {
	if rb.maxBodySize > 0 {
		handler = maxBodySizeHandler(rb.maxBodySize)(handler)
	}
	if rb.rateLimit > 0 {
		handler = rateLimitHandler(rb.rateLimit, rb.rateBurst)(handler)
	}
	if len(rb.methods) > 0 {
		handler = limitMethodsHandler(rb.methods)(handler)
	}
	if rb.originAny || len(rb.originIncludes) > 0 || len(rb.originExcludes) > 0 {
		handler = originMethodHandler(rb.originAny, rb.originIncludes, rb.originExcludes)(handler)
	}
	if rb.headers != "" {
		handler = allowsHeadersHandler(rb.headers)(handler)
	}
	if rb.allowCredentials {
		handler = allowCredentialsHandler()(handler)
	}
	return optionsMethodHandler()(handler)
}

func handleRoute(router *http.ServeMux, ri *routeInfo, handler http.Handler) {
	router.Handle(ri.path, ri.applyHandlers(handler))
}
