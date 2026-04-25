package models

import "github.com/google/uuid"

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
