package slogx

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	start := time.Now()
    logger := Default()

	method := slog.String("method", info.FullMethod)
	logger.Info(ctx, "start handling grpc method", method)

	resp, err = handler(ctx, req)

	after := time.Since(start)

	durAttr := slog.Duration("duration", after)
	if err != nil {
		logger.Error(
			ctx,
			"finish with error",
			method,
			durAttr,
			Err(err),
		)
	} else {
		logger.Info(ctx, "finish success", method, durAttr)
	}

	return
}
