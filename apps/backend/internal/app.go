package internal

import (
	"context"
	"database/sql"
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

	db, err := postgres.NewConnection()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	repo, err := repository.New(db, logger)
	if err != nil {
		logger.Fatal("failed to create repository", zap.Error(err))
	}

	server := NewServer(logger)
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
