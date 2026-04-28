package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type UserGetter interface {
	GetByUsername(ctx context.Context, username string) (models.User, error)
	GetTop(ctx context.Context, limit int) ([]models.User, error)
}

type UserUpdater interface {
	UpdateByUsername(ctx context.Context, opts models.UserUpdateOpts) error
}

func GetMe(repo *repository.Repository, log log.Logger) gin.HandlerFunc {
	userSrv, err := services.NewUserService(repo, log)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return getMe(log, userSrv)
}

func getMe(log log.Logger, getter UserGetter) gin.HandlerFunc {
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

		user, err := getter.GetByUsername(c.Request.Context(), username)
		if err != nil {
			log.Error("failed to get user", zap.String("username", username), zap.Error(err))
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": user})
	}
}

func UpdateUser(repo *repository.Repository, log log.Logger) gin.HandlerFunc {
	userSrv, err := services.NewUserService(repo, log)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return updateUser(log, userSrv)
}

func updateUser(log log.Logger, updater UserUpdater) gin.HandlerFunc {
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

		var req struct {
			NotificationEnabled      *bool   `json:"notificationEnabled"`
			DisputeReadiness         *bool   `json:"disputeReadiness"`
			InvestigationReadiness   *bool   `json:"investigationReadiness"`
			MinimumDisputeAmountNano *string `json:"minimumDisputeAmountNano"`
			Rating                   *int    `json:"rating"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		var minimumDisputeAmountNano *int64
		if req.MinimumDisputeAmountNano != nil {
			v, err := strconv.ParseInt(*req.MinimumDisputeAmountNano, 10, 64)
			if err != nil || v < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			minimumDisputeAmountNano = &v
		}

		opts := models.UserUpdateOpts{
			Username:                 username,
			NotificationEnabled:      req.NotificationEnabled,
			DisputeReadiness:         req.DisputeReadiness,
			InvestigationReadiness:   req.InvestigationReadiness,
			MinimumDisputeAmountNano: minimumDisputeAmountNano,
			Rating:                   req.Rating,
		}

		if err := updater.UpdateByUsername(c, opts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		}

		c.Status(http.StatusNoContent)
	}
}

func GetTop(repo *repository.Repository, log log.Logger) gin.HandlerFunc {
	userSrv, err := services.NewUserService(repo, log)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return getTop(log, userSrv)
}

func getTop(log log.Logger, getter UserGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := getter.GetTop(c, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top"})
		}

		c.JSON(http.StatusOK, gin.H{"data": users})
	}
}
