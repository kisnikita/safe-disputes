package api

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
)

type DisputeCreator interface {
	CreateDispute(ctx context.Context, dispute models.Dispute, creatorUsername string) error
}

type DisputeLister interface {
	ListDisputes(ctx context.Context, opts models.DisputeListOpts, creatorUsername string) ([]models.Dispute, error)
}

type DisputeGetter interface {
	GetDispute(ctx context.Context, disputeID string, creatorUsername string) (models.Dispute, error)
	GetDisputeForEvidence(ctx context.Context, disputeID string) (models.Dispute, error)
}

type DisputeAcceptor interface {
	AcceptDispute(ctx context.Context, disputeID string, acceptorUsername string) error
}

type DisputeRejector interface {
	RejectDispute(ctx context.Context, disputeID string, acceptorUsername string) error
}

type DisputeClaimer interface {
	ClaimDispute(ctx context.Context, disputeID string, claimerUsername string) error
}

type DisputeVoter interface {
	VoteDispute(ctx context.Context, disputeID string, claimerUsername string, win bool) error
}

func CreateDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return createDispute(log, disputeSrv)
}

func createDispute(log log.Logger, disputeCreator DisputeCreator) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		title := c.PostForm("title") // title is empty
		description := c.PostForm("description")
		opponent := c.PostForm("opponent")
		amountStr := c.PostForm("amount")
		amount, _ := strconv.ParseInt(amountStr, 10, 32)

		var imageData []byte
		var imageType string

		// читаем файл из multipart
		if fileHeader, err := c.FormFile("image"); err == nil {
			file, err := fileHeader.Open()
			if err != nil {
				c.JSON(500, gin.H{"error": "cannot open uploaded file"})
				return
			}
			defer file.Close()

			// считываем всё в []byte
			buf, err := io.ReadAll(file)
			if err != nil {
				c.JSON(500, gin.H{"error": "cannot read uploaded file"})
				return
			}
			imageData = buf
			imageType = fileHeader.Header.Get("Content-Type") // например "image/jpeg"
		}

		dispute := models.NewDispute(title, description, opponent, int(amount), imageData, imageType)
		err := disputeCreator.CreateDispute(c, dispute, creator)
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			log.Error("opponent not found", zap.String("opponent", opponent), zap.Error(err))
			c.JSON(404, gin.H{"error": "opponent not found"})
			return
		case errors.Is(err, services.ErrMinimalAmount):
			log.Error("amount too less", zap.Int64("amount", amount), zap.Error(err))
			c.JSON(400, gin.H{"error": "amount too less"})
			return
		case errors.Is(err, services.ErrUnready):
			log.Error("opponent not ready", zap.String("opponent", opponent), zap.Error(err))
			c.JSON(400, gin.H{"error": "opponent not ready"})
			return
		case err != nil:
			log.Error("failed to create dispute", zap.String("title", title), zap.String("opponent", opponent), zap.Error(err))
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}

		c.Status(201)
	}
}

func ListDisputes(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return listDisputes(log, disputeSrv)
}

func listDisputes(log log.Logger, lister DisputeLister) gin.HandlerFunc {
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
		statusStr := c.DefaultQuery("status", "current")
		status := models.Status(statusStr)

		var resultPtr *bool
		if resStr, ok := c.GetQuery("result"); ok {
			if b, err := strconv.ParseBool(resStr); err == nil {
				resultPtr = &b
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid result param"})
				return
			}
		}

		limit := 10
		if limStr := c.Query("limit"); limStr != "" {
			if l, err := strconv.Atoi(limStr); err == nil && l > 0 {
				limit = l
			}
		}

		cursor := c.Query("cursor") // RFC3339 timestamp or empty

		opts := models.DisputeListOpts{
			Status: &status,
			Result: resultPtr,
			Limit:  limit,
			Cursor: cursor,
		}

		// --- fetch from repo ---
		disputes, err := lister.ListDisputes(c, opts, creator)
		if err != nil {
			log.Error("ListDisputes failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		log.Info("ListDisputes cnt", zap.Int("count", len(disputes)))

		// --- prepare pagination response ---
		var nextCursor *string
		if len(disputes) > limit {
			// берем CreatedAt из (limit)-го индекса (0-based)
			ts := disputes[limit].CreatedAt.UTC().Format(time.RFC3339Nano)
			nextCursor = &ts
			disputes = disputes[:limit]
		}

		// --- map to DTO if needed (here возвращаем модели напрямую) ---
		c.JSON(http.StatusOK, gin.H{
			"data":       disputes,
			"nextCursor": nextCursor,
		})
	}
}

func GetDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return getDispute(log, disputeSrv)
}

func getDispute(log log.Logger, getter DisputeGetter) gin.HandlerFunc {
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

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		dispute, err := getter.GetDispute(c, disputeID, creator)
		if err != nil {
			log.Error("GetDispute failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": dispute})
	}
}

func AcceptDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return acceptDispute(log, disputeSrv)
}

func acceptDispute(log log.Logger, acceptor DisputeAcceptor) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		acceptorUsername, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		err := acceptor.AcceptDispute(c, disputeID, acceptorUsername)
		if err != nil {
			log.Error("AcceptDispute failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func RejectDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return rejectDispute(log, disputeSrv)
}

func rejectDispute(log log.Logger, rejector DisputeRejector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		u, exist := c.Get("username")
		if !exist {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		acceptorUsername, ok := u.(string)
		if !ok {
			c.JSON(400, gin.H{"error": "invalid username"})
			return
		}

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		err := rejector.RejectDispute(c, disputeID, acceptorUsername)
		if err != nil {
			log.Error("RejectDispute failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func ClaimDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return claimDispute(log, disputeSrv)
}

func claimDispute(log log.Logger, claimer DisputeClaimer) gin.HandlerFunc {
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

		err := claimer.ClaimDispute(c, disputeID, username)
		if err != nil {
			log.Error("ClaimDispute failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func VoteDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return voteDispute(log, disputeSrv)
}

func voteDispute(log log.Logger, voter DisputeVoter) gin.HandlerFunc {
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
		var body struct {
			Vote bool `json:"vote"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := voter.VoteDispute(c, disputeID, username, body.Vote)
		if err != nil {
			log.Error("ClaimDispute failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func GetDisputeForEvidence(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	return getDisputeForEvidence(log, disputeSrv)
}

func getDisputeForEvidence(log log.Logger, getter DisputeGetter) gin.HandlerFunc {
	return func(c *gin.Context) {

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		dispute, err := getter.GetDisputeForEvidence(c, disputeID)
		if err != nil {
			log.Error("GetDisputeForEvidence failed", zap.String("id", disputeID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": dispute})
	}
}
