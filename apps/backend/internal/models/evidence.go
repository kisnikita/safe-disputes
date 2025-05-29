package models

import "github.com/google/uuid"

type Evidence struct {
	ID          uuid.UUID `db:"id" json:"id"`
	DisputeID   uuid.UUID `db:"dispute_id" json:"disputeID"`
	UserID      uuid.UUID `db:"user_id" json:"userID"`
	Description string    `db:"description" json:"description"`
	ImageData   []byte    `db:"image_data" json:"imageData"`
	ImageType   string    `db:"image_type" json:"imageType"`
}

type EvidenceOpts struct {
	DisputeID   string
	Username    string
	Description string
	ImageData   []byte
	ImageType   string
}

func NewEvidence(disputeID, userID uuid.UUID, description string, imageData []byte, imageType string) Evidence {
	return Evidence{
		ID:          uuid.New(),
		DisputeID:   disputeID,
		UserID:      userID,
		Description: description,
		ImageData:   imageData,
		ImageType:   imageType,
	}
}
