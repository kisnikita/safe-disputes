package models

import (
	"testing"
	"time"
)

func TestNewDisputeSetsNextDeadlineToEndsAt(t *testing.T) {
	dispute, err := NewDispute(CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !dispute.NextDeadline.Equal(dispute.EndsAt) {
		t.Fatalf("expected next deadline equal to endsAt, got next=%s endsAt=%s", dispute.NextDeadline, dispute.EndsAt)
	}
}

func TestNewDisputeSetsNextDeadlineToEndsAtWhenEndsAtSooner(t *testing.T) {
	endsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	dispute, err := NewDispute(CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          endsAt,
		ContractAddress: "addr",
		Boc:             "boc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !dispute.NextDeadline.Equal(dispute.EndsAt) {
		t.Fatalf("expected next deadline equal to endsAt, got next=%s endsAt=%s", dispute.NextDeadline, dispute.EndsAt)
	}
}

func TestNewDisputeRejectsPastEndsAt(t *testing.T) {
	_, err := NewDispute(CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		AmountNano:      "100000000000",
		DepositNano:     "20000000000",
		EndsAt:          time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	})
	if err == nil {
		t.Fatal("expected error for past endsAt")
	}
}
