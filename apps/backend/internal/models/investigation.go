package models

import (
	"github.com/google/uuid"
	"time"
)

type InvestigationStatus string

const (
	InvestigationStatusCurrent InvestigationStatus = "current"
	InvestigationStatusPassed  InvestigationStatus = "passed"
)

type Investigation struct {
	InvestigationDB
	Result InvestigationResult `db:"result" json:"result"`
	Vote   string              `db:"vote" json:"vote"`
}

type InvestigationListOpts struct {
	UserID uuid.UUID
	Status *InvestigationStatus
	Limit  int
	Cursor string
}

type InvestigationUpdateOpts struct {
	ID     uuid.UUID            `json:"id"`
	Status *InvestigationStatus `json:"status"`
	P1     *int
	P2     *int
	Draw   *int
	Total  *int
}

func NewInvestigation(disputeID uuid.UUID, total int, title string) Investigation {
	return Investigation{
		InvestigationDB: InvestigationDB{
			ID:        uuid.New(),
			DisputeID: disputeID,
			Total:     total,
			Status:    InvestigationStatusCurrent,
			EndsAt:    time.Now().Add(3 * time.Hour),
			Title:     title,
		},
	}
}
