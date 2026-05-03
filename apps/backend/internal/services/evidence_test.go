package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeEvidenceDeps struct {
	user         models.User
	disputeParticipantSelf      models.DisputeParticipant
	disputeParticipantOpponent  models.DisputeParticipant
	opponentID   uuid.UUID
	totalUsers   int
	dispute      models.Dispute
	usersByIDs   []models.User
	broadcastIDs []uuid.UUID
	isFirst      bool

	insertEvidenceCalls      int
	insertInvestigationCalls int
	updatedDP               []models.DisputeParticipantUpdateOpts
}

func (f *fakeEvidenceDeps) InsertEvidence(context.Context, models.Evidence) error {
	f.insertEvidenceCalls++
	return nil
}
func (f *fakeEvidenceDeps) IsFirstEvidence(context.Context, string) (bool, error) { return f.isFirst, nil }
func (f *fakeEvidenceDeps) GetEvidences(context.Context, uuid.UUID) ([]models.Evidence, error) {
	return nil, nil
}
func (f *fakeEvidenceDeps) BroadcastInvestigation(context.Context, models.User2Investigation, uuid.UUID, uuid.UUID) ([]uuid.UUID, error) {
	return f.broadcastIDs, nil
}
func (f *fakeEvidenceDeps) GetUserByID(context.Context, uuid.UUID) (models.User, error) { return models.User{}, nil }
func (f *fakeEvidenceDeps) GetUserByUsername(context.Context, string) (models.User, error) {
	return f.user, nil
}
func (f *fakeEvidenceDeps) ExistByUsername(context.Context, string) (bool, error) { return false, nil }
func (f *fakeEvidenceDeps) GetTotalUsers(context.Context) (int, error)            { return f.totalUsers, nil }
func (f *fakeEvidenceDeps) GetUsers(context.Context, []uuid.UUID) ([]models.User, error) {
	return f.usersByIDs, nil
}
func (f *fakeEvidenceDeps) GetTopUsers(context.Context, int) ([]models.User, error) { return nil, nil }
func (f *fakeEvidenceDeps) UpdateUser(context.Context, models.UserUpdateOpts) error  { return nil }
func (f *fakeEvidenceDeps) EarnWinnerRating(context.Context, []uuid.UUID) error      { return nil }
func (f *fakeEvidenceDeps) UpdateDisputeParticipant(_ context.Context, opts models.DisputeParticipantUpdateOpts) error {
	f.updatedDP = append(f.updatedDP, opts)
	return nil
}
func (f *fakeEvidenceDeps) GetDisputeParticipant(_ context.Context, _ uuid.UUID, userID uuid.UUID) (models.DisputeParticipant, error) {
	if userID == f.user.ID {
		return f.disputeParticipantSelf, nil
	}
	return f.disputeParticipantOpponent, nil
}
func (f *fakeEvidenceDeps) GetOpponentID(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
	return f.opponentID, nil
}
func (f *fakeEvidenceDeps) InsertInvestigation(context.Context, models.Investigation) error {
	f.insertInvestigationCalls++
	return nil
}
func (f *fakeEvidenceDeps) GetDisputeByID(context.Context, uuid.UUID, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeEvidenceDeps) ListDisputes(context.Context, models.DisputeListOpts) ([]models.Dispute, error) {
	return nil, nil
}
func (f *fakeEvidenceDeps) GetDisputeForEvidence(context.Context, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}

func TestEvidenceServiceProvideEvidenceFirst(t *testing.T) {
	userID := uuid.New()
	deps := &fakeEvidenceDeps{
		isFirst:    true,
		user:       models.User{ID: userID, Username: "alice"},
		disputeParticipantSelf:    models.DisputeParticipant{ID: uuid.New()},
		dispute:    models.Dispute{ID: uuid.New(), Title: "D1"},
		totalUsers: 10,
	}
	svc := EvidenceService{
		logger:          noopLogger{},
		evidenceCreator: deps,
		evidenceChecker: deps,
		userFinder:      deps,
		disputeParticipantUpdater:      deps,
		disputeParticipantGetter:       deps,
	}

	err := svc.ProvideEvidence(context.Background(), models.EvidenceOpts{DisputeID: uuid.NewString(), Username: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.insertEvidenceCalls != 1 {
		t.Fatalf("expected 1 insert evidence, got %d", deps.insertEvidenceCalls)
	}
	if len(deps.updatedDP) != 1 {
		t.Fatalf("expected 1 disputeParticipant update, got %d", len(deps.updatedDP))
	}
	if deps.insertInvestigationCalls != 0 {
		t.Fatalf("expected no investigation insert, got %d", deps.insertInvestigationCalls)
	}
}

func TestEvidenceServiceProvideEvidenceSecond(t *testing.T) {
	userID := uuid.New()
	opID := uuid.New()
	deps := &fakeEvidenceDeps{
		isFirst:      false,
		user:         models.User{ID: userID, Username: "alice"},
		disputeParticipantSelf:      models.DisputeParticipant{ID: uuid.New()},
		disputeParticipantOpponent:  models.DisputeParticipant{ID: uuid.New()},
		opponentID:   opID,
		totalUsers:   12,
		dispute:      models.Dispute{ID: uuid.New(), Title: "D2"},
		broadcastIDs: []uuid.UUID{uuid.New(), uuid.New()},
		usersByIDs: []models.User{
			{ID: uuid.New(), NotificationEnabled: true, ChatID: 101},
			{ID: uuid.New(), NotificationEnabled: false, ChatID: 202},
		},
	}
	sender := &fakeMessageSender{}
	svc := EvidenceService{
		logger:               noopLogger{},
		evidenceCreator:      deps,
		evidenceChecker:      deps,
		userFinder:           deps,
		disputeParticipantUpdater:           deps,
		disputeParticipantGetter:            deps,
		opponentGetter:       deps,
		investigationCreator: deps,
		evidenceBroadcaster:  deps,
		disputesFinder:       deps,
		msgSender:            sender,
	}

	err := svc.ProvideEvidence(context.Background(), models.EvidenceOpts{DisputeID: uuid.NewString(), Username: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.insertEvidenceCalls != 1 {
		t.Fatalf("expected 1 evidence insert, got %d", deps.insertEvidenceCalls)
	}
	if len(deps.updatedDP) != 2 {
		t.Fatalf("expected 2 disputeParticipant updates, got %d", len(deps.updatedDP))
	}
	if deps.insertInvestigationCalls != 1 {
		t.Fatalf("expected 1 investigation insert, got %d", deps.insertInvestigationCalls)
	}
	if sender.calls != 1 {
		t.Fatalf("expected 1 notification, got %d", sender.calls)
	}
}

func TestEvidenceServiceGetEvidencesInvalidID(t *testing.T) {
	svc := EvidenceService{logger: noopLogger{}, evidenceGetter: &fakeEvidenceDeps{}}

	_, err := svc.GetEvidences(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected error")
	}
}
