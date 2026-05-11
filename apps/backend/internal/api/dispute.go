package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type DisputePrechecker interface {
	PrecheckCreateDispute(ctx context.Context, opponent string, amountNano int64, actorUsername string) error
}

type DisputeCreator interface {
	CreateDispute(ctx context.Context, req models.CreateDisputeReq, actorUsername string) error
}

type DisputeLister interface {
	ListDisputes(ctx context.Context, opts models.DisputeListOpts, actorUsername string) ([]models.DisputeCard, error)
}

type DisputeSeener interface {
	MarkDisputesSeen(ctx context.Context, actorUsername string, disputeIDs []string) error
}

type DisputeGetter interface {
	GetDispute(ctx context.Context, disputeID string, actorUsername string) (models.DisputeDetails, error)
	GetDisputeForEvidence(ctx context.Context, disputeID string) (models.Dispute, error)
}

type DisputeAcceptor interface {
	AcceptDispute(ctx context.Context, disputeID string, acceptorUsername string, boc string) error
}

type DisputeRejector interface {
	RejectDispute(ctx context.Context, disputeID string, rejectorUsername string) error
}

type DisputeClaimer interface {
	ClaimDispute(ctx context.Context, disputeID string, claimerUsername string, boc string) error
}

type DisputeVoter interface {
	VoteDispute(ctx context.Context, disputeID string, claimerUsername string, win bool, boc string) error
}

func PrecheckDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	log = log.With(zap.String("handler", "PrecheckDispute"))
	return precheckDispute(log, disputeSrv)
}

func precheckDispute(log log.Logger, prechecker DisputePrechecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		var req struct {
			Opponent   string `json:"opponent" binding:"required"`
			AmountNano string `json:"amountNano" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("invalid request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		amountNano, err := models.ParsePositiveNano(req.AmountNano)
		if err != nil {
			log.Error("amountNano must be a positive integer", zap.String("amountNano", req.AmountNano), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err = prechecker.PrecheckCreateDispute(c, req.Opponent, amountNano, actorUsername)
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			log.Error("opponent not found", zap.String("opponent", req.Opponent), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "opponent not found"})
			return
		case errors.Is(err, services.ErrSelfOpponent):
			log.Error("creator and opponent must be different", zap.String("actor", actorUsername), zap.String("opponent", req.Opponent), zap.Error(err))
			c.JSON(http.StatusConflict, gin.H{"error": "creator and opponent must be different"})
			return
		case errors.Is(err, services.ErrMinimalAmount):
			log.Error("amount too less", zap.Int64("amountNano", amountNano), zap.Error(err))
			c.JSON(http.StatusConflict, gin.H{"error": "amount too less"})
			return
		case errors.Is(err, services.ErrUnready):
			log.Error("opponent not ready", zap.String("opponent", req.Opponent), zap.Error(err))
			c.JSON(http.StatusConflict, gin.H{"error": "opponent not ready"})
			return
		case err != nil:
			log.Error("failed to precheck dispute", zap.String("opponent", req.Opponent), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func CreateDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender,
	txMonitor services.TransactionMonitor,
) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	disputeSrv = disputeSrv.WithTransactionMonitor(txMonitor)
	log = log.With(zap.String("handler", "CreateDispute"))
	return createDispute(log, disputeSrv)
}

func createDispute(log log.Logger, disputeCreator DisputeCreator) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		var req models.CreateDisputeReq
		err := c.ShouldBind(&req)
		if err != nil {
			log.Error("invalid request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		log.Info("CreateDispute params", zap.Any("params", req))

		if req.ImageData, req.ImageType, err = getFile(c, "image"); err != nil {
			log.Error("failed to get file", zap.Error(err))
			c.JSON(500, gin.H{"error": "cannot open uploaded file"})
			return
		}

		if err = disputeCreator.CreateDispute(c, req, actorUsername); err != nil {
			handleApiError(c, log, actorUsername, err)
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
	log = log.With(zap.String("handler", "ListDisputes"))
	return listDisputes(log, disputeSrv)
}

func listDisputes(log log.Logger, lister DisputeLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

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

		cursor := c.Query("cursor")

		opts := models.DisputeListOpts{
			Status: &status,
			Result: resultPtr,
			Limit:  limit,
			Cursor: cursor,
		}

		disputes, err := lister.ListDisputes(c, opts, actorUsername)
		if err != nil {
			handleApiError(c, log, actorUsername, err)
			return
		}

		log.Info("ListDisputes cnt", zap.Int("count", len(disputes)))

		var nextCursor *string
		if len(disputes) > limit {
			ts := disputes[limit].CreatedAt.Format(time.RFC3339Nano)
			nextCursor = &ts
			disputes = disputes[:limit]
		}

		c.JSON(http.StatusOK, gin.H{
			"data":       disputes,
			"nextCursor": nextCursor,
		})
	}
}

func MarkDisputesSeen(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	log = log.With(zap.String("handler", "MarkDisputesSeen"))
	return markDisputesSeen(log, disputeSrv)
}

func markDisputesSeen(log log.Logger, seener DisputeSeener) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		var body struct {
			DisputeIDs []string `json:"disputeIDs"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if len(body.DisputeIDs) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		for _, rawID := range body.DisputeIDs {
			if _, err := uuid.Parse(strings.TrimSpace(rawID)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dispute id"})
				return
			}
		}

		if err := seener.MarkDisputesSeen(c, actorUsername, body.DisputeIDs); err != nil {
			handleApiError(c, log, actorUsername, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func GetDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	log = log.With(zap.String("handler", "GetDispute"))
	return getDispute(log, disputeSrv)
}

func getDispute(log log.Logger, getter DisputeGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- auth ---
		actorUsername, ok := getActorUsername(c)
		if !ok {
			return
		}

		disputeID := c.Param("id")
		if disputeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dispute ID is required"})
			return
		}

		dispute, err := getter.GetDispute(c, disputeID, actorUsername)
		if err != nil {
			handleApiError(c, log.With(zap.String("disputeID", disputeID)), actorUsername, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": dispute})
	}
}

func AcceptDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender,
	txMonitor services.TransactionMonitor,
) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	disputeSrv = disputeSrv.WithTransactionMonitor(txMonitor)
	log = log.With(zap.String("handler", "AcceptDispute"))
	return acceptDispute(log, disputeSrv)
}

func acceptDispute(log log.Logger, acceptor DisputeAcceptor) gin.HandlerFunc {
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
		boc := c.Query("boc")
		if boc == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "boc is required"})
			return
		}

		if err := acceptor.AcceptDispute(c, disputeID, actorUsername, boc); err != nil {
			handleApiError(c, log, actorUsername, err)
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
	log = log.With(zap.String("handler", "RejectDispute"))
	return rejectDispute(log, disputeSrv)
}

