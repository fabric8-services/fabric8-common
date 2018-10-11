package httpsupport

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"context"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
)

// Doer is a wrapper interface for goa client Doer
type HttpDoer interface {
	client.Doer
}

// HTTPClient defines the Do method of the http client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type configuration interface {
	IsPostgresDeveloperModeEnabled() bool
}

// HTTPClientDoer implements HttpDoer
type HTTPClientDoer struct {
	HTTPClient HTTPClient
}

// DefaultHttpDoer creates a new HttpDoer with default http client
func DefaultHttpDoer() HttpDoer {
	return &HTTPClientDoer{HTTPClient: http.DefaultClient}
}

// Do overrides Do method of the default goa client Doer. It's needed for mocking http clients in tests.
func (d *HTTPClientDoer) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return d.HTTPClient.Do(req)
}

// Host returns the host from the given request if run in prod mode or if config is nil
// and "auth.openshift.io" if run in dev mode
func Host(req *goa.RequestData, config configuration) string {
	if config != nil && config.IsPostgresDeveloperModeEnabled() {
		return "auth.openshift.io"
	}
	return req.Host
}

// ReadBody reads body from a ReadCloser and returns it as a string
func ReadBody(body io.ReadCloser) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CloseResponse reads the body and close the response. To be used to prevent file descriptor leaks.
func CloseResponse(response *http.Response) error {
	_, err := ioutil.ReadAll(response.Body)
	if err != io.EOF {
		return err
	}
	return response.Body.Close()
}

// AddParam adds a parameter to URL
func AddParam(urlString string, paramName string, paramValue string) (string, error) {
	return AddParams(urlString, map[string]string{paramName: paramValue})
}

// AddParams adds parameters to URL
func AddParams(urlString string, params map[string]string) (string, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	parameters := parsedURL.Query()
	for k, v := range params {
		parameters.Add(k, v)
	}
	parsedURL.RawQuery = parameters.Encode()

	return parsedURL.String(), nil
}

// AddTrailingSlashToURL adds a trailing slash to the URL if it doesn't have it already
// If URL is an empty string the function returns an empty string too
func AddTrailingSlashToURL(url string) string {
	if url != "" && !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}
