package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/services"
)

type fakeDisputeCreator struct {
	err        error
	called     bool
	creator    string
	boc        string
	disputeArg models.Dispute
}

func (f *fakeDisputeCreator) CreateDispute(_ context.Context, dispute models.Dispute, creatorUsername, boc string) error {
	f.called = true
	f.creator = creatorUsername
	f.boc = boc
	f.disputeArg = dispute
	return f.err
}

type fakeDisputePrechecker struct {
	err      error
	called   bool
	opponent string
	amount   int
	creator  string
}

func (f *fakeDisputePrechecker) PrecheckCreateDispute(_ context.Context, opponent string, amount int, creatorUsername string) error {
	f.called = true
	f.opponent = opponent
	f.amount = amount
	f.creator = creatorUsername
	return f.err
}

type fakeDisputeLister struct {
	err      error
	disputes []models.Dispute
	called   bool
	opts     models.DisputeListOpts
	creator  string
}

func (f *fakeDisputeLister) ListDisputes(_ context.Context, opts models.DisputeListOpts, creatorUsername string) ([]models.Dispute, error) {
	f.called = true
	f.opts = opts
	f.creator = creatorUsername
	if f.err != nil {
		return nil, f.err
	}
	return f.disputes, nil
}

type fakeDisputeGetter struct {
	err            error
	dispute        models.Dispute
	calledID       string
	calledUsername string
}

func (f *fakeDisputeGetter) GetDispute(_ context.Context, disputeID string, creatorUsername string) (models.Dispute, error) {
	f.calledID = disputeID
	f.calledUsername = creatorUsername
	if f.err != nil {
		return models.Dispute{}, f.err
	}
	return f.dispute, nil
}

func (f *fakeDisputeGetter) GetDisputeForEvidence(_ context.Context, disputeID string) (models.Dispute, error) {
	if f.err != nil {
		return models.Dispute{}, f.err
	}
	return f.dispute, nil
}

type fakeDisputeVoter struct {
	err      error
	called   bool
	id       string
	username string
	vote     bool
}

func (f *fakeDisputeVoter) VoteDispute(_ context.Context, disputeID string, username string, win bool) error {
	f.called = true
	f.id = disputeID
	f.username = username
	f.vote = win
	return f.err
}

func TestCreateDispute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		creator := &fakeDisputeCreator{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes", createDispute(noopLogger{}, creator))

		form := url.Values{}
		form.Set("title", "test")
		form.Set("description", "desc")
		form.Set("opponent", "bob")
		form.Set("amount", "100")
		form.Set("endsAt", time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339))
		form.Set("contractAddress", "addr")
		form.Set("boc", "te6cckEBAQEAAgAAAA==")

		req := httptest.NewRequest(http.MethodPost, "/disputes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
		}
		if !creator.called {
			t.Fatal("expected CreateDispute to be called")
		}
		if creator.creator != "alice" {
			t.Fatalf("expected creator alice, got %q", creator.creator)
		}
		if creator.disputeArg.Amount != 100 {
			t.Fatalf("expected amount 100, got %d", creator.disputeArg.Amount)
		}
		if creator.boc == "" {
			t.Fatal("expected boc to be passed")
		}
	})

	t.Run("invalid past endsAt returns bad request", func(t *testing.T) {
		creator := &fakeDisputeCreator{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes", createDispute(noopLogger{}, creator))

		form := url.Values{}
		form.Set("title", "test")
		form.Set("description", "desc")
		form.Set("opponent", "bob")
		form.Set("amount", "100")
		form.Set("endsAt", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339))
		form.Set("contractAddress", "addr")
		form.Set("boc", "te6cckEBAQEAAgAAAA==")

		req := httptest.NewRequest(http.MethodPost, "/disputes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
		if creator.called {
			t.Fatal("expected CreateDispute not to be called")
		}
	})

	t.Run("maps tx failure to conflict", func(t *testing.T) {
		creator := &fakeDisputeCreator{err: fmt.Errorf("%w: details", services.ErrTxFailed)}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes", createDispute(noopLogger{}, creator))

		form := url.Values{}
		form.Set("title", "test")
		form.Set("description", "desc")
		form.Set("opponent", "bob")
		form.Set("amount", "100")
		form.Set("endsAt", time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339))
		form.Set("contractAddress", "addr")
		form.Set("boc", "te6cckEBAQEAAgAAAA==")

		req := httptest.NewRequest(http.MethodPost, "/disputes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected %d, got %d", http.StatusConflict, rr.Code)
		}
	})
}

