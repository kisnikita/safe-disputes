package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeDisputeRepo struct {
	usersByUsername map[string]models.User
	usersByID       map[uuid.UUID]models.User
	u2dByUser       map[uuid.UUID]models.User2Dispute
	dispute         models.Dispute
	opponentID      uuid.UUID

	insertDisputeCalls int
	insertU2DCalls     int
	insertedU2D        []models.User2Dispute
	updatedU2D         []models.U2DUpdateOpts
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

func (f *fakeDisputeRepo) GetDisputeByID(context.Context, uuid.UUID, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeDisputeRepo) ListDisputes(context.Context, models.DisputeListOpts) ([]models.Dispute, error) {
	return nil, nil
}
func (f *fakeDisputeRepo) GetDisputeForEvidence(context.Context, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeDisputeRepo) InsertDispute(context.Context, models.Dispute) error {
	f.insertDisputeCalls++
	return nil
}
func (f *fakeDisputeRepo) InsertUser2Dispute(_ context.Context, u2d models.User2Dispute) error {
	f.insertU2DCalls++
	f.insertedU2D = append(f.insertedU2D, u2d)
	return nil
}
func (f *fakeDisputeRepo) GetOpponentID(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
	return f.opponentID, nil
}
func (f *fakeDisputeRepo) GetUser2Dispute(_ context.Context, _ uuid.UUID, userID uuid.UUID) (models.User2Dispute, error) {
	u2d, ok := f.u2dByUser[userID]
	if !ok {
		return models.User2Dispute{}, errors.New("u2d not found")
	}
	return u2d, nil
}
func (f *fakeDisputeRepo) UpdateUser2Dispute(_ context.Context, opts models.U2DUpdateOpts) error {
	f.updatedU2D = append(f.updatedU2D, opts)
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
		ID:                   uuid.New(),
		Username:             "bob",
		DisputeReadiness:     true,
		MinimumDisputeAmount: 50,
		NotificationEnabled:  true,
		ChatID:               777,
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
		logger:         noopLogger{},
		disputeCreator: repo,
		u2dCreator:     repo,
		userFinder:     repo,
		msgSender:      sender,
		txMonitor:      txMonitor,
	}

	err := svc.CreateDispute(context.Background(), models.Dispute{
		DisputeDB: models.DisputeDB{
			ID:              uuid.New(),
			Title:           "test",
			Description:     "desc",
			Amount:          100,
			ContractAddress: "addr",
		},
		Opponent: "bob",
	}, "alice", "boc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txMonitor.calls != 1 || txMonitor.boc != "boc" {
		t.Fatalf("expected tx monitor call with boc, got calls=%d boc=%q", txMonitor.calls, txMonitor.boc)
	}
	if repo.insertDisputeCalls != 1 {
		t.Fatalf("expected 1 dispute insert, got %d", repo.insertDisputeCalls)
	}
	if repo.insertU2DCalls != 2 {
		t.Fatalf("expected 2 u2d inserts, got %d", repo.insertU2DCalls)
	}
	if sender.calls != 1 || sender.chatIDs[0] != 777 {
		t.Fatalf("expected message to chat 777, got calls=%d chats=%v", sender.calls, sender.chatIDs)
	}
}

func TestDisputeServiceCreateDisputeIgnoresOpponentSettings(t *testing.T) {
	repo := &fakeDisputeRepo{
		usersByUsername: map[string]models.User{
			"alice": {ID: uuid.New(), Username: "alice"},
			"bob":   {ID: uuid.New(), Username: "bob", DisputeReadiness: false, MinimumDisputeAmount: 500},
		},
	}
	svc := DisputeService{
		logger:         noopLogger{},
		disputeCreator: repo,
		userFinder:     repo,
		u2dCreator:     repo,
		msgSender:      &fakeMessageSender{},
		txMonitor:      &fakeTxMonitor{},
	}

	err := svc.CreateDispute(context.Background(), models.Dispute{
		DisputeDB: models.DisputeDB{
			ID:              uuid.New(),
			Title:           "t",
			Description:     "d",
			Amount:          100,
			ContractAddress: "addr",
		},
		Opponent: "bob",
	}, "alice", "boc")
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
		logger:         noopLogger{},
		disputeCreator: repo,
		userFinder:     repo,
		u2dCreator:     repo,
		msgSender:      &fakeMessageSender{},
		txMonitor:      &fakeTxMonitor{err: ErrTxFailed},
	}

	err := svc.CreateDispute(context.Background(), models.Dispute{
		DisputeDB: models.DisputeDB{
			ID:              uuid.New(),
			Title:           "t",
			Description:     "d",
			Amount:          100,
			ContractAddress: "addr",
		},
		Opponent: "bob",
	}, "alice", "boc")
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
			"bob": {ID: uuid.New(), Username: "bob", DisputeReadiness: true, MinimumDisputeAmount: 50},
		},
	}
	svc := DisputeService{logger: noopLogger{}, userFinder: repo}

	if err := svc.PrecheckCreateDispute(context.Background(), "bob", 100, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisputeServicePrecheckCreateDisputeSelfOpponent(t *testing.T) {
	svc := DisputeService{logger: noopLogger{}, userFinder: &fakeDisputeRepo{}}

	err := svc.PrecheckCreateDispute(context.Background(), "alice", 100, "alice")
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

func TestDisputeServiceVoteDispute(t *testing.T) {
	voter := models.User{ID: uuid.New(), Username: "alice"}
	opponent := models.User{ID: uuid.New(), Username: "bob", NotificationEnabled: true, ChatID: 888}
	disputeID := uuid.New()

	t.Run("opponent not voted yet", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			u2dByUser: map[uuid.UUID]models.User2Dispute{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Result: models.DisputesResultProcessed},
			},
			dispute:    models.Dispute{DisputeDB: models.DisputeDB{ID: disputeID, Title: "D1"}},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, u2dGetter: repo, u2dUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedU2D) != 1 {
			t.Fatalf("expected 1 update, got %d", len(repo.updatedU2D))
		}
		if sender.calls != 0 {
			t.Fatalf("expected no messages, got %d", sender.calls)
		}
	})

	t.Run("draw branch updates both and notifies opponent", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			u2dByUser: map[uuid.UUID]models.User2Dispute{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Vote: false, Result: models.DisputesResultAnswered},
			},
			dispute:    models.Dispute{DisputeDB: models.DisputeDB{ID: disputeID, Title: "D2"}},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, u2dGetter: repo, u2dUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedU2D) != 2 {
			t.Fatalf("expected 2 updates, got %d", len(repo.updatedU2D))
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 message, got %d", sender.calls)
		}
		if !strings.Contains(sender.messages[0], "вничью") {
			t.Fatalf("expected draw message, got %q", sender.messages[0])
		}
	})
}
