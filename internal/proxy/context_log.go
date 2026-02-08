package proxy

import (
	"context"

	"go.uber.org/zap"
)

type key struct{}

var logKey key

func NewContext(ctx context.Context, log *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, logKey, log)
}

func FromContext(ctx context.Context) (*zap.SugaredLogger, bool) {
	log, ok := ctx.Value(logKey).(*zap.SugaredLogger)
	return log, ok
}
