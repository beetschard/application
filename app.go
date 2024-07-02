package application

import (
	"context"
	"errors"
	"github.com/beetschard/application/internal/pkg/router"
	"github.com/jessevdk/go-flags"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var _ context.Context = (*Context[any])(nil)

const DefaultApplicationTimeout = time.Second * 5

type (
	API     = router.API
	Handler = router.Handler
)

type (
	APIFunc[Args any] func(Context[Args]) (API, error)
	GoFunc[Args any]  func(Context[Args]) error
	Context[Args any] struct {
		stored *storedContext[Args]
		Args   struct {
			Args    Args
			Network string `long:"network" description:"network to serve on" default:"tcp" env:"APPLICATION_NETWORK"`
			Address string `long:"address" description:"Address to serve on" default:"0.0.0.0:5443" env:"APPLICATION_ADDRESS"`
		}
	}
	storedContext[Args any] struct {
		context context.Context
		opts    options[Args]
		cancel  context.CancelFunc
		wg      sync.WaitGroup
		started atomic.Bool
	}
)

func Run[Args any](opt ...Option[Args]) {
	exit := 0
	defer func() { os.Exit(exit) }()

	ctx, cancel := newContext[Args]()
	defer cancel()

	ctx.applyOpts(getOptions[Args](opt))
	ctx.parseArgs()

	errs := make(chan error)
	done := make(chan struct{})

	var wg sync.WaitGroup
	for _, fn := range ctx.stored.opts.goFuncs {
		wg.Add(1)
		go func(fn GoFunc[Args]) {
			defer wg.Done()
			if err := fn(*ctx); err != nil {
				errs <- err
			}
		}(fn)
	}

	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case err := <-errs:
		cancel()
		slog.Info("waiting for all goroutines to exit")
		appWaitTimeoutErr(done, errs, err, ctx.stored.opts.timeout)
		exit = 1
	case <-done:
		slog.Info("all goroutines finished, goodbye")
	}
}

func appWaitTimeoutErr(done <-chan struct{}, errs <-chan error, err error, timeout time.Duration) {
	c := (<-chan time.Time)(make(chan time.Time))
	if timeout > 0 {
		c = time.After(timeout)
	}

	for {
		slog.Error("application returned an error", "error", err)

		select {
		case <-done:
			slog.Info("all goroutines finished, goodbye")
			return
		case <-c:
			slog.Info("goroutines did not finish after timeout", "timeout", timeout)
			return
		case err = <-errs:
		}
	}
}

func getOptions[Args any](opt []Option[Args]) []Option[Args] {
	defaultPrefix := []Option[Args]{
		OptionApplicationTimeout[Args](DefaultApplicationTimeout),
	}
	defaultSuffix := []Option[Args]{
		OptionGoFunc[Args](runRouter[Args]),
	}
	return append(append(defaultPrefix, opt...), defaultSuffix...)
}

func newContext[Args any]() (*Context[Args], context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context[Args]{
		stored: &storedContext[Args]{
			context: ctx,
			cancel:  cancel,
		},
	}, cancel
}

func (ctx *Context[Args]) applyOpts(opt []Option[Args]) {
	for _, opt := range opt {
		opt(&ctx.stored.opts)
	}
}

func (ctx *Context[Args]) parseArgs() {
	parser := flags.NewParser(&ctx.Args, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if e := new(flags.Error); errors.As(err, &e) {
			if errors.Is(e.Type, flags.ErrHelp) {
				os.Exit(1)
			}
		}
		slog.Error("failed to parse args", "error", err)
		os.Exit(1)
	}
}

func runRouter[Args any](ctx Context[Args]) error {
	if len(ctx.stored.opts.apis) == 0 {
		return nil
	}

	r := router.New()
	for _, api := range ctx.stored.opts.apis {
		api, err := api(ctx)
		if err != nil {
			slog.Error("failed to initialize api", "error", err)
			os.Exit(1)
		}
		r.AddAPI(api)
	}

	return r.ServeHTTP(&ctx, ctx.Args.Network, ctx.Args.Address)
}

func (ctx *Context[Args]) Deadline() (deadline time.Time, ok bool) {
	return ctx.stored.context.Deadline()
}
func (ctx *Context[Args]) Done() <-chan struct{} { return ctx.stored.context.Done() }
func (ctx *Context[Args]) Err() error            { return ctx.stored.context.Err() }
func (ctx *Context[Args]) Value(key any) any     { return ctx.stored.context.Value(key) }
