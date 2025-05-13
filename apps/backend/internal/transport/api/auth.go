package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

func TelegramAuth(repo *repository.Repository, log log.Logger) gin.HandlerFunc {
	_, err := services.NewUserService(repo, log)
	if err != nil {
		log.Fatal("failed to create user service", zap.Error(err))
	}
	// TODO: change secret-token to os.Getenv("TELEGRAM_SECRET_TOKEN")
	secretToken := "secret-token"
	return func(c *gin.Context) {
		// Read Authorization header: expected format "tma {initData}"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(400, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "tma" {
			c.JSON(400, gin.H{"error": "invalid Authorization header format"})
			return
		}
		initDataRaw := parts[1]
		fmt.Println("initDataRaw:", initDataRaw)

		// Parse and validate initData
		err := initdata.Validate(initDataRaw, secretToken, time.Hour*24)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid initData"})
			return
		}

		idata, err := initdata.Parse(initDataRaw)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to parse initData"})
			return
		}
		//err := userService.Insert(c, idata.User.Username)

		// data contains fields from Telegram InitData: User, AuthDate, etc.
		// TODO: lookup or create user in DB using data.User.ID

		// Return accessToken or session
		c.JSON(200, gin.H{
			"accessToken": "generated-jwt-or-session-token",
			"user":        idata.User.Username,
		})
	}
}
