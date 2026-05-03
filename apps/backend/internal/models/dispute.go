package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation error")

type DisputeRead struct {
	ID              string    `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	Cryptocurrency  string  `json:"cryptocurrency"`
	AmountNano      int64   `json:"amountNano"`
	ImageData       []byte  `json:"imageData"`
	ImageType       *string `json:"imageType"`
	ContractAddress string  `json:"contractAddress"`
	EndsAt          time.Time `json:"endsAt"`
	NextDeadline    time.Time `json:"nextDeadline"`
	Opponent        string  `json:"opponent"`
	PhotoUrl        *string `json:"photoUrl"`
	Result          Result  `json:"result"`
	Vote            bool    `json:"vote"` // true for "win", false for "lose"
	Claim           bool    `json:"claim"` // true if user has claimed the dispute
}

type DisputeListOpts struct {
	Creator uuid.UUID
	Status  *Status
	Result  *bool
	Limit   int
	Cursor  string
}

type CreateDisputeReq struct {
	Title           string `form:"title" binding:"required"`
	Description     string `form:"description" binding:"required"`
	Opponent        string `form:"opponent" binding:"required"`
	AmountNano      string `form:"amountNano" binding:"required"`
	EndsAt          string `form:"endsAt" binding:"required"`
	ContractAddress string `form:"contractAddress" binding:"required"`
	Boc             string `form:"boc" binding:"required"`
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