func rejectDispute(log log.Logger, rejector DisputeRejector) gin.HandlerFunc {
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

		if err := rejector.RejectDispute(c, disputeID, actorUsername); err != nil {
			handleApiError(c, log, actorUsername, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func ClaimDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender,
	txMonitor services.TransactionMonitor,
) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	disputeSrv = disputeSrv.WithTransactionMonitor(txMonitor)
	log = log.With(zap.String("handler", "ClaimDispute"))
	return refundDispute(log, disputeSrv)
}

func refundDispute(log log.Logger, claimer DisputeClaimer) gin.HandlerFunc {
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
		boc := c.Query("boc")
		if boc == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "boc is required"})
			return
		}

		if err := claimer.ClaimDispute(c, disputeID, actorUsername, boc); err != nil {
			handleApiError(c, log, actorUsername, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func VoteDispute(repo *repository.Repository, log log.Logger, sender services.MessageSender,
	txMonitor services.TransactionMonitor,
) gin.HandlerFunc {
	disputeSrv, err := services.NewDisputeService(repo, log, sender)
	if err != nil {
		log.Fatal("failed to create dispute service", zap.Error(err))
	}
	disputeSrv = disputeSrv.WithTransactionMonitor(txMonitor)
	log = log.With(zap.String("handler", "VoteDispute"))
	return voteDispute(log, disputeSrv)
}

func voteDispute(log log.Logger, voter DisputeVoter) gin.HandlerFunc {
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
		var body struct {
			Vote bool   `json:"vote"`
			Boc  string `json:"boc"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if err := voter.VoteDispute(c, disputeID, actorUsername, body.Vote, body.Boc); err != nil {
			handleApiError(c, log, actorUsername, err)
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
	log = log.With(zap.String("handler", "GetDisputeForEvidence"))
	return getDisputeForEvidence(log, disputeSrv)
}

func getDisputeForEvidence(log log.Logger, getter DisputeGetter) gin.HandlerFunc {
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

		dispute, err := getter.GetDisputeForEvidence(c, disputeID)
		if err != nil {
			handleApiError(c, log, actorUsername, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": dispute})
	}
}
