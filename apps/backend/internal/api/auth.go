package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type userCreator interface {
	CreateIfNotExist(ctx context.Context, username string) error
}

func TelegramAuth(repo *repository.Repository, log log.Logger) gin.HandlerFunc {
	userSrv, err := services.NewUserService(repo, log)
	if err != nil {
		log.Fatal("failed to create user service", zap.Error(err))
	}
	return telegramAuth(log, userSrv)
}

func telegramAuth(log log.Logger, userSrv userCreator) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		username, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		err := userSrv.CreateIfNotExist(c, username)
		if err != nil {
			log.Error("failed to create user", zap.String("username", username), zap.Error(err))
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}

		c.Status(200)
	}
}
