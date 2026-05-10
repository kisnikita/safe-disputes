package models

import "github.com/google/uuid"

type InvestigationResult string

const (
	InvestigationResultNew       InvestigationResult = "new"
	InvestigationResultSent      InvestigationResult = "sent"
	InvestigationResultCorrect   InvestigationResult = "correct"
	InvestigationResultInCorrect InvestigationResult = "incorrect"
)

type JurorUpdateOpts struct {
	ID     uuid.UUID
	Vote   *string
	Result *InvestigationResult
	SeenAt *bool
}

func NewJuror(investigationID, userID uuid.UUID) Juror {
	return Juror{
		ID:              uuid.New(),
		InvestigationID: investigationID,
		UserID:          userID,
		Result:          InvestigationResultNew,
	}
}
