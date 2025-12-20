package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/evgeniy-krivenko/grpc-notes/internal/api/notes"
	"github.com/evgeniy-krivenko/grpc-notes/internal/config"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run app: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

    cfg, err := config.Parse()
    if err != nil {
        return fmt.Errorf("parse cfg: %v", err)
    }

    notesSvc := notes.New()

	srv, err := grpcx.New(grpcx.NewOptions(
		cfg.GRPC.Addr,
		grpcx.WithServices(notesSvc),
	))
	if err != nil {
		return fmt.Errorf("init grpc server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return srv.Run(ctx) })

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}
