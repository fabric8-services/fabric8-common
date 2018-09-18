package sentry

import (
	"context"
	"os"

	"github.com/fabric8-services/fabric8-common/log"

	"github.com/getsentry/raven-go"
)

// client encapsulates client to Sentry service
// also has mutex which controls access to the client
type client struct {
	c       *raven.Client
	sendErr chan func()
	userInfo func(ctx context.Context) (*raven.User, error)
}

var (
	sentryClient *client
)

// Sentry returns client declared inside package
func Sentry() *client {
	return sentryClient
}

// InitializeSentryClient initializes sentry client. This function returns
// function that can be used to close the sentry client and error.
// sentryDSN param is optional. If null then DSN set via SENTRY_DSN env var will be used
func InitializeSentryClient(sentryDSN *string, options ...func(*client)) (func(), error) {
	var dsn string
	if sentryDSN!=nil {
		dsn = *sentryDSN
	} else {
		dsn = os.Getenv("SENTRY_DSN")
	}
	c, err := raven.New(dsn)
	if err != nil {
		return nil, err
	}
	sentryClient = &client{
		c:       c,
		sendErr: make(chan func()),
	}
	// set all options passed by user
	for _, opt := range options {
		opt(sentryClient)
	}

	// wait on errors to be sent on channel of client object
	go sentryClient.loop()
	return func() {
		close(sentryClient.sendErr)
	}, nil
}

// WithUser helps to set user context
func WithUser(userInfo func(ctx context.Context) (*raven.User, error)) func(*client) {
	return func(c *client) {
		c.userInfo = userInfo
	}
}

// WithRelease helps you set release/commit of currently running
// code while initializing sentry client using function InitializeSentryClient
func WithRelease(release string) func(*client) {
	return func(c *client) {
		c.c.SetRelease(release)
	}
}

// WithEnvironment helps you set environment the deployed code is
// running in while initializing sentry client using function
// InitializeSentryClient
func WithEnvironment(env string) func(*client) {
	return func(c *client) {
		c.c.SetEnvironment(env)
	}
}

// waits on functions to be sent on channel
// which are then executed
func (c *client) loop() {
	for op := range c.sendErr {
		op()
	}
}

// CaptureError sends error 'err' to Sentry, meanwhile also sets user
// information by extracting user information from the context provided
func (c *client) CaptureError(ctx context.Context, err error) {
	// if method called during test which has uninitialized client
	if c == nil {
		return
	}
	// Extract user information. Ignoring error here but then before using the
	// object user make sure to check if it wasn't nil.
	var user *raven.User
	if c.userInfo!=nil {
		user, _ = c.userInfo(ctx)
	}
	reqID := log.ExtractRequestID(ctx)

	c.sendErr <- func() {
		if user != nil {
			c.c.SetUserContext(user)
		}

		additionalContext := make(map[string]string)
		if reqID != "" {
			additionalContext["req_ID"] = reqID
		}

		c.c.CaptureError(err, additionalContext)
		c.c.ClearContext()
	}
}
