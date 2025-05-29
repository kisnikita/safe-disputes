package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type DisputeFinder interface {
	GetDisputeByID(ctx context.Context, disputeID uuid.UUID, creatorID uuid.UUID) (models.Dispute, error)
	ListDisputes(ctx context.Context, opts models.DisputeListOpts) ([]models.Dispute, error)
	GetDisputeForEvidence(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error)
}

type DisputeCreator interface {
	InsertDispute(ctx context.Context, dispute models.Dispute) error
}

type User2DisputeCreator interface {
	InsertUser2Dispute(ctx context.Context, u2d models.User2Dispute) error
}

type OpponentGetter interface {
	GetOpponentID(ctx context.Context, disputeID uuid.UUID, creatorID uuid.UUID) (uuid.UUID, error)
}

type User2DisputeGetter interface {
	GetUser2Dispute(ctx context.Context, disputeID uuid.UUID, userID uuid.UUID) (models.User2Dispute, error)
}

type User2DisputeUpdater interface {
	UpdateUser2Dispute(ctx context.Context, opts models.U2DUpdateOpts) error
}

type DisputeService struct {
	logger log.Logger

	disputeCreator DisputeCreator
	disputeFinder  DisputeFinder
	u2dCreator     User2DisputeCreator
	u2dGetter      User2DisputeGetter
	u2dUpdater     User2DisputeUpdater
	opponentGetter OpponentGetter
	userFinder     UserFinder
	msgSender      MessageSender
}

func NewDisputeService(repo *repository.Repository, log log.Logger, msgSender MessageSender) (DisputeService, error) {
	if repo == nil {
		return DisputeService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return DisputeService{}, fmt.Errorf("logger is nil")
	}
	return DisputeService{
		logger:         log,
		disputeCreator: repo,
		disputeFinder:  repo,
		u2dCreator:     repo,
		u2dGetter:      repo,
		u2dUpdater:     repo,
		opponentGetter: repo,
		userFinder:     repo,
		msgSender:      msgSender,
	}, nil
}

func (s DisputeService) CreateDispute(ctx context.Context, dispute models.Dispute, creatorUsername string) error {
	if dispute.Title == "" || dispute.Description == "" || dispute.Opponent == "" || dispute.Amount <= 0 {
		return fmt.Errorf("invalid dispute data: title, description, opponent and amount must be provided")
	}

	opponent, err := s.userFinder.GetUserByUsername(ctx, dispute.Opponent)
	if err != nil {
		return fmt.Errorf("failed to check if opponent exists: %w", ErrUserNotFound)
	}

	if !opponent.DisputeReadiness {
		return fmt.Errorf("opponent %s %w", opponent.Username, ErrUnready)
	}

	if opponent.MinimumDisputeAmount > dispute.Amount {
		return fmt.Errorf("%d %w", dispute.Amount, ErrMinimalAmount)
	}

	err = s.disputeCreator.InsertDispute(ctx, dispute)
	if err != nil {
		return fmt.Errorf("failed to create dispute: %w", err)
	}

	u2dOpponent := models.NewUser2Dispute(opponent.ID, dispute.ID, models.DisputesStatusNew, models.DisputesResultNew)
	err = s.u2dCreator.InsertUser2Dispute(ctx, u2dOpponent)
	if err != nil {
		return fmt.Errorf("failed to create user2dispute for opponent: %w", err)
	}

	creator, err := s.userFinder.GetUserByUsername(ctx, creatorUsername)
	if err != nil {
		return fmt.Errorf("failed to get creator user: %w", err)
	}
	u2dCreator := models.NewUser2Dispute(creator.ID, dispute.ID, models.DisputesStatusCurrent, models.DisputesResultSent)
	err = s.u2dCreator.InsertUser2Dispute(ctx, u2dCreator)
	if err != nil {
		return fmt.Errorf("failed to create user2dispute for creator: %w", err)
	}

	if opponent.NotificationEnabled {
		if err = s.msgSender.SendMessage(opponent.ChatID, "У вас новое пари от "+creator.Username); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) ListDisputes(ctx context.Context, opts models.DisputeListOpts, creatorUsername string,
) ([]models.Dispute, error) {

	creator, err := s.userFinder.GetUserByUsername(ctx, creatorUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator user: %w", err)
	}

	opts.Creator = creator.ID

	disputes, err := s.disputeFinder.ListDisputes(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}

	if len(disputes) == 0 {
		s.logger.Info("no disputes found", zap.String("creator", creator.Username))
		return []models.Dispute{}, nil
	}

	for i := range disputes {
		opponentID, err := s.opponentGetter.GetOpponentID(ctx, disputes[i].ID, creator.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get opponent for dispute %s: %w", disputes[i].ID, err)
		}
		opponent, err := s.userFinder.GetUserByID(ctx, opponentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get opponent user: %w", err)
		}
		disputes[i].Opponent = opponent.Username
	}

	return disputes, nil
}

func (s DisputeService) GetDispute(ctx context.Context, disputeID string, creatorUsername string,
) (models.Dispute, error) {
	creator, err := s.userFinder.GetUserByUsername(ctx, creatorUsername)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get creator user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, creator.ID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute: %w", err)
	}

	opponentID, err := s.opponentGetter.GetOpponentID(ctx, dispute.ID, creator.ID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get opponent for dispute %s: %w", dispute.ID, err)
	}
	opponent, err := s.userFinder.GetUserByID(ctx, opponentID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get opponent user: %w", err)
	}
	dispute.Opponent = opponent.Username

	return dispute, nil
}

