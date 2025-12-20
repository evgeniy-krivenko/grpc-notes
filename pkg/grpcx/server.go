package grpcx

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Service interface {
	RegisterService(grpc.ServiceRegistrar)
}

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options -all-variadic true
type Options struct {
	addr     string    `option:"mandatory" validate:"required,hostname_port"`
	services []Service `validate:"required,min=1"`

	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor

	maxConnIdle time.Duration `default:"5m"`
	time        time.Duration `default:"2h"`
	timeout     time.Duration `default:"20s"`
}

type Server struct {
	opts Options
	srv  *grpc.Server
}

func New(opts Options) (*Server, error) {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(opts.unaryInterceptors...),
		grpc.ChainStreamInterceptor(opts.streamInterceptors...),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: opts.maxConnIdle,
			Time:              opts.time,
			Timeout:           opts.timeout,
		}),
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

	if err := s.srv.Serve(listener); err != nil && err != grpc.ErrServerStopped {
		return fmt.Errorf("listen and server: %v", err)
	}

	return nil
}
