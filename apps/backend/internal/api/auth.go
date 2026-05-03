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
	CreateIfNotExist(ctx context.Context, username string, photoUrl *string) error
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
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		var photoUrl *string
		if photo, ok := c.Get("photoUrl"); ok {
			photoStr, ok := photo.(string)
			if !ok {
				c.JSON(400, gin.H{"error": "invalid photo url"})
				return
			}
			if photoStr != "" {
				photoUrl = &photoStr
			}
		}

		err := userSrv.CreateIfNotExist(c, actorUsername, photoUrl)
		if err != nil {
			log.Error("failed to create user", zap.String("username", actorUsername), zap.Error(err))
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}

		c.Status(200)
	}
}
