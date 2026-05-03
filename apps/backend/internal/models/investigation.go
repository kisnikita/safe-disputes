package models

import (
	"time"

	"github.com/google/uuid"
)

type InvestigationStatus string

const (
	InvestigationStatusCurrent InvestigationStatus = "current"
	InvestigationStatusPassed  InvestigationStatus = "passed"
)

type InvestigationRead struct {
	ID        string              `json:"id"`
	DisputeID string              `json:"disputeID"`
	Total     int                 `json:"total"`
	P1        int                 `json:"p1"`
	P2        int                 `json:"p2"`
	Draw      int                 `json:"draw"`
	Status    InvestigationStatus `json:"status"`
	CreatedAt time.Time           `json:"createdAt"`
	EndsAt    time.Time           `json:"endsAt"`
	Title     string              `json:"title"`
	Result    InvestigationResult `json:"result"`
	Vote      string              `json:"vote"`
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
		ID:        uuid.New(),
		DisputeID: disputeID,
		Total:     total,
		Status:    InvestigationStatusCurrent,
		EndsAt:    time.Now().Add(3 * time.Hour),
		Title:     title,
	}
}
