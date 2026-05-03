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

type DisputeParticipantUpdateOpts struct {
	ID     uuid.UUID `json:"id"`
	Status *Status   `json:"status"`
	Result *Result   `json:"result"`
	Vote   *bool     `json:"vote"`
	Claim  *bool     `json:"claim"`
}

func NewDisputeParticipant(userID, disputeID uuid.UUID, result Result) DisputeParticipant {
	return DisputeParticipant{
		ID:        uuid.New(),
		UserID:    userID,
		DisputeID: disputeID,
		Status:    DisputesStatusNew,
		Result:    result,
		Vote:      false, // Vote will be set when the user votes on the dispute
	}
}
