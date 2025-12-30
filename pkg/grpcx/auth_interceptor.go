package grpcx

import (
	"context"
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var mockToken = "Bearer 304346c6-34ee-4624-a85e-a1b198626158"

func AuthInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (res any, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "md from incoming request")
	}

	headers := md.Get("authorization")

	if !slices.Contains(headers, mockToken) {
		return nil, status.Error(
			codes.Unauthenticated,
			"metadata doesn't contain correct authorization token",
		)
	}

	return handler(ctx, req)
}
