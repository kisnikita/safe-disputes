package models

import "github.com/google/uuid"

type Status string

const (
	DisputesStatusNew     Status = "new"
	DisputesStatusCurrent Status = "current"
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

type User2Dispute struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	DisputeID uuid.UUID `db:"dispute_id" json:"dispute_id"`
	Status    Status    `db:"status" json:"status"` // "new", "waiting", "pending", "rejected", "finish"

	Result Result `db:"result" json:"result"` // "sent", "processed", "answered", "evidence", "inspected" ||
	// "rejected", "win", "lose", "draw"

	Vote  bool `db:"vote" json:"vote"`   // true for "win", false for "lose"
	Claim bool `db:"claim" json:"claim"` // true if user has claimed the dispute
}

type U2DUpdateOpts struct {
	ID     uuid.UUID `json:"id"`
	Status *Status   `json:"status"`
	Result *Result   `json:"result"`
	Vote   *bool     `json:"vote"`
	Claim  *bool     `json:"claim"`
}

func NewUser2Dispute(userID, disputeID uuid.UUID, status Status, result Result) User2Dispute {
	return User2Dispute{
		ID:        uuid.New(),
		UserID:    userID,
		DisputeID: disputeID,
		Status:    status,
		Result:    result,
		Vote:      false, // Vote will be set when the user votes on the dispute
	}
}
