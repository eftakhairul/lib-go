package logger

import (
	"context"
	"log/slog"
	"os"
)

type MyHandler struct {
	slog.Handler
}

func (h *MyHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value("request_id").(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func NewMyHandler() *MyHandler {
	return &MyHandler{}
}

func (h *MyHandler) Enable(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

func (h *MyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &MyHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *MyHandler) WithGroup(name string) slog.Handler {
	return &MyHandler{Handler: h.Handler.WithGroup(name)}
}

func NewJSONLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	MyHandler := MyHandler{Handler: jsonHandler}

	return slog.New(&MyHandler)
}
