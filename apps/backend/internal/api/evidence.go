package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
	"io"
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
		// --- auth ---
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

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}
		// --- parse request body ---
		description := c.PostForm("description")

		var imageData []byte
		var imageType string

		// читаем файл из multipart
		if fileHeader, err := c.FormFile("evidence"); err == nil {
			file, err := fileHeader.Open()
			if err != nil {
				c.JSON(500, gin.H{"error": "cannot open uploaded file"})
				return
			}
			defer file.Close()

			buf, err := io.ReadAll(file)
			if err != nil {
				c.JSON(500, gin.H{"error": "cannot read uploaded file"})
				return
			}
			imageData = buf
			imageType = fileHeader.Header.Get("Content-Type")
		}

		req := models.EvidenceOpts{
			DisputeID:   disputeID,
			Username:    username,
			Description: description,
			ImageData:   imageData,
			ImageType:   imageType,
		}

		err := evidencer.ProvideEvidence(c, req)
		if err != nil {
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
		disputeID := c.Query("dispute_id")
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
