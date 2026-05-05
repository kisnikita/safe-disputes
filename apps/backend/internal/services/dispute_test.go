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
	usersByUsername map[string]models.User
	usersByID       map[uuid.UUID]models.User
	participantByUser       map[uuid.UUID]models.Participant
	dispute         models.Dispute
	opponentID      uuid.UUID

	insertDisputeCalls int
	insertDPCalls     int
	insertedDP        []models.Participant
	updatedDP         []models.ParticipantUpdateOpts
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

func (f *fakeDisputeRepo) GetDisputeByID(context.Context, uuid.UUID, uuid.UUID) (models.Dispute, error) {
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
		logger:         noopLogger{},
		disputeCreator: repo,
		participantCreator:     repo,
		userFinder:     repo,
		msgSender:      sender,
		txMonitor:      txMonitor,
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "test",
		Description:     "desc",
		Opponent:        "bob",
		AmountNano:      "100000000000",
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
		logger:         noopLogger{},
		disputeCreator: repo,
		userFinder:     repo,
		participantCreator:     repo,
		msgSender:      &fakeMessageSender{},
		txMonitor:      &fakeTxMonitor{},
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
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
		logger:         noopLogger{},
		disputeCreator: repo,
		userFinder:     repo,
		participantCreator:     repo,
		msgSender:      &fakeMessageSender{},
		txMonitor:      &fakeTxMonitor{err: ErrTxFailed},
	}

	err := svc.CreateDispute(context.Background(), models.CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
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
	svc := DisputeService{logger: noopLogger{}, userFinder: 
	&fakeDisputeRepo{usersByUsername: map[string]models.User{"alice": {ID: uuid.New(), Username: "alice"}}}}

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
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Result: models.DisputesResultProcessed},
			},
				dispute:    models.Dispute{ID: disputeID, Title: "D1"},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, participantGetter: repo, 
		participantUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true)
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

	t.Run("draw branch updates both and notifies opponent", func(t *testing.T) {
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Vote: false, Result: models.DisputesResultAnswered},
			},
				dispute:    models.Dispute{ID: disputeID, Title: "D2"},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{logger: noopLogger{}, userFinder: repo, participantGetter: repo, 
		participantUpdater: repo, opponentGetter: repo, disputeFinder: repo, msgSender: sender}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDP) != 2 {
			t.Fatalf("expected 2 updates, got %d", len(repo.updatedDP))
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 message, got %d", sender.calls)
		}
		if !strings.Contains(sender.messages[0], "вничью") {
			t.Fatalf("expected draw message, got %q", sender.messages[0])
		}
	})

	t.Run("evidence branch caps deadline by endsAt", func(t *testing.T) {
		soonEndsAt := time.Now().Add(90 * time.Minute)
		repo := &fakeDisputeRepo{
			usersByUsername: map[string]models.User{"alice": voter},
			usersByID:       map[uuid.UUID]models.User{opponent.ID: opponent},
			participantByUser: map[uuid.UUID]models.Participant{
				voter.ID:    {ID: uuid.New(), Status: models.DisputesStatusCurrent},
				opponent.ID: {ID: uuid.New(), Vote: true, Result: models.DisputesResultAnswered},
			},
				dispute:    models.Dispute{ID: disputeID, Title: "D3", EndsAt: soonEndsAt},
			opponentID: opponent.ID,
		}
		sender := &fakeMessageSender{}
		svc := DisputeService{
			logger:         noopLogger{},
			userFinder:     repo,
			participantGetter:      repo,
			participantUpdater:     repo,
			opponentGetter: repo,
			disputeFinder:  repo,
			disputeCreator: repo,
			msgSender:      sender,
		}

		err := svc.VoteDispute(context.Background(), disputeID.String(), "alice", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repo.updatedDeadlines) != 1 {
			t.Fatalf("expected 1 deadline update, got %d", len(repo.updatedDeadlines))
		}
		if !repo.updatedDeadlines[0].Equal(soonEndsAt) {
			t.Fatalf("expected deadline capped to endsAt, got %s want %s", repo.updatedDeadlines[0], soonEndsAt)
		}
		if sender.calls != 1 {
			t.Fatalf("expected 1 message, got %d", sender.calls)
		}
	})
}
