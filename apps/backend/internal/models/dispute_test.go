package models

import (
	"testing"
	"time"
)

func TestNewDisputeSetsNextDeadlineTo24HoursWhenEndsAtLater(t *testing.T) {
	dispute, err := NewDispute(CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		Amount:          "100",
		EndsAt:          time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := dispute.NextDeadline.Sub(dispute.CreatedAt)
	if got < 23*time.Hour+59*time.Minute || got > 24*time.Hour+time.Minute {
		t.Fatalf("expected next deadline around 24h, got %s", got)
	}
}

func TestNewDisputeSetsNextDeadlineToEndsAtWhenEndsAtSooner(t *testing.T) {
	endsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	dispute, err := NewDispute(CreateDisputeReq{
		Title:           "t",
		Description:     "d",
		Opponent:        "bob",
		Amount:          "100",
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
		Amount:          "100",
		EndsAt:          time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		ContractAddress: "addr",
		Boc:             "boc",
	})
	if err == nil {
		t.Fatal("expected error for past endsAt")
	}
}
