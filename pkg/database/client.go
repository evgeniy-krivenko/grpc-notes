package database

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

type logger interface {
	Warn(context.Context, string, ...slog.Attr)
}

//go:generate go run github.com/kazhuravlev/options-gen/cmd/options-gen@v0.33.2 -out-filename=client_options.gen.go -from-struct=Options
type Options struct {
	address  string `option:"mandatory" validate:"required,hostname_port"`
	username string `option:"mandatory" validate:"required"`
	password string `option:"mandatory" validate:"required"`
	database string `option:"mandatory" validate:"required"`

	retry         bool `default:"true"`
	retryAttempts uint `default:"1" validate:"min=1,max=10"`

	logger logger

	// TODO: use in pool later
	maxOpenConns int `default:"5" validate:"max=20"`
	maxIdleConns int `default:"5" validate:"max=20"`
}

func NewPGX(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options for pgx: %v", err)
	}

	if opts.logger == nil {
		opts.logger = noopLogger{}
	}

	ds := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(opts.username, opts.password),
		Host:   opts.address,
		Path:   opts.database,
	}

	pool, err := pgxpool.New(ctx, ds.String())
	if err != nil {
		return nil, fmt.Errorf("open new pgx pool: %v", err)
	}

	if !opts.retry {
		return pool, pool.Ping(ctx)
	}

	if err := retry.Do(
		func() error { return pool.Ping(ctx) },
		retry.Delay(time.Millisecond*300),
		retry.Attempts(opts.retryAttempts),
		retry.OnRetry(func(attempt uint, err error) {
			opts.logger.Warn(
				ctx,
				"failed ping to database",
				slog.Any("err", err),
				slog.Uint64("attempt", uint64(attempt)),
			)
		}),
	); err != nil {
		return nil, fmt.Errorf("ping to database: %v", err)
	}

	return pool, nil
}

type noopLogger struct{}

func (n noopLogger) Warn(context.Context, string, ...slog.Attr) {}
