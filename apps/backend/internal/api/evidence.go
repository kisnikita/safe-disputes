package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type DisputeEvidencer interface {
	ProvideEvidence(ctx context.Context, evidence models.EvidenceOpts) error
}

type EvidenceGetter interface {
	GetEvidences(ctx context.Context, disputeID string) ([]models.Evidence, error)
}

func ProvideEvidence(repo *repository.Repository, log log.Logger, sender services.MessageSender,
	txMonitor services.TransactionMonitor,
) gin.HandlerFunc {
	disputeSrv, err := services.NewEvidenceService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	disputeSrv = disputeSrv.WithTransactionMonitor(txMonitor)
	log = log.With(zap.String("handler", "ProvideEvidence"))
	return provideEvidence(log, disputeSrv)
}

func provideEvidence(log log.Logger, evidencer DisputeEvidencer) gin.HandlerFunc {
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
		boc := c.PostForm("boc")
		if boc == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "boc is required"})
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
			Boc:         boc,
			Description: description,
			ImageData:   data,
			ImageType:   extension,
		}

		if err := evidencer.ProvideEvidence(c, req); err != nil {
			handleApiError(c, log, actorUsername, err)
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
	log = log.With(zap.String("handler", "GetEvidencesByDispute"))
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
			handleApiError(c, log, "", err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": evidences})
	}
}
