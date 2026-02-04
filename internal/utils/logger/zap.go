package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() *zap.SugaredLogger {
	conf := zap.NewDevelopmentConfig()
	conf.DisableStacktrace = true
	conf.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := conf.Build()

	sugar := logger.Sugar()
	defer sugar.Sync()

	return sugar
}
