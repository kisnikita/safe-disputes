package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeDisputeRepo struct {
	usersByUsername   map[string]models.User
	usersByID         map[uuid.UUID]models.User
	participantByUser map[uuid.UUID]models.Participant
	dispute           models.Dispute
	opponentID        uuid.UUID

	insertDisputeCalls int
	insertDPCalls      int
	insertedDP         []models.Participant
	updatedDP          []models.ParticipantUpdateOpts
	updatedDeadlines   []time.Time
}

type fakeTxMonitor struct {
	err   error
	calls int
	boc   string
}

func (f *fakeTxMonitor) WaitForSuccess(_ context.Context, boc string) error {
	f.calls++
	f.boc = boc
	return f.err
}

func (f *fakeDisputeRepo) GetDisputeByID(context.Context, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeDisputeRepo) ListDisputes(context.Context, models.DisputeListOpts) ([]models.Dispute, error) {
	return nil, nil
}
func (f *fakeDisputeRepo) ListDisputeCards(context.Context, string, models.DisputeListOpts,
) ([]models.DisputeCard, error) {
	return nil, nil
}
func (f *fakeDisputeRepo) GetDisputeDetailsByID(context.Context, uuid.UUID, string) (models.DisputeDetails, error) {
	return models.DisputeDetails{}, nil
}
func (f *fakeDisputeRepo) GetDisputeForEvidence(context.Context, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeDisputeRepo) InsertDispute(context.Context, models.Dispute) error {
	f.insertDisputeCalls++
	return nil
}
func (f *fakeDisputeRepo) UpdateDisputeNextDeadline(_ context.Context, _ uuid.UUID, nextDeadline time.Time) error {
	f.updatedDeadlines = append(f.updatedDeadlines, nextDeadline)
	return nil
}
func (f *fakeDisputeRepo) InsertParticipant(_ context.Context, participant models.Participant) error {
	f.insertDPCalls++
	f.insertedDP = append(f.insertedDP, participant)
	return nil
}
func (f *fakeDisputeRepo) GetOpponentID(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
	return f.opponentID, nil
}
func (f *fakeDisputeRepo) GetParticipant(_ context.Context, _ uuid.UUID, userID uuid.UUID) (models.Participant, error) {
	participant, ok := f.participantByUser[userID]
	if !ok {
		return models.Participant{}, errors.New("participant not found")
	}
	return participant, nil
}
func (f *fakeDisputeRepo) UpdateParticipant(_ context.Context, opts models.ParticipantUpdateOpts) error {
	f.updatedDP = append(f.updatedDP, opts)
	return nil
}
func (f *fakeDisputeRepo) GetUserByID(_ context.Context, id uuid.UUID) (models.User, error) {
	u, ok := f.usersByID[id]
	if !ok {
		return models.User{}, errors.New("user not found")
	}
	return u, nil
}
func (f *fakeDisputeRepo) GetUserByUsername(_ context.Context, username string) (models.User, error) {
	u, ok := f.usersByUsername[username]
	if !ok {
		return models.User{}, errors.New("user not found")
	}
	return u, nil
}
func (f *fakeDisputeRepo) ExistByUsername(context.Context, string) (bool, error) { return false, nil }
func (f *fakeDisputeRepo) GetTotalUsers(context.Context) (int, error)            { return 0, nil }
func (f *fakeDisputeRepo) GetUsers(context.Context, []uuid.UUID) ([]models.User, error) {
	return nil, nil
}
func (f *fakeDisputeRepo) GetTopUsers(context.Context, int) ([]models.User, error) { return nil, nil }

