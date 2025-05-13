package zap

import (
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type Logger struct {
	*zap.Logger
}

func (l *Logger) With(fields ...zap.Field) log.Logger {
	if len(fields) == 0 {
		return l
	}

	return &Logger{Logger: l.Logger.With(fields...)}
}

func (l *Logger) Debug(msg string, args ...zap.Field) {
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...zap.Field) {
	l.Logger.Info(msg, args...)
}

func (l *Logger) Error(msg string, args ...zap.Field) {
	l.Logger.Error(msg, args...)
}

func (l *Logger) Fatal(msg string, args ...zap.Field) {
	l.Logger.Fatal(msg, args...)
}

func (l *Logger) Sync() error {
	err := l.Logger.Sync()
	if err == nil {
		return nil
	}

	var errs error
	for _, e := range multierr.Errors(err) {
		if strings.Contains(e.Error(), "sync /dev/stdout") {
			continue
		}
		errs = multierr.Append(errs, e)
	}

	return errs
}

func New() *Logger {
	zapLogger, _ := zap.NewDevelopment()

	return &Logger{Logger: zapLogger}
}
