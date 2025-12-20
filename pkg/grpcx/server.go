package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type logger interface {
	Info(ctx context.Context, msg string, attrs ...slog.Attr)
}

type Service interface {
	RegisterService(grpc.ServiceRegistrar)
}

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options -all-variadic true
type Options struct {
	addr     string    `option:"mandatory" validate:"required,hostname_port"`
	services []Service `validate:"required,min=1"`

	logger logger

	grpcOptions []grpc.ServerOption

	maxConnIdle time.Duration `default:"5m"`
	time        time.Duration `default:"2h"`
	timeout     time.Duration `default:"20s"`
}

type Server struct {
	opts   Options
	srv    *grpc.Server
	logger logger
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("grpc server validate: %v", err)
	}

	if opts.logger == nil {
		opts.logger = &noopLogger{}
	}

	opts.grpcOptions = append(opts.grpcOptions,
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: opts.maxConnIdle,
			Time:              opts.time,
			Timeout:           opts.timeout,
		}),
	)

	srv := grpc.NewServer(
		opts.grpcOptions...,
	)

	for _, svc := range opts.services {
		svc.RegisterService(srv)
	}

	return &Server{opts: opts, srv: srv}, nil
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.opts.addr)
	if err != nil {
		return fmt.Errorf("run grpc: %v", err)
	}

	go func() {
		<-ctx.Done()
		s.srv.GracefulStop()
	}()

	s.opts.logger.Info(
		ctx,
		"run grpc server",
		slog.String("addr", s.opts.addr),
	)

	if err := s.srv.Serve(listener); err != nil && err != grpc.ErrServerStopped {
		return fmt.Errorf("listen and server: %v", err)
	}

	return nil
}

type noopLogger struct{}

func (n *noopLogger) Info(
	ctx context.Context,
	msg string,
	attrs ...slog.Attr,
) {
}
