package log

import "context"

type ctxKey int

const loggerKey ctxKey = iota

var defaultLogger Logger = &nopLogger{}

func IntoContext(ctx context.Context, logger Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return defaultLogger
	}
	if l, ok := ctx.Value(loggerKey).(Logger); ok && l != nil {
		return l
	}
	return defaultLogger
}
