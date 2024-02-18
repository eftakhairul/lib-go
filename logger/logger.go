package logger

import (
	"context"
	"io"
	"log/slog"
	"time"
)

const RequestIdKey = "requestID"

// MyOptions is a customized [HandlerOptions] that specifies the options for a [MyHandler].
type MyOptions struct {
	slog.HandlerOptions

	// Time format (Default: time.StampMilli)
	TimeFormat string
}

// MyHandler is a customized [Handler] that writes Records to an [io.Writer]
type MyHandler struct {
	slog.Handler
	TimeFormat string
}

func (h *MyHandler) Handle(ctx context.Context, r slog.Record) error {

	timeAttr := slog.Attr{
		Key:   slog.TimeKey,
		Value: slog.StringValue(r.Time.Format(h.TimeFormat)),
	}
	r.AddAttrs(timeAttr)

	if id, ok := ctx.Value(RequestIdKey).(string); ok {
		r.AddAttrs(slog.String(RequestIdKey, id))
	}
	return h.Handler.Handle(ctx, r)
}

// NewMyHandler creates a new [MyHandler] that writes to w.
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

// NewJSONLogger creates a new customized [Logger] that writes to w in JSON format.
func NewJSONLogger(w io.Writer, opts *MyOptions) *slog.Logger {
	if opts == nil {
		opts = &MyOptions{}
	}

	var timeFormat string
	if opts.TimeFormat == "" {
		timeFormat = opts.TimeFormat
	} else {
		timeFormat = time.RFC1123
	}

	// if opts.ReplaceAttr == nil {
	// 	opts.ReplaceAttr = func(_ []string, a slog.Attr) slog.Attr { return a }
	// }

	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: opts.ReplaceAttr,
	})
	MyHandler := MyHandler{Handler: jsonHandler, TimeFormat: timeFormat}

	return slog.New(&MyHandler)
}
