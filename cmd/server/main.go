package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"buf.build/go/protovalidate"
	protovalidateic "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	openapi "github.com/evgeniy-krivenko/grpc-notes/docs/api/notes/v1"
	notesapi "github.com/evgeniy-krivenko/grpc-notes/internal/api/notes"
	"github.com/evgeniy-krivenko/grpc-notes/internal/config"
	"github.com/evgeniy-krivenko/grpc-notes/internal/ctxtr"
	"github.com/evgeniy-krivenko/grpc-notes/internal/repository"
	notesusecase "github.com/evgeniy-krivenko/grpc-notes/internal/usecase/notes"
	gw "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/database"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/gwserver"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
	"github.com/evgeniy-krivenko/grpc-notes/third_party/swagger"
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

	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create protovalidator: %v", err)
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

	gwSrv, err := buildGWServer(ctx, &cfg)
	if err != nil {
		return fmt.Errorf("build gateway server: %v", err)
	}

	swaggerSrv, err := buildSwaggerServer(&cfg)
	if err != nil {
		return fmt.Errorf("build swagger server: %v", err)
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
				protovalidateic.UnaryServerInterceptor(validator),
			),
			grpc.ChainStreamInterceptor(
				slogx.LoggingStreamInterceptor,
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
	eg.Go(func() error { return gwSrv.Run(ctx) })
	eg.Go(func() error { return swaggerSrv.Run(ctx) })

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}

func buildGWServer(ctx context.Context, cfg *config.Config) (*gwserver.Server, error) {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := gw.RegisterNoteAPIHandlerFromEndpoint(ctx, mux, cfg.GRPC.Addr, opts); err != nil {
		return nil, fmt.Errorf("register grpc gateway: %v", err)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		},
	})

	wsMiddleware := func(h http.Handler) http.Handler {
		return wsproxy.WebsocketProxy(h)
	}

	return gwserver.New(gwserver.NewOptions(
		cfg.HTTP.Addr,
		mux,
		gwserver.WithMiddlewares(corsMiddleware.Handler, wsMiddleware),
		gwserver.WithLogger(slogx.Default()),
	))
}

func buildSwaggerServer(cfg *config.Config) (*gwserver.Server, error) {
	mux := http.NewServeMux()

	swaggerStaticsHandler := http.StripPrefix("/swagger", http.FileServer(http.FS(swagger.Content)))
	mux.Handle("GET /swagger/", swaggerStaticsHandler)

	swaggerSpecsHandler := http.StripPrefix("/swagger/specs", http.FileServer(http.FS(openapi.Content)))
	mux.Handle("GET /swagger/specs/", swaggerSpecsHandler)

	return gwserver.New(gwserver.NewOptions(
		cfg.SwaggerHTTP.Addr,
		mux,
		gwserver.WithLogger(slogx.Default()),
	))
}
