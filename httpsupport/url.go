package httpsupport

import (
	"fmt"
	"strings"

	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/goadesign/goa"
)

// AbsoluteURL prefixes a relative URL with absolute address
// If config is not nil and run in dev mode then host is replaced by "auth.openshift.io"
func AbsoluteURL(req *goa.RequestData, relative string, config configuration) string {
	host := Host(req, config)
	return absoluteURLForHost(req, host, relative)
	// output: http://api.service.domain.org/somepath
}

// ReplaceDomainPrefixInAbsoluteURL replaces the last name in the host of the URL by a new name.
// Example: https://api.service.domain.org -> https://sso.service.domain.org
// If replaceBy == "" then return trim the last name.
// Example: https://api.service.domain.org -> https://service.domain.org
// Also prefixes a relative URL with absolute address
// If config is not nil and run in dev mode then "auth.openshift.io" is used as a host
func ReplaceDomainPrefixInAbsoluteURL(req *goa.RequestData, replaceBy, relative string, config configuration) (string, error) {
	host := Host(req, config)
	newHost, err := ReplaceDomainPrefix(host, replaceBy)
	if err != nil {
		return "", err
	}
	return absoluteURLForHost(req, newHost, relative), nil
}

func absoluteURLForHost(req *goa.RequestData, host, relative string) string {
	scheme := "http"
	if req.URL != nil && req.URL.Scheme == "https" { // isHTTPS
		scheme = "https"
	}
	xForwardProto := req.Header.Get("X-Forwarded-Proto")
	if xForwardProto != "" {
		scheme = xForwardProto
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, relative)
}

// ReplaceDomainPrefix replaces the last name in the host by a new name. Example: api.service.domain.org -> sso.service.domain.org
// If replaceBy == "" then return trim the last name. Example: api.service.domain.org -> service.domain.org
func ReplaceDomainPrefix(host string, replaceBy string) (string, error) {
	split := strings.SplitN(host, ".", 2)
	if len(split) < 2 {
		return host, errors.NewBadParameterError("host", host).Expected("must contain more than one domain")
	}
	if replaceBy == "" {
		return split[1], nil
	}
	return replaceBy + "." + split[1], nil
}
