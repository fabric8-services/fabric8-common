package gocksupport

import (
	"net/http"

	gock "gopkg.in/h2non/gock.v1"
)

// SpyOnCalls checks the number of calls
func SpyOnCalls(counter *int) gock.Matcher {
	matcher := gock.NewBasicMatcher()
	matcher.Add(spyOnCallsMatchFunc(counter))
	return matcher
}

func spyOnCallsMatchFunc(counter *int) gock.MatchFunc {
	return func(req *http.Request, _ *gock.Request) (bool, error) {
		*counter++
		return true, nil
	}
}
