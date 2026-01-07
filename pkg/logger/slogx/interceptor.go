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

func LoggingStreamInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	logger := Default()
	ctx := ss.Context()

	method := slog.String("method", info.FullMethod)
	logger.Info(ctx, "start handling grpc stream method", method)

	wrappedHandler := wrappedStream{logger: logger, ServerStream: ss}

	return handler(srv, &wrappedHandler)
}

type wrappedStream struct {
	logger *Logger
	grpc.ServerStream
}

func (w *wrappedStream) SendMsg(m any) error {
	w.logger.Info(w.Context(), "Intercepted server Send", slog.Any("message", m))

	return w.ServerStream.SendMsg(m)
}

func (w *wrappedStream) RecvMsg(m any) error {
	err := w.ServerStream.RecvMsg(m)

	w.logger.Info(w.Context(), "Intercepted server Recv", slog.Any("message", m))

	return err
}
