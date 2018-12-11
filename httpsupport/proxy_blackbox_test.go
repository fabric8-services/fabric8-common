package httpsupport_test

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-common/test/recorder"

	"github.com/fabric8-services/fabric8-common/httpsupport"

	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	go startServer()
	waitForServer(t)

	// GET with custom header and 201 response
	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.NoError(t, err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})

	statusCtx := newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Del("Accept-Encoding")

	err = httpsupport.RouteHTTP(statusCtx, "http://localhost:8889")
	require.NoError(t, err)

	assert.Equal(t, 201, rw.Code)
	assert.Equal(t, "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body, err := httpsupport.ReadBody(rw.Result().Body)
	require.NoError(t, err)
	assert.Equal(t, veryLongBody, body)

	// POST, gzipped, changed target path
	rw = httptest.NewRecorder()
	req, err = http.NewRequest("POST", u.String(), nil)
	require.NoError(t, err)

	ctx = context.Background()
	goaCtx = goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx = newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Set("Accept-Encoding", "gzip")

	err = httpsupport.RouteHTTPToPath(statusCtx, "http://localhost:8889", "/api")
	require.NoError(t, err)

	assert.Equal(t, 201, rw.Code)
	assert.Equal(t, "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body, err = httpsupport.ReadBody(rw.Result().Body)
	require.NoError(t, err)
	assert.Equal(t, veryLongBody, body)
}

func TestProxyWithOptions(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	r, err := recorder.New("proxy_blackbox_test")
	require.NoError(t, err)
	//nolint
	defer r.Stop()
	u, err := url.Parse("http://domain.org/api/foo")
	require.NoError(t, err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx := newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Del("Accept-Encoding")
	// when
	err = httpsupport.RouteHTTP(statusCtx, "https://test", httpsupport.WithProxyTransport(r))
	// then
	require.NoError(t, err)
	assert.Equal(t, 200, rw.Code)
	body, err := httpsupport.ReadBody(rw.Result().Body)
	require.NoError(t, err)
	assert.Equal(t, "ok!", body)
}

func TestFailsIfResponseDataIsMissing(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// Missing ResponseData
	ctx := context.Background()
	err := httpsupport.RouteHTTP(ctx, "http://auth", nil)
	require.Error(t, err)
	assert.Equal(t, "unable to get response from context", err.Error())
}

func TestFailsIfInvalidTargetURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// Invalid URL
	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.NoError(t, err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})

	err = httpsupport.RouteHTTP(goaCtx, "%@#", nil)
	require.Error(t, err)
	assert.Equal(t, "parse %@: invalid URL escape \"%@\"", err.Error())
}

type statusContext struct {
	context.Context
	*goa.ResponseData
	*goa.RequestData
}

func newStatusContext(ctx context.Context, r *http.Request) *statusContext {
	resp := goa.ContextResponse(ctx)
	req := goa.ContextRequest(ctx)
	req.Request = r
	return &statusContext{Context: ctx, ResponseData: resp, RequestData: req}
}

func startServer() {
	http.HandleFunc("/api", handlerGzip)
	err := http.ListenAndServe(":8889", nil)
	if err != nil {
		panic(err)
	}
}

func waitForServer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8889/api", nil)
	require.NoError(t, err)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		client := &http.Client{Timeout: time.Duration(500 * time.Millisecond)}
		res, err := client.Do(req)
		if err == nil && res.StatusCode == 201 {
			return
		}
	}
	assert.Fail(t, "failed to start server")
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Custom-Test-Header", "proxyTest")
	w.WriteHeader(201)
	fmt.Fprint(w, veryLongBody)
}

func handlerGzip(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		handler(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	handler(gzr, r)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

var veryLongBody = generateLongBody()

func generateLongBody() string {
	body := uuid.NewV4().String()
	for i := 0; i < 10000; i++ {
		body = body + uuid.NewV4().String()
	}
	return body
}