func (s DisputeService) AcceptDispute(ctx context.Context, disputeID string, acceptorUsername string) error {
	// Get acceptor and update
	acceptor, err := s.userFinder.GetUserByUsername(ctx, acceptorUsername)
	if err != nil {
		return fmt.Errorf("failed to get acceptor user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	u2d, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return err
	}

	if u2d.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", u2d.ID)
	}

	status := models.DisputesStatusCurrent
	result := models.DisputesResultProcessed
	opts := models.U2DUpdateOpts{
		ID:     u2d.ID,
		Status: &status,
		Result: &result,
	}
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
		return fmt.Errorf("failed to update rejector dispute status: %w", err)
	}

	// Get opponent and update
	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	u2dOp, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}
	opts.ID = u2dOp.ID
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}

	if opponent.NotificationEnabled {
		msg := fmt.Sprintf("Ваше пари %s было принято %s", dispute.Title, acceptor.Username)
		if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) RejectDispute(ctx context.Context, disputeID string, rejectorUsername string) error {
	// Get rejector and update
	acceptor, err := s.userFinder.GetUserByUsername(ctx, rejectorUsername)
	if err != nil {
		return fmt.Errorf("failed to get acceptor user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	u2d, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return err
	}

	if u2d.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", u2d.ID)
	}

	status := models.DisputesStatusPassed
	result := models.DisputesResultRejected
	opts := models.U2DUpdateOpts{
		ID:     u2d.ID,
		Status: &status,
		Result: &result,
	}
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
		return fmt.Errorf("failed to update rejector dispute status: %w", err)
	}

	// Get opponent and update
	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	u2dOp, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}
	opts.ID = u2dOp.ID
	tr := true
	opts.Claim = &tr
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}

	if opponent.NotificationEnabled {
		msg := fmt.Sprintf("Ваше пари %s было отменено %s", dispute.Title, acceptor.Username)
		if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) ClaimDispute(ctx context.Context, disputeID string, claimerUsername string) error {
	// Get claimer and update
	claimer, err := s.userFinder.GetUserByUsername(ctx, claimerUsername)
	if err != nil {
		return fmt.Errorf("failed to get claimer user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	u2d, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, claimer.ID)
	if err != nil {
		return err
	}

	if u2d.Status != models.DisputesStatusPassed {
		return fmt.Errorf("user2duspite %s is not in current status", u2d.ID)
	}

	fl := false
	opts := models.U2DUpdateOpts{
		ID:    u2d.ID,
		Claim: &fl,
	}
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
		return fmt.Errorf("failed to update claimer dispute status: %w", err)
	}
	return nil
}

func (s DisputeService) VoteDispute(ctx context.Context, disputeID string, claimerUsername string,
	win bool) error {
	voter, err := s.userFinder.GetUserByUsername(ctx, claimerUsername)
	if err != nil {
		return fmt.Errorf("failed to get claimer user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	u2d, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, voter.ID)
	if err != nil {
		return err
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, voter.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	u2dOp, err := s.u2dGetter.GetUser2Dispute(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, voter.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}

	var opts models.U2DUpdateOpts
	tr := true
	status := models.DisputesStatusPassed

	// --- Opponent not voted yet ---
	if u2dOp.Result == models.DisputesResultProcessed {
		opts.ID = u2d.ID
		opts.Vote = &win
		res := models.DisputesResultAnswered
		opts.Result = &res
		err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		return nil
	}

	// -- Opponent already voted ---

	// -- draw --
	if !u2dOp.Vote && !win {
		res := models.DisputesResultDraw
		opts.ID = u2dOp.ID
		opts.Result = &res
		opts.Status = &status
		opts.Claim = &tr
		err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		opts.ID = u2d.ID
		if err = s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось вничью. Заберите награду!",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- win --
	if !u2dOp.Vote && win {
		res := models.DisputesResultLose
		opts.ID = u2dOp.ID
		opts.Result = &res
		opts.Status = &status
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}

		res = models.DisputesResultWin
		opts.ID = u2d.ID
		opts.Result = &res
		opts.Claim = &tr
		opts.Vote = &win
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось поражением.", dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- lose --
	if u2dOp.Vote && !win {
		res := models.DisputesResultLose
		opts.ID = u2d.ID
		opts.Status = &status
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		res = models.DisputesResultWin
		opts.ID = u2dOp.ID
		opts.Result = &res
		opts.Claim = &tr
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось победой. Заберите награду!",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- investigation --
	if u2dOp.Vote && win {
		res := models.DisputesResultEvidence
		opts.ID = u2d.ID
		opts.Vote = &win
		opts.Result = &res
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		opts.ID = u2dOp.ID
		if err := s.u2dUpdater.UpdateUser2Dispute(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s требует доказательств.",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (s DisputeService) GetDisputeForEvidence(ctx context.Context, disputeID string) (models.Dispute, error) {
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeForEvidence(ctx, disputeUUID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute for evidence: %w", err)
	}

	return dispute, nil
}
