package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeEvidencer struct {
	err    error
	called bool
	opts   models.EvidenceOpts
}

func (f *fakeEvidencer) ProvideEvidence(_ context.Context, evidence models.EvidenceOpts) error {
	f.called = true
	f.opts = evidence
	return f.err
}

type fakeEvidenceGetter struct {
	err      error
	calledID string
	items    []models.Evidence
}

func (f *fakeEvidenceGetter) GetEvidences(_ context.Context, disputeID string) ([]models.Evidence, error) {
	f.calledID = disputeID
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestEvidenceDispute(t *testing.T) {
	t.Run("returns unauthorized when username missing", func(t *testing.T) {
		r := gin.New()
		r.POST("/disputes/:id/evidence", evidenceDispute(noopLogger{}, &fakeEvidencer{}))

		req := httptest.NewRequest(http.MethodPost, "/disputes/123/evidence", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("passes payload to service", func(t *testing.T) {
		evidencer := &fakeEvidencer{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/:id/evidence", evidenceDispute(noopLogger{}, evidencer))

		req := httptest.NewRequest(http.MethodPost, "/disputes/123/evidence", strings.NewReader("description=test evidence"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}
		if !evidencer.called {
			t.Fatal("expected ProvideEvidence to be called")
		}
		if evidencer.opts.DisputeID != "123" || evidencer.opts.Username != "alice" {
			t.Fatalf("unexpected args: %#v", evidencer.opts)
		}
		if evidencer.opts.Description != "test evidence" {
			t.Fatalf("expected description to be parsed, got %q", evidencer.opts.Description)
		}
	})
}

func TestGetEvidencesByDispute(t *testing.T) {
	t.Run("returns bad request when disputeID missing", func(t *testing.T) {
		r := gin.New()
		r.GET("/evidences", getEvidencesByDispute(noopLogger{}, &fakeEvidenceGetter{}))

		req := httptest.NewRequest(http.MethodGet, "/evidences", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("returns internal error on service failure", func(t *testing.T) {
		getter := &fakeEvidenceGetter{err: errors.New("boom")}
		r := gin.New()
		r.GET("/evidences", getEvidencesByDispute(noopLogger{}, getter))

		req := httptest.NewRequest(http.MethodGet, "/evidences?disputeID=123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("returns evidences on success", func(t *testing.T) {
		getter := &fakeEvidenceGetter{items: []models.Evidence{{ID: uuid.New()}}}
		r := gin.New()
		r.GET("/evidences", getEvidencesByDispute(noopLogger{}, getter))

		req := httptest.NewRequest(http.MethodGet, "/evidences?disputeID=123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		if getter.calledID != "123" {
			t.Fatalf("expected dispute id 123, got %q", getter.calledID)
		}
	})
}