func TestPrecheckDispute(t *testing.T) {
	t.Run("maps user not found to 404", func(t *testing.T) {
		prechecker := &fakeDisputePrechecker{err: services.ErrUserNotFound}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/precheck", precheckDispute(noopLogger{}, prechecker))

		req := httptest.NewRequest(http.MethodPost, "/disputes/precheck", strings.NewReader(`{"opponent":"bob","amount":100}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("returns no content and passes args", func(t *testing.T) {
		prechecker := &fakeDisputePrechecker{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/precheck", precheckDispute(noopLogger{}, prechecker))

		req := httptest.NewRequest(http.MethodPost, "/disputes/precheck", strings.NewReader(`{"opponent":"bob","amount":100}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}
		if !prechecker.called {
			t.Fatal("expected PrecheckCreateDispute to be called")
		}
		if prechecker.creator != "alice" {
			t.Fatalf("expected creator alice, got %q", prechecker.creator)
		}
		if prechecker.opponent != "bob" || prechecker.amount != 100 {
			t.Fatalf("unexpected args: opponent=%q amount=%d", prechecker.opponent, prechecker.amount)
		}
	})

	t.Run("maps self opponent to conflict", func(t *testing.T) {
		prechecker := &fakeDisputePrechecker{err: services.ErrSelfOpponent}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/precheck", precheckDispute(noopLogger{}, prechecker))

		req := httptest.NewRequest(http.MethodPost, "/disputes/precheck", strings.NewReader(`{"opponent":"alice","amount":100}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected %d, got %d", http.StatusConflict, rr.Code)
		}
	})
}

func TestListDisputes(t *testing.T) {
	t.Run("returns bad request for invalid result query", func(t *testing.T) {
		lister := &fakeDisputeLister{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/disputes", listDisputes(noopLogger{}, lister))

		req := httptest.NewRequest(http.MethodGet, "/disputes?result=not-bool", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
		if lister.called {
			t.Fatal("lister should not be called")
		}
	})

	t.Run("returns paginated response", func(t *testing.T) {
		t1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 3, 1, 11, 0, 0, 123, time.UTC)
		lister := &fakeDisputeLister{disputes: []models.Dispute{
			{DisputeDB: models.DisputeDB{ID: uuid.New(), Title: "d1", CreatedAt: t1}},
			{DisputeDB: models.DisputeDB{ID: uuid.New(), Title: "d2", CreatedAt: t2}},
		}}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/disputes", listDisputes(noopLogger{}, lister))

		req := httptest.NewRequest(http.MethodGet, "/disputes?limit=1&status=current", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		if lister.creator != "alice" {
			t.Fatalf("expected creator alice, got %q", lister.creator)
		}
		if lister.opts.Limit != 1 {
			t.Fatalf("expected limit 1, got %d", lister.opts.Limit)
		}

		body := decodeJSONMap(t, rr)
		data := body["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("expected 1 item on page, got %d", len(data))
		}

		nextCursor, ok := body["nextCursor"].(string)
		if !ok || nextCursor == "" {
			t.Fatalf("expected non-empty nextCursor, got %#v", body["nextCursor"])
		}
	})
}

func TestGetDispute(t *testing.T) {
	t.Run("returns unauthorized when username missing", func(t *testing.T) {
		r := gin.New()
		r.GET("/disputes/:id", getDispute(noopLogger{}, &fakeDisputeGetter{}))

		req := httptest.NewRequest(http.MethodGet, "/disputes/123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("returns internal server error on getter failure", func(t *testing.T) {
		getter := &fakeDisputeGetter{err: errors.New("boom")}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/disputes/:id", getDispute(noopLogger{}, getter))

		req := httptest.NewRequest(http.MethodGet, "/disputes/123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

func TestVoteDispute(t *testing.T) {
	t.Run("returns bad request for invalid json body", func(t *testing.T) {
		voter := &fakeDisputeVoter{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/:id/vote", voteDispute(noopLogger{}, voter))

		req := httptest.NewRequest(http.MethodPost, "/disputes/123/vote", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
		if voter.called {
			t.Fatal("voter should not be called")
		}
	})

	t.Run("passes parsed vote to service", func(t *testing.T) {
		voter := &fakeDisputeVoter{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.POST("/disputes/:id/vote", voteDispute(noopLogger{}, voter))

		req := httptest.NewRequest(http.MethodPost, "/disputes/123/vote", strings.NewReader(`{"vote":true}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}
		if !voter.called {
			t.Fatal("expected VoteDispute to be called")
		}
		if voter.id != "123" || voter.username != "alice" || !voter.vote {
			t.Fatalf("unexpected call args: id=%q user=%q vote=%v", voter.id, voter.username, voter.vote)
		}
	})
}