func TestDisputeServiceCreateDispute(t *testing.T) {
	creator := models.User{ID: uuid.New(), Username: "alice"}
	opponent := models.User{
		ID:                       uuid.New(),
		Username:                 "bob",
		DisputeReadiness:         true,
		MinimumDisputeAmountNano: 50 * models.NanoPerTON,
		NotificationEnabled:      true,
		ChatID:                   777,
	}
	repo := &fakeDisputeRepo{
		usersByUsername: map[string]models.User{
			"alice": creator,
			"bob":   opponent,
		},
	}
	sender := &fakeMessageSender{}
	txMonitor := &fakeTxMonitor{}
	svc := DisputeService{
		logger:             noopLogger{},
		disputeCreator:     repo,
		participantCreator: repo,
		userFinder:         repo,
		msgSender:          sender,
		txMonitor:          txMonitor,
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "test",
		Description:     "desc",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	}, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txMonitor.calls != 1 || txMonitor.boc != "boc" {
		t.Fatalf("expected tx monitor call with boc, got calls=%d boc=%q", txMonitor.calls, txMonitor.boc)
	}
	if repo.insertDisputeCalls != 1 {
		t.Fatalf("expected 1 dispute insert, got %d", repo.insertDisputeCalls)
	}
	if repo.insertDPCalls != 2 {
		t.Fatalf("expected 2 participant inserts, got %d", repo.insertDPCalls)
	}
	if sender.calls != 1 || sender.chatIDs[0] != 777 {
		t.Fatalf("expected message to chat 777, got calls=%d chats=%v", sender.calls, sender.chatIDs)
	}
}

func TestDisputeServiceCreateDisputeIgnoresOpponentSettings(t *testing.T) {
	repo := &fakeDisputeRepo{
		usersByUsername: map[string]models.User{
			"alice": {ID: uuid.New(), Username: "alice"},
			"bob":   {ID: uuid.New(), Username: "bob", DisputeReadiness: false, MinimumDisputeAmountNano: 500 * models.NanoPerTON},
		},
	}
	svc := DisputeService{
		logger:             noopLogger{},
		disputeCreator:     repo,
		userFinder:         repo,
		participantCreator: repo,
		msgSender:          &fakeMessageSender{},
		txMonitor:          &fakeTxMonitor{},
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	}, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisputeServiceCreateDisputeTxFailed(t *testing.T) {
	repo := &fakeDisputeRepo{
		usersByUsername: map[string]models.User{
			"alice": {ID: uuid.New(), Username: "alice"},
			"bob":   {ID: uuid.New(), Username: "bob"},
		},
	}
	svc := DisputeService{
		logger:             noopLogger{},
		disputeCreator:     repo,
		userFinder:         repo,
		participantCreator: repo,
		msgSender:          &fakeMessageSender{},
		txMonitor:          &fakeTxMonitor{err: ErrTxFailed},
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	}, "alice")
	if !errors.Is(err, ErrTxFailed) {
		t.Fatalf("expected ErrTxFailed, got %v", err)
	}
	if repo.insertDisputeCalls != 0 {
		t.Fatalf("expected no inserts when tx failed, got %d", repo.insertDisputeCalls)
	}
}

