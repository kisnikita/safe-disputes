package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeInvestigationDeps struct {
	user             models.User
	participant      models.Juror
	investigation    models.Investigation
	winners          []uuid.UUID
	disputeUsers     []models.User
	participantByUser        map[uuid.UUID]models.Participant
	dispute          models.Dispute
	listResult       []models.InvestigationRead
	getResult        models.InvestigationRead
	listReceivedOpts models.InvestigationListOpts
	listActorUsername string
	getActorUsername  string

	updatedParticipants []models.JurorUpdateOpts
	updatedInv      []models.InvestigationUpdateOpts
	updatedDP      []models.ParticipantUpdateOpts
	deleteNoVoteCnt int
	earnWinnerCnt   int
	updateWinnerCnt int
}

func (f *fakeInvestigationDeps) InsertInvestigation(context.Context, models.Investigation) error {
	return nil
}
func (f *fakeInvestigationDeps) ListInvestigations(_ context.Context, opts models.InvestigationListOpts) ([]models.Investigation, error) {
	f.listReceivedOpts = opts
	return nil, nil
}
func (f *fakeInvestigationDeps) ListInvestigationReads(_ context.Context, actorUsername string, opts models.InvestigationListOpts) ([]models.InvestigationRead, error) {
	f.listReceivedOpts = opts
	f.listActorUsername = actorUsername
	return f.listResult, nil
}
func (f *fakeInvestigationDeps) GetInvestigationRead(_ context.Context, _ uuid.UUID, actorUsername string) (models.InvestigationRead, error) {
	f.getActorUsername = actorUsername
	return f.getResult, nil
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
func (f *fakeInvestigationDeps) GetJuror(context.Context, uuid.UUID, uuid.UUID) (models.Juror, error) {
	return f.participant, nil
}
func (f *fakeInvestigationDeps) GetWinnersIDs(context.Context, uuid.UUID, string) ([]uuid.UUID, error) {
	return f.winners, nil
}
func (f *fakeInvestigationDeps) GetDisputesUsers(context.Context, uuid.UUID) ([]models.User, error) {
	return f.disputeUsers, nil
}
func (f *fakeInvestigationDeps) UpdateJuror(_ context.Context, opts models.JurorUpdateOpts) error {
	f.updatedParticipants = append(f.updatedParticipants, opts)
	return nil
}
func (f *fakeInvestigationDeps) UpdateWinnersResult(context.Context, uuid.UUID, []uuid.UUID) error {
	f.updateWinnerCnt++
	return nil
}
func (f *fakeInvestigationDeps) GetUserByID(context.Context, uuid.UUID) (models.User, error) {
	return models.User{}, nil
}
func (f *fakeInvestigationDeps) GetUserByUsername(context.Context, string) (models.User, error) {
	return f.user, nil
}
func (f *fakeInvestigationDeps) ExistByUsername(context.Context, string) (bool, error) {
	return false, nil
}
func (f *fakeInvestigationDeps) GetTotalUsers(context.Context) (int, error) { return 0, nil }
func (f *fakeInvestigationDeps) GetUsers(context.Context, []uuid.UUID) ([]models.User, error) {
	return nil, nil
}
func (f *fakeInvestigationDeps) GetTopUsers(context.Context, int) ([]models.User, error) {
	return nil, nil
}
func (f *fakeInvestigationDeps) UpdateUser(context.Context, models.UserUpdateOpts) error { return nil }
func (f *fakeInvestigationDeps) UpdateUserPhotoURL(context.Context, string, *string) error {
	return nil
}
func (f *fakeInvestigationDeps) EarnWinnerRating(context.Context, []uuid.UUID) error {
	f.earnWinnerCnt++
	return nil
}
func (f *fakeInvestigationDeps) UpdateParticipant(_ context.Context, opts models.ParticipantUpdateOpts) error {
	f.updatedDP = append(f.updatedDP, opts)
	return nil
}
func (f *fakeInvestigationDeps) GetParticipant(_ context.Context, _ uuid.UUID, userID uuid.UUID) (models.Participant, error) {
	return f.participantByUser[userID], nil
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
		listResult: []models.InvestigationRead{{ID: uuid.New().String()}},
	}
	svc := InvestigationService{logger: noopLogger{}, investigationReadFinder: deps}

	res, err := svc.ListInvestigation(context.Background(), models.InvestigationListOpts{Limit: 5}, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if deps.listActorUsername != "alice" {
		t.Fatalf("expected actor username to be propagated")
	}
}

func TestInvestigationServiceGetInvestigationInvalidID(t *testing.T) {
	deps := &fakeInvestigationDeps{user: models.User{ID: uuid.New(), Username: "alice"}}
	svc := InvestigationService{logger: noopLogger{}, investigationReadFinder: deps}

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
		participant:   models.Juror{ID: uuid.New()},
		investigation: models.Investigation{ID: invID, DisputeID: uuid.New(), Total: 3, P1: 1, P2: 0, Draw: 0},
	}
	svc := InvestigationService{
		logger:               noopLogger{},
		userFinder:           deps,
		jurorFinder:  deps,
		jurorUpdater: deps,
		userUpdater:          deps,
		investigationFinder:  deps,
		investigationUpdater: deps,
	}

	err := svc.VoteInvestigation(context.Background(), invID.String(), "alice", "p2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.updatedParticipants) != 1 {
		t.Fatalf("expected 1 participant update, got %d", len(deps.updatedParticipants))
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
		participant:   models.Juror{ID: uuid.New()},
		investigation: models.Investigation{ID: invID, DisputeID: disputeID, Total: 1, P1: 0, P2: 0, Draw: 0},
		winners:       []uuid.UUID{user1.ID},
		disputeUsers:  []models.User{user1, user2},
		participantByUser: map[uuid.UUID]models.Participant{
			user1.ID: {ID: uuid.New()},
			user2.ID: {ID: uuid.New()},
		},
		dispute: models.Dispute{ID: disputeID, Title: "INV"},
	}
	sender := &fakeMessageSender{}
	svc := InvestigationService{
		logger:               noopLogger{},
		userFinder:           deps,
		userUpdater:          deps,
		jurorFinder:  deps,
		jurorUpdater: deps,
		investigationFinder:  deps,
		investigationUpdater: deps,
		investigationDeleter: deps,
		participantGetter:            deps,
		participantUpdater:           deps,
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
	if len(deps.updatedDP) != 2 {
		t.Fatalf("expected 2 participant updates, got %d", len(deps.updatedDP))
	}
	if sender.calls != 2 {
		t.Fatalf("expected 2 messages for draw, got %d", sender.calls)
	}
}
