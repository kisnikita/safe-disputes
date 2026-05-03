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
)

type DisputeEvidencer interface {
	ProvideEvidence(ctx context.Context, evidence models.EvidenceOpts) error
}

type EvidenceGetter interface {
	GetEvidences(ctx context.Context, disputeID string) ([]models.Evidence, error)
}

func EvidenceDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewEvidenceService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return evidenceDispute(log, disputeSrv)
}

func evidenceDispute(log log.Logger, evidencer DisputeEvidencer) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}
		description := c.PostForm("description")
		if description == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "description is required"})
			return
		}

		data, extension, err := getFile(c, "evidence")
		if err != nil {
			log.Error("failed to get file", zap.Error(err))
			c.JSON(500, gin.H{"error": "cannot open uploaded file"})
			return 
		} 

		req := models.EvidenceOpts{
			DisputeID:   disputeID,
			Username:    actorUsername,
			Description: description,
			ImageData:   data,
			ImageType:   extension,
		}

		if err := evidencer.ProvideEvidence(c, req); err != nil {
			log.Error("ProvideEvidence failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func GetEvidencesByDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewEvidenceService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return getEvidencesByDispute(log, disputeSrv)
}

func getEvidencesByDispute(log log.Logger, getter EvidenceGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		disputeID := c.Query("disputeID")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		evidences, err := getter.GetEvidences(c, disputeID)
		if err != nil {
			log.Error("GetEvidences failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": evidences})
	}
}
