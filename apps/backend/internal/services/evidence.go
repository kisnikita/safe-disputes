package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type EvidenceCreator interface {
	InsertEvidence(ctx context.Context, evidence models.Evidence) error
}

type EvidenceChecker interface {
	IsFirstEvidence(ctx context.Context, disputeID string) (bool, error)
}

type EvidenceGetter interface {
	GetEvidences(ctx context.Context, disputeID uuid.UUID) ([]models.Evidence, error)
}

type EvidenceBroadcaster interface {
	BroadcastInvestigation(ctx context.Context, u2i models.User2Investigation, p1, p2 uuid.UUID) ([]uuid.UUID, error)
}

type EvidenceService struct {
	logger log.Logger

	evidenceCreator      EvidenceCreator
	evidenceChecker      EvidenceChecker
	evidenceGetter       EvidenceGetter
	userFinder           UserFinder
	u2dUpdater           User2DisputeUpdater
	u2dGetter            User2DisputeGetter
	opponentGetter       OpponentGetter
	investigationCreator InvestigationCreator
	evidenceBroadcaster  EvidenceBroadcaster
	disputesFinder       DisputeFinder
	msgSender            MessageSender
}

func NewEvidenceService(repo *repository.Repository, log log.Logger, msgSender MessageSender) (EvidenceService, error) {
	if repo == nil {
		return EvidenceService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return EvidenceService{}, fmt.Errorf("logger is nil")
	}

	return EvidenceService{
		logger:               log,
		evidenceCreator:      repo,
		evidenceChecker:      repo,
		evidenceGetter:       repo,
		userFinder:           repo,
		u2dUpdater:           repo,
		u2dGetter:            repo,
		opponentGetter:       repo,
		investigationCreator: repo,
		evidenceBroadcaster:  repo,
		disputesFinder:       repo,
		msgSender:            msgSender,
	}, nil
}

func (s EvidenceService) ProvideEvidence(ctx context.Context, opts models.EvidenceOpts) error {
	if opts.DisputeID == "" || opts.Username == "" {
		return fmt.Errorf("invalid opts data: disputeID, username and imageData must be provided")
	}

	// --- GETTERS ---

	disputeUUID, err := uuid.Parse(opts.DisputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	user, err := s.userFinder.GetUserByUsername(ctx, opts.Username)
	if err != nil {
		return fmt.Errorf("failed to get user by username: %w", err)
	}

	evidence := models.NewEvidence(disputeUUID, user.ID, opts.Description, opts.ImageData, opts.ImageType)

	u2d, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user2dispute: %w", err)
	}

	isFirst, err := s.evidenceChecker.IsFirstEvidence(ctx, opts.DisputeID)
	if err != nil {
		return fmt.Errorf("failed to check if first evidence: %w", err)
	}

	// --- FIRST EVIDENCE ---
	if isFirst {
		if err := s.evidenceCreator.InsertEvidence(ctx, evidence); err != nil {
			return fmt.Errorf("failed to insert first evidence: %w", err)
		}
		result := models.DisputesResultEvidenceAnswered
		optsU2D := models.U2DUpdateOpts{
			ID:     u2d.ID,
			Result: &result,
		}
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, optsU2D); err != nil {
			return fmt.Errorf("failed to update user2dispute result: %w", err)
		}
		return nil
	}

	// --- SECOND EVIDENCE ---

	// --- INSERT EVIDENCE AND UPDATE USERS ---
	if err := s.evidenceCreator.InsertEvidence(ctx, evidence); err != nil {
		return fmt.Errorf("failed to insert first evidence: %w", err)
	}

	result := models.DisputesResultInspected
	optsU2D := models.U2DUpdateOpts{
		ID:     u2d.ID,
		Result: &result,
	}
	if err := s.u2dUpdater.UpdateUser2Dispute(ctx, optsU2D); err != nil {
		return fmt.Errorf("failed to update user2dispute result: %w", err)
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}
	u2dOp, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user2dispute: %w", err)
	}
	optsU2D.ID = u2dOp.ID
	if err := s.u2dUpdater.UpdateUser2Dispute(ctx, optsU2D); err != nil {
		return fmt.Errorf("failed to update user2dispute result: %w", err)
	}

	// --- CREATE INVESTIGATION ---
	total, err := s.userFinder.GetTotalUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get total users: %w", err)
	}

	dispute, err := s.disputesFinder.GetDisputeByID(ctx, disputeUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute by ID: %w", err)
	}

	investigation := models.NewInvestigation(disputeUUID, total-2, dispute.Title)

	err = s.investigationCreator.InsertInvestigation(ctx, investigation)
	if err != nil {
		return fmt.Errorf("failed to insert opts: %w", err)
	}

	u2i := models.NewUser2Investigation(investigation.ID, uuid.Nil)
	userIDs, err := s.evidenceBroadcaster.BroadcastInvestigation(ctx, u2i, user.ID, opID)
	if err != nil {
		return fmt.Errorf("failed to broadcast investigation: %w", err)
	}

	users, err := s.userFinder.GetUsers(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, u := range users {
		if u.NotificationEnabled {
			msg := "Вам доступно новое расследование!"
			if err = s.msgSender.SendMessage(u.ChatID, msg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s EvidenceService) GetEvidences(ctx context.Context, disputeID string) ([]models.Evidence, error) {
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return nil, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	evidences, err := s.evidenceGetter.GetEvidences(ctx, disputeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidences: %w", err)
	}

	return evidences, nil
}
