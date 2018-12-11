package httpsupport

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/fabric8-services/fabric8-common/log"

	"github.com/goadesign/goa"
	"github.com/pkg/errors"
)

// RouteHTTPToPath uses a reverse proxy to route the http request to the scheme, host provided in targetHost
// and path provided in targetPath.
//
// Usage example in Goa controller (listen to http://localhost:8080) if the request is to proxy to Auth service:
//  err := proxy.RouteHTTP(ctx, "http://auth", "/api/status")
//	if err != nil {
//		return jsonapi.JSONErrorResponse(ctx, err)
//	}
// In the example above any request to http://localhost:8080/test?id=xyz will be routed to http://auth/api/status?id=xyz
func RouteHTTPToPath(ctx context.Context, targetHost string, targetPath string) error {
	return route(ctx, targetHost, &targetPath)
}

// RouteHTTP uses a reverse proxy to route the http request to the scheme, host, and base path provided in target.
// If the target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
//
// Usage example in Goa controller (listen to http://localhost:8080) if the request is to proxy to Auth service:
//  err := proxy.RouteHTTP(ctx, "http://auth")
//	if err != nil {
//		return jsonapi.JSONErrorResponse(ctx, err)
//	}
// In the example above any request to http://localhost:8080/status?id=xyz will be routed to http://auth/status?id=xyz
func RouteHTTP(ctx context.Context, target string, options ...HTTPProxyOption) error {
	return route(ctx, target, nil, options...)
}

// HTTPProxyOption an option to customiwze the HTTP proxy
type HTTPProxyOption func(proxy *httputil.ReverseProxy)

// WithProxyTransport an option to customize the proxy with the given roundtripper
func WithProxyTransport(r http.RoundTripper) HTTPProxyOption {
	return func(proxy *httputil.ReverseProxy) {
		proxy.Transport = r
	}
}

func route(ctx context.Context, targetHost string, targetPath *string, options ...HTTPProxyOption) error {
	rw := goa.ContextResponse(ctx)
	if rw == nil {
		return errors.New("unable to get response from context")
	}
	req := goa.ContextRequest(ctx)
	if req == nil {
		return errors.New("unable to get request from context")
	}

	targetURL, err := url.Parse(targetHost)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":         err,
			"target_host": targetHost,
			"request_uri": req.RequestURI,
		}, "unable to parse target host")
		return err
	}

	director := newDirector(ctx, req, targetURL, targetPath)
	proxy := &httputil.ReverseProxy{Director: director}
	// configure the proxy with the options
	for _, opt := range options {
		opt(proxy)
	}

	proxy.ServeHTTP(rw, req.Request)

	return nil
}

func newDirector(ctx context.Context, originalRequestData *goa.RequestData, target *url.URL, targetPath *string) func(*http.Request) {
	targetQuery := target.RawQuery
	return func(req *http.Request) {
		// Get the original request URL for info log
		scheme := "http"
		if req.URL != nil && req.URL.Scheme == "https" { // isHTTPS
			scheme = "https"
		}
		xForwardProto := req.Header.Get("X-Forwarded-Proto")
		if xForwardProto != "" {
			scheme = xForwardProto
		}
		originalReq := &url.URL{Scheme: scheme, Host: originalRequestData.Host, Path: req.URL.Path, RawQuery: req.URL.RawQuery}

		// Modify the request to route to a new target
		if target.Scheme == "" {
			req.URL.Scheme = "http"
		} else {
			req.URL.Scheme = target.Scheme
		}
		req.URL.Host = target.Host
		if targetPath != nil {
			req.URL.Path = *targetPath
		} else {
			req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		}
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		requestID := log.ExtractRequestID(ctx)
		if requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}

		// Log the original and target URLs
		originalReqString := originalReq.String()
		targetReqString := req.URL.String()
		log.Info(ctx, map[string]interface{}{
			"original_req_url": originalReqString,
			"target_req_url":   targetReqString,
			"target":           target,
			"target_string":    target.String(),
		}, "Routing %s to %s", originalReqString, targetReqString)
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
