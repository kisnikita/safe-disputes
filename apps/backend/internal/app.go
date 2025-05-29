package internal

import (
	"context"
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/repository/postgres"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	zapadapter "github.com/kisnikita/safe-disputes/backend/pkg/log/zap"
)

func StartApp() {
	logger := zapadapter.New()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Fatal("failed to load .env file", zap.Error(err))
	}

	db, err := postgres.NewConnection()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	repo, err := repository.New(db, logger)
	if err != nil {
		logger.Fatal("failed to create repository", zap.Error(err))
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_SECRET_TOKEN"))
	if err != nil {
		logger.Fatal("failed to create Telegram bot", zap.Error(err))
	}
	userData := checkChat(logger, bot)
	go updateChatID(logger, repo, userData)

	server := NewServer(logger, bot)
	server.RegisterRoutes(repo)
	go server.StartServer()

	gracefulShutdown(db, server, logger)
}

func gracefulShutdown(db *sql.DB, server *Server, logger log.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server Shutdown Failed", zap.Error(err))
	}

	logger.Info("Server exited")
	if err := db.Close(); err != nil {
		logger.Error("failed to close database connection", zap.Error(err))
	}
	logger.Info("application stopped")
}

type userChatData struct {
	Username string
	ChatID   int64
}

func checkChat(log log.Logger, bot *tgbotapi.BotAPI) chan userChatData {
	ch := make(chan userChatData)
	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := bot.GetUpdatesChan(ucfg)
	var (
		chatID   int64
		username string
	)
	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}
			chatID = update.Message.Chat.ID
			username = update.Message.From.UserName

			welcome := tgbotapi.NewMessage(chatID, "Привет! Я сохраню ваш chat_id и буду присылать уведомления.")
			if _, err := bot.Send(welcome); err != nil {
				log.Error("failed to send welcome: %v", zap.Error(err))
			}
			ch <- userChatData{ChatID: chatID, Username: username}
		}
	}()

	return ch
}

func updateChatID(log log.Logger, repo *repository.Repository, ch chan userChatData) {
	for {
		userData := <-ch
		ctx := context.Background()
		exist, err := repo.ExistByUsername(ctx, userData.Username)
		if err != nil {
			log.Error("failed to check if user exists", zap.String("username", userData.Username), zap.Error(err))
			continue
		}
		if !exist {
			u := models.NewUser(userData.Username)
			u.ChatID = userData.ChatID
			u.NotificationEnabled = true
			err := repo.InsertUser(ctx, u)
			if err != nil {
				log.Error("failed to insert new user", zap.String("username", userData.Username), zap.Int64("chatID", userData.ChatID), zap.Error(err))
				continue
			}
			log.Info("new user inserted", zap.String("username", userData.Username), zap.Int64("chatID", userData.ChatID))
			continue
		}

		err = repo.UpdateChatID(ctx, userData.ChatID, userData.Username)
		if err != nil {
			log.Error("failed to update chat ID", zap.String("username", userData.Username), zap.Int64("chatID", userData.ChatID), zap.Error(err))
		} else {
			log.Info("chat ID updated successfully", zap.String("username", userData.Username), zap.Int64("chatID", userData.ChatID))
		}
	}
}
