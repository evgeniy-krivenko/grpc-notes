package gwserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 3 * time.Second
)

type Logger interface {
	Info(context.Context, string, ...slog.Attr)
}

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options -all-variadic true
type Options struct {
	addr    string       `option:"mandatory" validate:"hostname_port"`
	handler http.Handler `option:"mandatory" validate:"required"`

	middlewares []func(http.Handler) http.Handler
	logger      Logger
}

type Server struct {
	Options
	srv *http.Server
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate gw server opts: %v", err)
	}

	handler := opts.handler

	for _, md := range opts.middlewares {
		handler = md(handler)
	}

	// c := cors.New(cors.Options{
	// 	AllowedOrigins: opts.allowedOrigins,
	// 	AllowedMethods: []string{
	// 		http.MethodGet,
	// 		http.MethodPost,
	// 		http.MethodDelete,
	// 		http.MethodPut,
	// 	},
	// })
	//
	// handler := c.Handler(opts.mux)
	// handler = wsproxy.WebsocketProxy(handler)

	srv := &http.Server{
		Addr:              opts.addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return &Server{Options: opts, srv: srv}, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		return s.srv.Shutdown(ctx)
	})

	eg.Go(func() error {
		if s.logger != nil {
			s.logger.Info(ctx, "listen and serve", slog.String("addr", s.addr))
		}

		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %v", err)
		}

		return nil
	})

	return eg.Wait()
}
