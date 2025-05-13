package log

import "go.uber.org/zap"

type Logger interface {
	Debug(msg string, args ...zap.Field)
	Info(msg string, args ...zap.Field)
	Error(msg string, args ...zap.Field)
	Fatal(msg string, args ...zap.Field)
	With(args ...zap.Field) Logger
	Sync() error
}
