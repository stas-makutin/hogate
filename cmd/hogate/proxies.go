package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type proxy HTTPProxy

func validateProxyConfig(cfgError configError) {
	routes := dedicatedRoutePaths()
	for i, prx := range config.Proxies {
		proxyError := func(msg string) {
			cfgError(fmt.Sprintf("proxies, proxy %v: %v", i, msg))
		}

		p := (*proxy)(prx)

		if err := p.valid(routes); err != nil {
			proxyError(err.Error())
		}

		validateRoutePropertiesConfig(prx, &prx.routeBase, proxyError)
	}
}

func addProxyRoutes(router *http.ServeMux) map[string]struct{} {
	routes := dedicatedRoutePaths()
	for _, prx := range config.Proxies {
		p := (*proxy)(prx)
		if err := p.valid(routes); err == nil {

			scope := parseScope(p.Scope)
			p.parsedScope = make([]string, len(scope))
			i := 0
			for k := range scope {
				p.parsedScope[i] = k
				i++
			}

			prx.preparedPrefix = prx.Prefix
			if prx.preparedPrefix == "" {
				prx.preparedPrefix = prx.Route
			}
			prx.preparedPrefix = strings.TrimSuffix(prx.preparedPrefix, "/")

			var handler http.Handler = p.makeProxy()
			if prx.preparedPrefix != "" {
				handler = http.StripPrefix(prx.preparedPrefix, handler)
			}

			router.Handle(prx.Route, authorizationHandlerWithFlags(prx.applyHandlers(handler), prx.Flags, prx.parsedScope...))

			routes[p.Route] = struct{}{}
		}
	}
	return routes
}

func (p *proxy) valid(routes map[string]struct{}) error {
	if p.Route == "" {
		return fmt.Errorf("route is empty")
	}
	if _, ok := routes[p.Route]; ok || p.Route == "" {
		return fmt.Errorf("the route '%s' already exists", p.Route)
	}
	if p.Target == "" {
		return fmt.Errorf("target is empty")
	}
	if _, err := url.Parse(p.Target); err != nil {
		return fmt.Errorf("target is invalid: %v", err)
	}
	return nil
}

func (p *proxy) rewriteLocation(loc string, targetURL *url.URL, outReq *http.Request) string {
	u, err := url.Parse(loc)
	if err != nil {
		return loc
	}

	// If the Location header is an absolute URL, check if it points to the target host
	if u.IsAbs() {
		if u.Host != targetURL.Host {
			return loc // External redirect (e.g. OAuth provider), leave untouched
		}
		// Convert to public scheme & host using X-Forwarded-* headers set by r.SetURL()
		if proto := outReq.Header.Get("X-Forwarded-Proto"); proto != "" {
			u.Scheme = proto
		} else {
			u.Scheme = "https"
		}
		if fwdHost := outReq.Header.Get("X-Forwarded-Host"); fwdHost != "" {
			u.Host = fwdHost
		}
	}

	// prepend mountPrefix if not already present (e.g. "/home" -> "/srv/home")
	if p.preparedPrefix != "" && strings.HasPrefix(u.Path, "/") && !strings.HasPrefix(u.Path, p.preparedPrefix+"/") && u.Path != p.preparedPrefix {
		u.Path = p.preparedPrefix + u.Path
	}

	return u.String()
}

func (p *proxy) rewriteSetCookie(rawCookie string) string {
	parts := strings.Split(rawCookie, ";")
	var rewritten []string

	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		lower := strings.ToLower(trimmed)

		// keep the first part (cookie Name=Value) untouched
		if i == 0 {
			rewritten = append(rewritten, trimmed)
			continue
		}

		// strip internal LAN Domain so browser scopes cookie to public host
		if strings.HasPrefix(lower, "domain=") {
			continue
		}

		// scope backend cookie Path under prefix (e.g. Path=/ -> Path=/srv)
		if p.preparedPrefix != "" && strings.HasPrefix(lower, "path=/") {
			origPath := trimmed[len("path="):]
			if !strings.HasPrefix(origPath, p.preparedPrefix) {
				trimmed = "Path=" + p.preparedPrefix + origPath
			}
		}

		rewritten = append(rewritten, trimmed)
	}

	return strings.Join(rewritten, "; ")
}

func (p *proxy) makeProxy() http.Handler {
	targetURL, _ := url.Parse(p.Target)

	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Host = targetURL.Host
			if r.In.Header.Get("Origin") != "" {
				r.Out.Header.Set("Origin", targetURL.String())
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if loc := resp.Header.Get("Location"); loc != "" {
				resp.Header.Set("Location", p.rewriteLocation(loc, targetURL, resp.Request))
			}

			if rawCookies := resp.Header.Values("Set-Cookie"); len(rawCookies) > 0 {
				resp.Header.Del("Set-Cookie")
				for _, raw := range rawCookies {
					resp.Header.Add("Set-Cookie", p.rewriteSetCookie(raw))
				}
			}

			if loc := resp.Header.Get("Location"); strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, p.Route) {
				resp.Header.Set("Location", strings.TrimSuffix(p.Route, "/")+loc)
			}
			return nil
		},
	}
}
