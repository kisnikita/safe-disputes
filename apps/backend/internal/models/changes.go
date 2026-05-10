package models

type DisputeChange struct {
	DisputeID string `json:"disputeID"`
	Status    Status `json:"status"`
}

type InvestigationChange struct {
	InvestigationID string              `json:"investigationID"`
	Status          InvestigationStatus `json:"status"`
}

type ChangesList struct {
	Disputes       []DisputeChange
	Investigations []InvestigationChange
	MaxUpdatedAt   string
}

type DisputeUnreadCounts struct {
	New     int `json:"new"`
	Current int `json:"current"`
	Passed  int `json:"passed"`
}

type InvestigationUnreadCounts struct {
	Current int `json:"current"`
	Passed  int `json:"passed"`
}

type ChangesUnreadCounts struct {
	Disputes       DisputeUnreadCounts       `json:"disputes"`
	Investigations InvestigationUnreadCounts `json:"investigations"`
}
