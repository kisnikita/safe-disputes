package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeInvestigationDeps struct {
	user             models.User
	u2i              models.User2Investigation
	investigation    models.Investigation
	winners          []uuid.UUID
	disputeUsers     []models.User
	u2dByUser        map[uuid.UUID]models.User2Dispute
	dispute          models.Dispute
	listResult       []models.Investigation
	listReceivedOpts models.InvestigationListOpts

	updatedU2I      []models.U2IUpdateOpts
	updatedInv      []models.InvestigationUpdateOpts
	updatedU2D      []models.U2DUpdateOpts
	deleteNoVoteCnt int
	earnWinnerCnt   int
	updateWinnerCnt int
}

func (f *fakeInvestigationDeps) InsertInvestigation(context.Context, models.Investigation) error { return nil }
func (f *fakeInvestigationDeps) ListInvestigations(_ context.Context, opts models.InvestigationListOpts) ([]models.Investigation, error) {
	f.listReceivedOpts = opts
	return f.listResult, nil
}
func (f *fakeInvestigationDeps) GetInvestigation(context.Context, uuid.UUID, uuid.UUID) (models.Investigation, error) {
	return f.investigation, nil
}
func (f *fakeInvestigationDeps) UpdateInvestigation(_ context.Context, opts models.InvestigationUpdateOpts) error {
	f.updatedInv = append(f.updatedInv, opts)
	return nil
}
func (f *fakeInvestigationDeps) DeleteUsersWithoutVote(context.Context, uuid.UUID) error {
	f.deleteNoVoteCnt++
	return nil
}
func (f *fakeInvestigationDeps) GetUser2Investigation(context.Context, uuid.UUID, uuid.UUID) (models.User2Investigation, error) {
	return f.u2i, nil
}
func (f *fakeInvestigationDeps) GetWinnersIDs(context.Context, uuid.UUID, string) ([]uuid.UUID, error) {
	return f.winners, nil
}
func (f *fakeInvestigationDeps) GetDisputesUsers(context.Context, uuid.UUID) ([]models.User, error) {
	return f.disputeUsers, nil
}
func (f *fakeInvestigationDeps) UpdateUser2Investigation(_ context.Context, opts models.U2IUpdateOpts) error {
	f.updatedU2I = append(f.updatedU2I, opts)
	return nil
}
func (f *fakeInvestigationDeps) UpdateWinnersResult(context.Context, uuid.UUID, []uuid.UUID) error {
	f.updateWinnerCnt++
	return nil
}
func (f *fakeInvestigationDeps) GetUserByID(context.Context, uuid.UUID) (models.User, error) { return models.User{}, nil }
func (f *fakeInvestigationDeps) GetUserByUsername(context.Context, string) (models.User, error) {
	return f.user, nil
}
func (f *fakeInvestigationDeps) ExistByUsername(context.Context, string) (bool, error) { return false, nil }
func (f *fakeInvestigationDeps) GetTotalUsers(context.Context) (int, error)            { return 0, nil }
func (f *fakeInvestigationDeps) GetUsers(context.Context, []uuid.UUID) ([]models.User, error) {
	return nil, nil
}
func (f *fakeInvestigationDeps) GetTopUsers(context.Context, int) ([]models.User, error) { return nil, nil }
func (f *fakeInvestigationDeps) UpdateUser(context.Context, models.UserUpdateOpts) error { return nil }
func (f *fakeInvestigationDeps) EarnWinnerRating(context.Context, []uuid.UUID) error {
	f.earnWinnerCnt++
	return nil
}
func (f *fakeInvestigationDeps) UpdateUser2Dispute(_ context.Context, opts models.U2DUpdateOpts) error {
	f.updatedU2D = append(f.updatedU2D, opts)
	return nil
}
func (f *fakeInvestigationDeps) GetUser2Dispute(_ context.Context, _ uuid.UUID, userID uuid.UUID) (models.User2Dispute, error) {
	return f.u2dByUser[userID], nil
}
func (f *fakeInvestigationDeps) GetDisputeByID(context.Context, uuid.UUID, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}
func (f *fakeInvestigationDeps) ListDisputes(context.Context, models.DisputeListOpts) ([]models.Dispute, error) {
	return nil, nil
}
func (f *fakeInvestigationDeps) GetDisputeForEvidence(context.Context, uuid.UUID) (models.Dispute, error) {
	return f.dispute, nil
}

