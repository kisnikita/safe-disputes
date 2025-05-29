package models

import "github.com/google/uuid"

type InvestigationResult string

const (
	InvestigationResultNew       InvestigationResult = "new"
	InvestigationResultSent      InvestigationResult = "sent"
	InvestigationResultCorrect   InvestigationResult = "correct"
	InvestigationResultInCorrect InvestigationResult = "incorrect"
)

type User2Investigation struct {
	ID              uuid.UUID           `db:"id" json:"id"`
	InvestigationID uuid.UUID           `db:"investigation_id" json:"investigation_id"`
	UserID          uuid.UUID           `db:"user_id" json:"user_id"`
	Vote            string              `db:"vote" json:"vote"`
	Result          InvestigationResult `db:"result" json:"result"`
}

type U2IUpdateOpts struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Vote   *string
	Result *InvestigationResult
}

func NewUser2Investigation(investigationID, userID uuid.UUID) User2Investigation {
	return User2Investigation{
		ID:              uuid.New(),
		InvestigationID: investigationID,
		UserID:          userID,
		Result:          InvestigationResultNew,
		Vote:            "",
	}
}
