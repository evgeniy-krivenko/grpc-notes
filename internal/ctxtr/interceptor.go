package ctxtr

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"

var ErrUserNotFound =  errors.New("user not found")

func MockAuthInterceptor(userID int64) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req any,
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (any, error) {
        ctx = context.WithValue(ctx, UserIDKey, userID)
        return handler(ctx, req)
    }
}

func UserID(ctx context.Context) (int64, error) {
    userID, ok := ctx.Value(UserIDKey).(int64)
    if !ok {
        return 0, ErrUserNotFound
    }

    return userID, nil
}
