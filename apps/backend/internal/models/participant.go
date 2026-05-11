package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	DisputesStatusCurrent Status = "current"
	DisputesStatusNew     Status = "new"
	DisputesStatusPassed  Status = "passed"
)

type Result string

const (
	DisputesResultNew              Result = "new"
	DisputesResultSent             Result = "sent"
	DisputesResultProcessed        Result = "processed"
	DisputesResultAnswered         Result = "answered"
	DisputesResultEvidence         Result = "evidence"
	DisputesResultEvidenceAnswered Result = "evidence_answered"
	DisputesResultInspected        Result = "inspected"
	DisputesResultRejected         Result = "rejected"
	DisputesResultWin              Result = "win"
	DisputesResultLose             Result = "lose"
	DisputesResultDraw             Result = "draw"
)

type ParticipantUpdateOpts struct {
	ID          uuid.UUID `json:"id"`
	Status      *Status   `json:"status"`
	Result      *Result   `json:"result"`
	IsWin       *bool     `json:"isWin"`
	IsClaimable *bool     `json:"isClaimable"`
	Seen        *bool     `json:"-"`
}

func NewParticipant(userID, disputeID uuid.UUID, result Result, isCreator bool) Participant {
	return Participant{
		ID:        uuid.New(),
		UserID:    userID,
		DisputeID: disputeID,
		IsCreator: isCreator,
		Status:    DisputesStatusNew,
		Result:    result,
		UpdatedAt: time.Now(),
	}
}

func (p *Participant) MarkSeen() {
	p.SeenAt = &p.UpdatedAt
}

func (p Participant) CanClaim() bool {
	return p.IsClaimable &&
		(p.Result == DisputesResultRejected ||
			p.Result == DisputesResultLose ||
			p.Result == DisputesResultDraw ||
			p.Result == DisputesResultWin)
}