func TestDisputeServicePrecheckCreateDispute(t *testing.T) {
	repo := &fakeDisputeRepo{
		usersByUsername: map[string]models.User{
			"bob": {ID: uuid.New(), Username: "bob", DisputeReadiness: true, MinimumDisputeAmountNano: 50 * models.NanoPerTON},
		},
	}
	svc := DisputeService{logger: noopLogger{}, userFinder: repo}

	if err := svc.PrecheckCreateDispute(context.Background(), "bob", 100*models.NanoPerTON, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisputeServicePrecheckCreateDisputeSelfOpponent(t *testing.T) {
	svc := DisputeService{logger: noopLogger{}, userFinder: &fakeDisputeRepo{}}

	err := svc.PrecheckCreateDispute(context.Background(), "alice", 100*models.NanoPerTON, "alice")
	if !errors.Is(err, ErrSelfOpponent) {
		t.Fatalf("expected ErrSelfOpponent, got %v", err)
	}
}

func TestDisputeServiceGetDisputeInvalidID(t *testing.T) {
	svc := DisputeService{logger: noopLogger{}, userFinder: &fakeDisputeRepo{usersByUsername: map[string]models.User{"alice": {ID: uuid.New(), Username: "alice"}}}}

	_, err := svc.GetDispute(context.Background(), "not-uuid", "alice")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDisputeServiceWinDispute(t *testing.T) {
	voter := models.User{ID: uuid.New(), Username: "alice"}
	opponent := models.User{ID: uuid.New(), Username: "bob", NotificationEnabled: true, ChatID: 888}
	disputeID := uuid.New()

	t.Run("opponent not voted yet", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Result: models.DisputesResultProcessed},
			},
			dispute:    models.Dispute{ID: disputeID, Title: "D1"},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, participantGetter: repo,
			participantUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender, txMonitor: &fakeTxMonitor{}}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true, "boc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDP) != 1 {
			t.Fatalf("expected 1 update, got %d", len(repo.updatedDP))
		}
		if sender.calls != 0 {
			t.Fatalf("expected no messages, got %d", sender.calls)
		}
	})

	t.Run("win branch updates both and notifies opponent", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), IsWin: false, Result: models.DisputesResultAnswered},
			},
			dispute:    models.Dispute{ID: disputeID, Title: "D2"},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, participantGetter: repo,
			participantUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender, txMonitor: &fakeTxMonitor{}}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true, "boc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDP) != 2 {
			t.Fatalf("expected 2 updates, got %d", len(repo.updatedDP))
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 message, got %d", sender.calls)
		}
		if !strings.Contains(sender.messages[0], "поражением") {
			t.Fatalf("expected lose message, got %q", sender.messages[0])
		}
	})

	t.Run("evidence branch caps deadline by endsAt", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), IsWin: true, Result: models.DisputesResultAnswered},
			},
			dispute:    models.Dispute{ID: disputeID, Title: "D3"},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{
			logger:             noopLogger{},
			userFinder:         repo,
			participantGetter:  repo,
			participantUpdater: repo,
			opponentGetter:     repo,
			disputeFinder:      repo,
			disputeCreator:     repo,
			msgSender:          sender,
			txMonitor:          &fakeTxMonitor{},
		}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true, "boc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDeadlines) != 1 {
			t.Fatalf("expected 1 deadline update, got %d", len(repo.updatedDeadlines))
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 message, got %d", sender.calls)
		}
	})
}

func TestDisputeServiceRejectDispute(t *testing.T) {
	creator := models.User{ID: uuid.New(), Username: "alice", NotificationEnabled: true, ChatID: 111}
	opponent := models.User{ID: uuid.New(), Username: "bob", NotificationEnabled: true, ChatID: 999}
	disputeID := uuid.New()

	t.Run("opponent rejects and creator becomes claimable", func(t *testing.T) {
		creatorParticipantID := uuid.New()
		opponentParticipantID := uuid.New()
		repo := &fakeDisputeRepo{
				usersByUsername: map[string]models.User{"bob": opponent},
				usersByID:       map[uuid.UUID]models.User{creator.ID: creator},
				participantByUser: map[uuid.UUID]models.Participant{
					opponent.ID: {
						ID:     opponentParticipantID,
						UserID: opponent.ID,
						Status: models.DisputesStatusNew,
						Result: models.DisputesResultNew,
					},
					creator.ID: {
						ID:     creatorParticipantID,
						UserID: creator.ID,
						Status: models.DisputesStatusNew,
						Result: models.DisputesResultSent,
					},
				},
				dispute:    models.Dispute{ID: disputeID, Title: "R1"},
				opponentID: creator.ID,
			}
		sender := &fakeMessageSender{}
		svc := DisputeService{
			logger:             noopLogger{},
			userFinder:         repo,
			participantGetter:  repo,
			participantUpdater: repo,
			opponentGetter:     repo,
			disputeFinder:      repo,
			msgSender:          sender,
		}

		err := svc.RejectDispute(context.Background(), disputeID.String(), opponent.Username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDP) != 2 {
			t.Fatalf("expected 2 updates, got %d", len(repo.updatedDP))
		}
		first := repo.updatedDP[0]
		second := repo.updatedDP[1]
		if first.ID != opponentParticipantID {
			t.Fatalf("expected first update for rejector participant, got %s", first.ID)
		}
		if first.IsClaimable == nil || *first.IsClaimable {
			t.Fatal("expected rejector(new) to remain non-claimable")
		}
		if second.ID != creatorParticipantID {
			t.Fatalf("expected second update for creator participant, got %s", second.ID)
		}
		if second.IsClaimable == nil || !*second.IsClaimable {
			t.Fatal("expected creator to become claimable")
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 notification, got %d", sender.calls)
		}
	})

	t.Run("creator cancels sent and creator stays claimable", func(t *testing.T) {
		creatorParticipantID := uuid.New()
		opponentParticipantID := uuid.New()
		repo := &fakeDisputeRepo{
				usersByUsername: map[string]models.User{"alice": creator},
				usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
				participantByUser: map[uuid.UUID]models.Participant{
					creator.ID: {
						ID:     creatorParticipantID,
						UserID: creator.ID,
						Status: models.DisputesStatusNew,
						Result: models.DisputesResultSent,
					},
					opponent.ID: {
						ID:     opponentParticipantID,
						UserID: opponent.ID,
						Status: models.DisputesStatusNew,
						Result: models.DisputesResultNew,
					},
				},
				dispute:    models.Dispute{ID: disputeID, Title: "R2"},
				opponentID: opponent.ID,
			}
		sender := &fakeMessageSender{}
		svc := DisputeService{
			logger:             noopLogger{},
			userFinder:         repo,
			participantGetter:  repo,
			participantUpdater: repo,
			opponentGetter:     repo,
			disputeFinder:      repo,
			msgSender:          sender,
		}

		err := svc.RejectDispute(context.Background(), disputeID.String(), creator.Username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDP) != 2 {
			t.Fatalf("expected 2 updates, got %d", len(repo.updatedDP))
		}
		first := repo.updatedDP[0]
		second := repo.updatedDP[1]
		if first.ID != creatorParticipantID {
			t.Fatalf("expected first update for rejector participant, got %s", first.ID)
		}
		if first.IsClaimable == nil || !*first.IsClaimable {
			t.Fatal("expected creator to become claimable")
		}
		if second.ID != opponentParticipantID {
			t.Fatalf("expected second update for opponent participant, got %s", second.ID)
		}
		if second.IsClaimable == nil || *second.IsClaimable {
			t.Fatal("expected opponent(new) to remain non-claimable")
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 notification, got %d", sender.calls)
		}
	})
}

