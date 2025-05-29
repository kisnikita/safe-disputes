package models

import (
	"github.com/google/uuid"
	"time"
)

type Dispute struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Title          string    `db:"title" json:"title"`
	Description    string    `db:"description" json:"description"`
	Opponent       string    `db:"opponent" json:"opponent"`
	Result         Result    `db:"result" json:"result"`
	Claim          bool      `db:"claim" json:"claim"` // true if user has claimed the dispute
	Vote           bool      `db:"vote" json:"vote"`   // true for "win", false for "lose"
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
	Cryptocurrency string    `db:"cryptocurrency" json:"cryptocurrency"`
	Amount         int       `db:"amount" json:"amount"`
	ImageData      []byte    `db:"image_data" json:"imageData"`
	ImageType      string    `db:"image_type" json:"imageType"`
}

type DisputeListOpts struct {
	Creator uuid.UUID
	Status  *Status
	Result  *bool
	Limit   int
	Cursor  string
}

func NewDispute(title, description, opponent string, amount int, imageData []byte, imageType string) Dispute {
	return Dispute{
		ID:             uuid.New(),
		Title:          title,
		Description:    description,
		Opponent:       opponent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Cryptocurrency: "TON",
		Amount:         amount,
		ImageData:      imageData,
		ImageType:      imageType,
	}
}
