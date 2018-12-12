package httpsupport

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
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestProxy(t *testing.T) {
	suite.Run(t, &ProxyTestSuite{})
}

type ProxyTestSuite struct {
	testsuite.UnitTestSuite
}

func (s *ProxyTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()
	go startServer()
	waitForServer(s.T())
}

func (s *ProxyTestSuite) SetupTest() {
	inOurTests = true
}

func (s *ProxyTestSuite) TestSingleJoiningSlash() {
	assert.Equal(s.T(), "abc/xyz", singleJoiningSlash("abc", "xyz"))
	assert.Equal(s.T(), "abc/xyz", singleJoiningSlash("abc", "/xyz"))
}

func (s *ProxyTestSuite) TestProxyOK() {
	// do not suppress panics
	inOurTests = true
	s.checkProxyOK()
	// suppress panics
	inOurTests = false
	s.checkProxyOK()
}

func (s *ProxyTestSuite) checkProxyOK() {
	// GET with custom header and 201 response
	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.NoError(s.T(), err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(s.T(), err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})

	statusCtx := newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Del("Accept-Encoding")

	err = RouteHTTP(statusCtx, "http://localhost:8889")
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 201, rw.Code)
	assert.Equal(s.T(), "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body, err := ReadBody(rw.Result().Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), veryLongBody, body)

	// POST, gzipped, changed target path
	rw = httptest.NewRecorder()
	req, err = http.NewRequest("POST", u.String(), nil)
	require.NoError(s.T(), err)
	// Do not ignore panics
	req = req.WithContext(context.WithValue(context.Background(), http.ServerContextKey, "ProxyTest"))

	ctx = context.Background()
	goaCtx = goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx = newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Set("Accept-Encoding", "gzip")

	err = RouteHTTPToPath(statusCtx, "http://localhost:8889", "/api")
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 201, rw.Code)
	assert.Equal(s.T(), "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body, err = ReadBody(rw.Result().Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), veryLongBody, body)
}

func (s *ProxyTestSuite) TestProxyWithOptions() {
	// given
	r, err := recorder.New("proxy_whitebox_test")
	require.NoError(s.T(), err)
	//nolint
	defer r.Stop()
	u, err := url.Parse("http://domain.org/api/foo")
	require.NoError(s.T(), err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(s.T(), err)

	rw := httptest.NewRecorder()
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx := newStatusContext(goaCtx, req)
	statusCtx.Request.Header.Del("Accept-Encoding")
	// when
	err = RouteHTTP(statusCtx, "https://test", WithProxyTransport(r))
	// then
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 200, rw.Code)
	body, err := ReadBody(rw.Result().Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "ok!", body)
}

func (s *ProxyTestSuite) TestFailsIfResponseDataIsMissing() {
	// Missing ResponseData
	ctx := context.Background()
	err := RouteHTTP(ctx, "http://auth", nil)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "unable to get response from context", err.Error())
}

func (s *ProxyTestSuite) TestFailsIfInvalidTargetURL() {
	// Invalid URL
	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.NoError(s.T(), err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(s.T(), err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})

	err = RouteHTTP(goaCtx, "%@#", nil)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "parse %@: invalid URL escape \"%@\"", err.Error())
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
	for i := 0; i < 100; i++ {
		body = body + uuid.NewV4().String()
	}
	return body
}
