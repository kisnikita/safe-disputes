package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

func getActorUsername(c *gin.Context) (string, bool) {
	v, ok := c.Get("username")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	actorUsername, ok := v.(string)
	if !ok || actorUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username"})
		return "", false
	}

	return actorUsername, true
}

func getFile(c *gin.Context, name string) ([]byte, string, error) {
	fileHeader, err := c.FormFile(name)
	switch {
	case errors.Is(err, http.ErrMissingFile):
		return nil, "", nil
	case err != nil:
		return nil, "", err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	extension := fileHeader.Header.Get("Content-Type")

	return data, extension, nil
}

func handleApiError(c *gin.Context, logger log.Logger, actor string, err error) {
	baseLogger := logger.With(zap.String("actor", actor), zap.Error(err))
	switch {
	case handleTxServiceError(c, baseLogger, err):
	case handleValidationError(c, baseLogger, err):
	default:
		handleInternalError(c, baseLogger, err)
	}
}

func handleTxServiceError(c *gin.Context, log log.Logger, err error) bool {
	switch {
	case errors.Is(err, services.ErrInvalidBOC):
		log.Error("invalid transaction boc")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction boc"})
		return true
	case errors.Is(err, services.ErrTxNotFinalized):
		log.Error("transaction not finalized in time")
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "transaction not finalized in time"})
		return true
	case errors.Is(err, services.ErrTxFailed):
		log.Error("transaction failed")
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return true
	case errors.Is(err, services.ErrTxMonitorUnavailable):
		log.Error("transaction monitor unavailable")
		c.JSON(http.StatusBadGateway, gin.H{"error": "transaction monitor unavailable"})
		return true
	default:
		return false
	}
}

func handleValidationError(c *gin.Context, log log.Logger, err error) bool {
	switch {
	case errors.Is(err, services.ErrValidation):
		log.Error("failed to validate dispute")
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed"})
		return true
	default:
		return false
	}
}

func handleInternalError(c *gin.Context, log log.Logger, err error) {
	log.Error("internal server error")
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
