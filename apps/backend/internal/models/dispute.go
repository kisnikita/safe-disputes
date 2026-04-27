package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Dispute struct {
	DisputeDB
	Opponent string  `json:"opponent"`
	PhotoUrl *string `json:"photoUrl"`
	Result   Result  `db:"result" json:"result"`
	Vote     bool    `db:"vote" json:"vote"`   // true for "win", false for "lose"
	Claim    bool    `db:"claim" json:"claim"` // true if user has claimed the dispute
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
		return Dispute{}, fmt.Errorf("invalid amount: %w", err)
	}
	endsAt, err := time.Parse(time.RFC3339, opts.EndsAt)
	if err != nil {
		return Dispute{}, err
	}
	if !endsAt.After(time.Now()) {
		return Dispute{}, fmt.Errorf("endsAt must be in the future")
	}

	createdAt := time.Now()
	acceptanceDeadline := createdAt.Add(24 * time.Hour)
	nextDeadline := acceptanceDeadline
	if endsAt.Before(acceptanceDeadline) {
		nextDeadline = endsAt
	}
	d := Dispute{
		DisputeDB: DisputeDB{
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
		},
		Opponent: opts.Opponent,
	}
	if opts.ImageType != "" {
		d.ImageType = &opts.ImageType
	}
	return d, nil
}
