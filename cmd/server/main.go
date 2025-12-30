package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	notesapi "github.com/evgeniy-krivenko/grpc-notes/internal/api/notes"
	"github.com/evgeniy-krivenko/grpc-notes/internal/config"
	"github.com/evgeniy-krivenko/grpc-notes/internal/ctxtr"
	"github.com/evgeniy-krivenko/grpc-notes/internal/repository"
	notesusecase "github.com/evgeniy-krivenko/grpc-notes/internal/usecase/notes"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/database"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
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

	if err := slogx.InitGlobal(
		os.Stdout,
		cfg.App.LogLevel,
		cfg.App.Pretty,
	); err != nil {
		return fmt.Errorf("init logger: %v", err)
	}

	logger := slogx.Default()

	db, err := database.NewPGX(ctx,
		database.NewOptions(
			fmt.Sprintf("%s:%s", cfg.Database.Host, cfg.Database.Port),
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Name,
			database.WithLogger(logger),
			database.WithRetry(false),
		))
	if err != nil {
		return fmt.Errorf("init database: %v", err)
	}

	repo := repository.New(db)

	notesUsecase, err := notesusecase.New(notesusecase.NewOptions(repo))
	if err != nil {
		return fmt.Errorf("init notes usecase: %v", err)
	}

	notesSvc, err := notesapi.New(notesapi.NewOptions(notesUsecase))
	if err != nil {
		return fmt.Errorf("init notes api: %v", err)
	}

	srv, err := grpcx.New(grpcx.NewOptions(
		cfg.GRPC.Addr,
		grpcx.WithLogger(logger),
		grpcx.WithServices(notesSvc),
		grpcx.WithGrpcOptions(
			grpc.ChainUnaryInterceptor(
				ctxtr.MockAuthInterceptor(1),
				grpcx.AuthInterceptor,
				slogx.LoggingInterceptor,
			),
			grpc.MaxConcurrentStreams(cfg.GRPC.MaxConcurrentStreams),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				Time:    cfg.GRPC.KeepaliveTime,
				Timeout: cfg.GRPC.KeepaliveTimeout,
			}),
		),
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
