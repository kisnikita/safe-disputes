package models

import "github.com/google/uuid"

type InvestigationResult string

const (
	InvestigationResultNew       InvestigationResult = "new"
	InvestigationResultSent      InvestigationResult = "sent"
	InvestigationResultCorrect   InvestigationResult = "correct"
	InvestigationResultInCorrect InvestigationResult = "incorrect"
)

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
