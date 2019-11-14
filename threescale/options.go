package threescale

import (
	"context"
)

// WithContext wraps the http transaction to 3scale backend with the provided context
func WithContext(ctx context.Context) Option {
	return func(args *Options) {
		args.context = ctx
	}
}

// WithExtensions embeds the provided extensions in the http transaction to 3scale
// https://github.com/3scale/apisonator/blob/v2.96.2/docs/extensions.md
func WithExtensions(extensions Extensions) Option {
	return func(args *Options) {
		args.extensions = extensions
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
