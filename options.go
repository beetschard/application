package application

import "time"

type (
	Option[Args any]  func(*options[Args])
	options[Args any] struct {
		apis    []APIFunc[Args]
		goFuncs []GoFunc[Args]
		timeout time.Duration
	}
)

func OptionAPI[Args any](api APIFunc[Args]) Option[Args] {
	return func(opts *options[Args]) {
		if api != nil {
			opts.apis = append(opts.apis, api)
		}
	}
}

func OptionGoFunc[Args any](fn GoFunc[Args]) Option[Args] {
	return func(opts *options[Args]) {
		if fn != nil {
			opts.goFuncs = append(opts.goFuncs, fn)
		}
	}
}

func OptionApplicationTimeout[Args any](to time.Duration) Option[Args] {
	return func(opts *options[Args]) {
		opts.timeout = to
	}
}
