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

type InvestigationCard struct {
	ID        string              `db:"id"         json:"id"`
	DisputeID string              `db:"dispute_id" json:"disputeID"`
	Status    InvestigationStatus `db:"status"     json:"status"`
	CreatedAt time.Time           `db:"created_at" json:"createdAt"`
	EndsAt    time.Time           `db:"ends_at"    json:"endsAt"`
	Title     string              `db:"title"      json:"title"`
	Result    InvestigationResult `db:"result"     json:"result"`
	Vote      string              `db:"vote"       json:"vote"`
}

type InvestigationDetails struct {
	ID        string              `db:"id"         json:"id"`
	DisputeID string              `db:"dispute_id" json:"disputeID"`
	Total     int                 `db:"total"      json:"total"`
	P1        int                 `db:"p1"         json:"p1"`
	P2        int                 `db:"p2"         json:"p2"`
	Draw      int                 `db:"draw"       json:"draw"`
	Status    InvestigationStatus `db:"status"     json:"status"`
	CreatedAt time.Time           `db:"created_at" json:"createdAt"`
	EndsAt    time.Time           `db:"ends_at"    json:"endsAt"`
	Title     string              `db:"title"      json:"title"`
	Result    InvestigationResult `db:"result"     json:"result"`
	Vote      string              `db:"vote"       json:"vote"`
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
