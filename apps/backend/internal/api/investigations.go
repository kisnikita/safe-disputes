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
	"time"
)

type InvestigationLister interface {
	ListInvestigation(ctx context.Context, opts models.InvestigationListOpts, username string) ([]models.Investigation, error)
}

type InvestigationGetter interface {
	GetInvestigation(ctx context.Context, id, username string) (models.Investigation, error)
}

type InvestigationVoter interface {
	VoteInvestigation(ctx context.Context, id, username, vote string) error
}

func ListInvestigations(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	investigationSrv, err := services.NewInvestigationService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create investigation service", zap.Error(err))
	}
	return listInvestigations(log, investigationSrv)
}

func listInvestigations(log log.Logger, lister InvestigationLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		creator, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		// --- parse query params ---
		statusStr := c.DefaultQuery("status", "open")
		status := models.InvestigationStatus(statusStr)

		limit := 10
		if limStr := c.Query("limit"); limStr != "" {
			if l, err := strconv.Atoi(limStr); err == nil && l > 0 {
				limit = l
			}
		}

		cursor := c.Query("cursor") // RFC3339 timestamp or empty

		opts := models.InvestigationListOpts{
			Status: &status,
			Limit:  limit,
			Cursor: cursor,
		}

		// --- fetch from repo ---
		investigations, err := lister.ListInvestigation(c, opts, creator)
		if err != nil {
			log.Error("ListInvestigations failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		log.Info("ListInvestigations cnt", zap.Int("count", len(investigations)))

		// --- prepare pagination response ---
		var nextCursor *string
		if len(investigations) > limit {
			// берем CreatedAt из (limit)-го индекса (0-based)
			ts := investigations[limit].CreatedAt.UTC().Format(time.RFC3339Nano)
			nextCursor = &ts
			investigations = investigations[:limit]
		}

		// --- map to DTO if needed (here возвращаем модели напрямую) ---
		c.JSON(http.StatusOK, gin.H{
			"data":       investigations,
			"nextCursor": nextCursor,
		})
	}
}

func GetInvestigation(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	investigationSrv, err := services.NewInvestigationService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create investigation service", zap.Error(err))
	}
	return getInvestigations(log, investigationSrv)
}

func getInvestigations(log log.Logger, getter InvestigationGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		user, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		invID := c.Param("id")
		if invID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "investigation ID is required"})
			return
		}

		inv, err := getter.GetInvestigation(c, invID, user)
		if err != nil {
			log.Error("GetInvestigation failed", zap.String("id", invID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": inv})
	}
}

func VoteInvestigation(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	investigationSrv, err := services.NewInvestigationService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create investigation service", zap.Error(err))
	}
	return voteInvestigations(log, investigationSrv)
}

func voteInvestigations(log log.Logger, voter InvestigationVoter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		user, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		invID := c.Param("id")
		if invID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "investigation ID is required"})
			return
		}

		vote := c.Query("vote")

		err := voter.VoteInvestigation(c, invID, user, vote)
		if err != nil {
			log.Error("GetInvestigation failed", zap.String("id", invID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
