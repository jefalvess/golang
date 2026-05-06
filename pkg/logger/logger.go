package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.SugaredLogger
)

func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := config.Build()
	if err != nil {
		panic(err)
	}
	Logger = l.Sugar()
}

func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}
