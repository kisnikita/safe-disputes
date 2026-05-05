package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation error")

type DisputeCard struct {
	ID           string    `db:"id"            json:"id"`
	Title        string    `db:"title"         json:"title"`
	CreatedAt    time.Time `db:"created_at"    json:"createdAt"`
	AmountNano   int64     `db:"amount_nano"   json:"amountNano"`
	EndsAt       time.Time `db:"ends_at"       json:"endsAt"`
	NextDeadline time.Time `db:"next_deadline" json:"nextDeadline"`
	Opponent     string    `db:"opponent"      json:"opponent"`
	PhotoUrl     *string   `db:"photo_url"     json:"photoUrl"`
	Result       Result    `db:"result"        json:"result"`
	Vote         bool      `db:"vote"          json:"vote"`  // true for "win", false for "lose"
	Claim        bool      `db:"claim"         json:"claim"` // true if user has claimed the dispute
}

type DisputeDetails struct {
	ID              string    `db:"id"               json:"id"`
	Title           string    `db:"title"            json:"title"`
	Description     string    `db:"description"      json:"description"`
	CreatedAt       time.Time `db:"created_at"       json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at"       json:"updatedAt"`
	Cryptocurrency  string    `db:"cryptocurrency"   json:"cryptocurrency"`
	AmountNano      int64     `db:"amount_nano"      json:"amountNano"`
	ImageData       []byte    `db:"image_data"       json:"imageData"`
	ImageType       *string   `db:"image_type"       json:"imageType"`
	ContractAddress string    `db:"contract_address" json:"contractAddress"`
	EndsAt          time.Time `db:"ends_at"          json:"endsAt"`
	NextDeadline    time.Time `db:"next_deadline"    json:"nextDeadline"`
	Opponent        string    `db:"opponent"         json:"opponent"`
	PhotoUrl        *string   `db:"photo_url"        json:"photoUrl"`
	Result          Result    `db:"result"           json:"result"`
	Vote            bool      `db:"vote"             json:"vote"`  // true for "win", false for "lose"
	Claim           bool      `db:"claim"            json:"claim"` // true if user has claimed the dispute
}

type DisputeListOpts struct {
	Creator uuid.UUID
	Status  *Status
	Result  *bool
	Limit   int
	Cursor  string
}

type CreateDisputeReq struct {
	Title           string `form:"title"           binding:"required"`
	Description     string `form:"description"     binding:"required"`
	Opponent        string `form:"opponent"        binding:"required"`
	AmountNano      string `form:"amountNano"      binding:"required"`
	EndsAt          string `form:"endsAt"          binding:"required"`
	ContractAddress string `form:"contractAddress" binding:"required"`
	Boc             string `form:"boc"             binding:"required"`
	ImageData       []byte
	ImageType       string
}

func NewDispute(opts CreateDisputeReq) (Dispute, error) {
	amountNano, err := ParsePositiveNano(opts.AmountNano)
	if err != nil {
		return Dispute{}, fmt.Errorf("%w: %w", ErrValidation, err)
	}
	endsAt, err := time.Parse(time.RFC3339, opts.EndsAt)
	if err != nil {
		return Dispute{}, fmt.Errorf("%w: %w", ErrValidation, err)
	}
	if !endsAt.After(time.Now()) {
		return Dispute{}, fmt.Errorf("%w: endsAt must be in the future", ErrValidation)
	}

	createdAt := time.Now()
	acceptanceDeadline := createdAt.Add(24 * time.Hour)
	nextDeadline := acceptanceDeadline
	if endsAt.Before(acceptanceDeadline) {
		nextDeadline = endsAt
	}
	d := Dispute{
		ID:              uuid.New(),
		Title:           opts.Title,
		Description:     opts.Description,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
		Cryptocurrency:  "TON",
		AmountNano:      amountNano,
		ImageData:       opts.ImageData,
		ContractAddress: opts.ContractAddress,
		EndsAt:          endsAt,
		NextDeadline:    nextDeadline,
	}
	if opts.ImageType != "" {
		d.ImageType = &opts.ImageType
	}
	return d, nil
}
