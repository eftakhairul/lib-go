# lib-go
Personal handy go libraries

## slog with Custom Handler
```go

ReplaceAttr := func(group []string, a slog.Attr) slog.Attr {
    if a.Key == "password" {
        return slog.Attr{}
    }

    return slog.Attr{Key: a.Key, Value: a.Value}
}
logger.NewJSONLogger(os.Stdout, loger.MyOptions{
    Level: slog.LevelDebug,
    ReplaceAttr: ReplaceAttr,
})
slog.SetDefault(logger)


ctx := ctx.WithValue(context.Background(), "requestID", "something")
logger.InfoContext(ctx, "testing first log message", slog.Attr{Key: "password", Value: "123456"})
```
