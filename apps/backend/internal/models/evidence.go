package models

import "github.com/google/uuid"

type EvidenceOpts struct {
	DisputeID   string
	Username    string
	Description string
	ImageData   []byte
	ImageType   string
}

func NewEvidence(disputeID, userID uuid.UUID, description string, imageData []byte, imageType string) Evidence {
	e := Evidence{
		ID:          uuid.New(),
		DisputeID:   disputeID,
		UserID:      userID,
		Description: description,
		ImageData:   imageData,
	}
	if imageType != "" {
		e.ImageType = &imageType
	}
	return e
}
