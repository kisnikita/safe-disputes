package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Dispute_old struct {
	ID              uuid.UUID `db:"id" json:"id"`
	Title           string    `db:"title" json:"title"`
	Description     string    `db:"description" json:"description"`
	Opponent        string    `db:"opponent" json:"opponent"`
	Result          Result    `db:"result" json:"result"`
	Claim           bool      `db:"claim" json:"claim"` // true if user has claimed the dispute
	Vote            bool      `db:"vote" json:"vote"`   // true for "win", false for "lose"
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
	Cryptocurrency  string    `db:"cryptocurrency" json:"cryptocurrency"`
	Amount          int       `db:"amount" json:"amount"`
	ImageData       []byte    `db:"image_data" json:"imageData"`
	ImageType       *string   `db:"image_type" json:"imageType"`
	ContractAddress string    `db:"contract_address" json:"contractAddress"`
}

type Dispute struct {
	DisputeDB
	Opponent        string `db:"opponent" json:"opponent"`
	Result          Result `db:"result" json:"result"`
	Vote            bool   `db:"vote" json:"vote"`
	Claim           bool   `db:"claim" json:"claim"`
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
	Amount          string `form:"amount" binding:"required"`
	ContractAddress string `form:"contractAddress" binding:"required"`
	ImageData       []byte
	ImageType       string
}

func NewDispute(opts CreateDisputeReq) Dispute {
	amount, _ := strconv.ParseInt(opts.Amount, 10, 32)
	d := Dispute{
		DisputeDB: DisputeDB {
			ID:              uuid.New(),
			Title:           opts.Title,
			Description:     opts.Description,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			Cryptocurrency:  "TON",
			Amount:          int(amount),
			ImageData:       opts.ImageData,
			ContractAddress: opts.ContractAddress,
		},
		Opponent:        opts.Opponent,
	}
	if opts.ImageType != "" {
		d.ImageType = &opts.ImageType
	}
	return d
}
