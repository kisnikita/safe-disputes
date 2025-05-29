package services

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type MessageService struct {
	logger log.Logger
	Bot    *tgbotapi.BotAPI
}

func NewMessageService(log log.Logger, bot *tgbotapi.BotAPI) MessageService {
	return MessageService{
		logger: log,
		Bot:    bot,
	}
}

func (s MessageService) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.Bot.Send(msg)
	if err != nil {
		s.logger.Error("failed to send message", zap.Int64("chatID", chatID), zap.String("text", text), zap.Error(err))
		return err
	}
	return nil
}