func TestInvestigationServiceListInvestigation(t *testing.T) {
	userID := uuid.New()
	deps := &fakeInvestigationDeps{
		user:       models.User{ID: userID, Username: "alice"},
		listResult: []models.Investigation{{InvestigationDB: models.InvestigationDB{ID: uuid.New()}}},
	}
	svc := InvestigationService{logger: noopLogger{}, userFinder: deps, investigationFinder: deps}

	res, err := svc.ListInvestigation(context.Background(), models.InvestigationListOpts{Limit: 5}, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if deps.listReceivedOpts.UserID != userID {
		t.Fatalf("expected user id to be propagated")
	}
}

func TestInvestigationServiceGetInvestigationInvalidID(t *testing.T) {
	deps := &fakeInvestigationDeps{user: models.User{ID: uuid.New(), Username: "alice"}}
	svc := InvestigationService{logger: noopLogger{}, userFinder: deps, investigationFinder: deps}

	_, err := svc.GetInvestigation(context.Background(), "bad-id", "alice")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvestigationServiceVoteInvestigationNonFinal(t *testing.T) {
	userID := uuid.New()
	invID := uuid.New()
	deps := &fakeInvestigationDeps{
		user:          models.User{ID: userID, Username: "alice", Rating: 3},
		u2i:           models.User2Investigation{ID: uuid.New()},
		investigation: models.Investigation{InvestigationDB: models.InvestigationDB{ID: invID, DisputeID: uuid.New(), Total: 3, P1: 1, P2: 0, Draw: 0}},
	}
	svc := InvestigationService{
		logger:               noopLogger{},
		userFinder:           deps,
		u2iFinder:            deps,
		u2iUpdater:           deps,
		userUpdater:          deps,
		investigationFinder:  deps,
		investigationUpdater: deps,
	}

	err := svc.VoteInvestigation(context.Background(), invID.String(), "alice", "p2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.updatedU2I) != 1 {
		t.Fatalf("expected 1 u2i update, got %d", len(deps.updatedU2I))
	}
	if len(deps.updatedInv) != 1 {
		t.Fatalf("expected 1 investigation update, got %d", len(deps.updatedInv))
	}
	if deps.deleteNoVoteCnt != 0 {
		t.Fatalf("expected no delete users without vote")
	}
}

func TestInvestigationServiceVoteInvestigationDrawFinal(t *testing.T) {
	user1 := models.User{ID: uuid.New(), Username: "u1", NotificationEnabled: true, ChatID: 101}
	user2 := models.User{ID: uuid.New(), Username: "u2", NotificationEnabled: true, ChatID: 202}
	invID := uuid.New()
	disputeID := uuid.New()
	deps := &fakeInvestigationDeps{
		user:          user1,
		u2i:           models.User2Investigation{ID: uuid.New()},
		investigation: models.Investigation{InvestigationDB: models.InvestigationDB{ID: invID, DisputeID: disputeID, Total: 1, P1: 0, P2: 0, Draw: 0}},
		winners:       []uuid.UUID{user1.ID},
		disputeUsers:   []models.User{user1, user2},
		u2dByUser: map[uuid.UUID]models.User2Dispute{
			user1.ID: {ID: uuid.New()},
			user2.ID: {ID: uuid.New()},
		},
		dispute: models.Dispute{DisputeDB: models.DisputeDB{ID: disputeID, Title: "INV"}},
	}
	sender := &fakeMessageSender{}
	svc := InvestigationService{
		logger:               noopLogger{},
		userFinder:           deps,
		userUpdater:          deps,
		u2iFinder:            deps,
		u2iUpdater:           deps,
		investigationFinder:  deps,
		investigationUpdater: deps,
		investigationDeleter: deps,
		u2dGetter:            deps,
		u2dUpdater:           deps,
		disputeFinder:        deps,
		msgSender:            sender,
	}

	err := svc.VoteInvestigation(context.Background(), invID.String(), user1.Username, "draw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.updatedInv) != 2 {
		t.Fatalf("expected 2 investigation updates, got %d", len(deps.updatedInv))
	}
	if deps.deleteNoVoteCnt != 1 {
		t.Fatalf("expected delete users without vote once, got %d", deps.deleteNoVoteCnt)
	}
	if len(deps.updatedU2D) != 2 {
		t.Fatalf("expected 2 u2d updates, got %d", len(deps.updatedU2D))
	}
	if sender.calls != 2 {
		t.Fatalf("expected 2 messages for draw, got %d", sender.calls)
	}
}
