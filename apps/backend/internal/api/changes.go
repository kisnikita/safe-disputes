package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type ChangesLister interface {
	ListChanges(ctx context.Context, since time.Time, actorUsername string,
	) (models.ChangesList, models.ChangesUnreadCounts, error)
}

func ListChanges(repo *repository.Repository, logger log.Logger) gin.HandlerFunc {
	changesSrv, err := services.NewChangesService(repo, logger)
	if err != nil {
		logger.Fatal("failed to create changes service", zap.Error(err))
	}
	return listChanges(logger, changesSrv)
}

func listChanges(logger log.Logger, lister ChangesLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		sinceRaw := c.Query("since")
		if sinceRaw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "since query param is required"})
			return
		}
		since, err := time.Parse(time.RFC3339Nano, sinceRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since param"})
			return
		}

		changes, unreadCounts, err := lister.ListChanges(c, since, actorUsername)
		if err != nil {
			logger.Error("ListChanges failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		nextSince := changes.MaxUpdatedAt
		if nextSince == "" {
			nextSince = sinceRaw
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"disputes":       changes.Disputes,
				"investigations": changes.Investigations,
			},
			"unreadCounts": unreadCounts,
			"nextSince":    nextSince,
		})
	}
}
