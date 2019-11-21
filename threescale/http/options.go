package http

import (
	"context"
	"time"
)

// InstrumentationCB provides a callback hook into the client at response time to provide information
// about the underlying request to the remote host
type InstrumentationCB func(ctx context.Context, hostName string, statusCode int, requestDuration time.Duration)

// Option defines a callback function which is used to provide functional options to a request
type Option func(*Options)

// Options to provide optional behaviour to the standard APIs for Authorize, AuthRep and Report
type Options struct {
	context           context.Context
	instrumentationCB InstrumentationCB
}

// WithContext wraps the http transaction to 3scale backend with the provided context
func WithContext(ctx context.Context) Option {
	return func(args *Options) {
		args.context = ctx
	}
}

// WithInstrumentationCallback allows the caller to provide an optional callback function that will
// be called in a separate goroutine, with the details of the underlying request to 3scale if present as an option
func WithInstrumentationCallback(callback InstrumentationCB) Option {
	return func(options *Options) {
		options.instrumentationCB = callback
	}
}

// newOptions for 3scale backend
func newOptions(opts ...Option) *Options {
	options := &Options{context: context.TODO()}

	for _, opt := range opts {
		opt(options)
	}

	return options
}
