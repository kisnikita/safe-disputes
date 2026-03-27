package services

import (
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...zap.Field) {}
func (noopLogger) Info(string, ...zap.Field)  {}
func (noopLogger) Error(string, ...zap.Field) {}
func (noopLogger) Fatal(string, ...zap.Field) {}
func (noopLogger) With(...zap.Field) log.Logger {
	return noopLogger{}
}
func (noopLogger) Sync() error { return nil }

type fakeMessageSender struct {
	err      error
	calls    int
	chatIDs  []int64
	messages []string
}

func (f *fakeMessageSender) SendMessage(chatID int64, text string) error {
	f.calls++
	f.chatIDs = append(f.chatIDs, chatID)
	f.messages = append(f.messages, text)
	return f.err
}
