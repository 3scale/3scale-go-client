package threescale

import "context"

// WithContext wraps the http transaction to 3scale backend with the provided context
func WithContext(ctx context.Context) Option {
	return func(args *Transaction) {
		args.context = ctx
	}
}

// WithContext embeds the provided extensions in the http transaction to 3scale
// https://github.com/3scale/apisonator/blob/v2.96.2/docs/extensions.md
func WithExtensions(extensions Extensions) Option {
	return func(args *Transaction) {
		args.extensions = extensions
	}
}
