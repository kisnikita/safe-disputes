package api

import (
	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"os"
	"strings"
	"time"
)

func Middleware() gin.HandlerFunc {
	secretToken := os.Getenv("TELEGRAM_SECRET_TOKEN")
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(400, gin.H{"error": "missing Authorization header"})
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "tma" {
			c.JSON(400, gin.H{"error": "invalid Authorization header format"})
			c.Abort()
			return
		}
		initDataRaw := parts[1]

		// Parse and validate initData
		err := initdata.Validate(initDataRaw, secretToken, time.Hour*24)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid initData"})
			c.Abort()
			return
		}

		idata, err := initdata.Parse(initDataRaw)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to parse initData"})
			c.Abort()
			return
		}
		c.Set("username", idata.User.Username)
		c.Next()
	}
}
