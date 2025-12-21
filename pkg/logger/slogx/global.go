package slogx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/lmittmann/tint"
)

var dl atomic.Pointer[Logger]

func InitGlobal(
	w io.Writer,
	logLevel string,
	pretty bool,
	extraHandlers ...func(slog.Handler) slog.Handler,
) error {
    level, err := ParseLevel(logLevel)
    if err != nil {
        return fmt.Errorf("init global logger: %v", err)
    }
    
	var handler slog.Handler
	if pretty {
		 handler = tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
		})
	} else {
        handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
           Level: level, 
        })
    }

    for _, eh := range extraHandlers {
        handler = eh(handler)
    }

	SetDefault(New(handler))

    return nil
}

func SetDefault(l *Logger) {
	dl.Store(l)
}

func Default() *Logger {
	return dl.Load()
}

func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().Info(ctx, msg, attrs...)
}

func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().Debug(ctx, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().Warn(ctx, msg, attrs...)
}

func Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	Default().Error(ctx, msg, attrs...)
}

func Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	Default().Log(ctx, level, msg, attrs...)
}