func TestDisputeServiceRefundDispute(t *testing.T) {
	refunder := models.User{ID: uuid.New(), Username: "alice"}
	disputeID := uuid.New()

	t.Run("claimable rejected participant can refund", func(t *testing.T) {
		participantID := uuid.New()
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": refunder},
			participantByUser: map[uuid.UUID]models.Participant{
				refunder.ID: {
					ID:          participantID,
					Result:      models.DisputesResultRejected,
					IsClaimable: true,
				},
			},
		}
		txMonitor := &fakeTxMonitor{}
		svc := DisputeService{
			logger:             noopLogger{},
			userFinder:         repo,
			participantGetter:  repo,
			participantUpdater: repo,
			txMonitor:          txMonitor,
		}

		err := svc.ClaimDispute(context.Background(), disputeID.String(), refunder.Username, "boc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if txMonitor.calls != 1 || txMonitor.boc != "boc" {
			t.Fatalf("expected tx monitor call with boc, got calls=%d boc=%q", txMonitor.calls, txMonitor.boc)
		}
		if len(repo.updatedDP) != 1 {
			t.Fatalf("expected 1 participant update, got %d", len(repo.updatedDP))
		}
		if repo.updatedDP[0].ID != participantID {
			t.Fatalf("expected update for participant %s, got %s", participantID, repo.updatedDP[0].ID)
		}
		if repo.updatedDP[0].IsClaimable == nil || *repo.updatedDP[0].IsClaimable {
			t.Fatal("expected is_claimable to be set to false")
		}
	})

	t.Run("non claimable participant cannot refund", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": refunder},
			participantByUser: map[uuid.UUID]models.Participant{
				refunder.ID: {
					ID:          uuid.New(),
					Result:      models.DisputesResultRejected,
					IsClaimable: false,
				},
			},
		}
		svc := DisputeService{
			logger:             noopLogger{},
			userFinder:         repo,
			participantGetter:  repo,
			participantUpdater: repo,
			txMonitor:          &fakeTxMonitor{},
		}

		err := svc.ClaimDispute(context.Background(), disputeID.String(), refunder.Username, "boc")
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
		if len(repo.updatedDP) != 0 {
			t.Fatalf("expected no participant updates, got %d", len(repo.updatedDP))
		}
	})
}
