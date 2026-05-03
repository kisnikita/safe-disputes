package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

type fakeInvestigationLister struct {
	err   error
	items []models.InvestigationRead
	opts  models.InvestigationListOpts
	user  string
}

func (f *fakeInvestigationLister) ListInvestigation(_ context.Context, opts models.InvestigationListOpts, username string) ([]models.InvestigationRead, error) {
	f.opts = opts
	f.user = username
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

type fakeInvestigationGetter struct {
	err  error
	item models.InvestigationRead
	id   string
	user string
}

func (f *fakeInvestigationGetter) GetInvestigation(_ context.Context, id, username string) (models.InvestigationRead, error) {
	f.id = id
	f.user = username
	if f.err != nil {
		return models.InvestigationRead{}, f.err
	}
	return f.item, nil
}

type fakeInvestigationVoter struct {
	err      error
	id       string
	username string
	vote     string
}

func (f *fakeInvestigationVoter) VoteInvestigation(_ context.Context, id, username, vote string) error {
	f.id = id
	f.username = username
	f.vote = vote
	return f.err
}

func TestListInvestigations(t *testing.T) {
	t.Run("returns unauthorized when username missing", func(t *testing.T) {
		r := gin.New()
		r.GET("/investigations", listInvestigations(noopLogger{}, &fakeInvestigationLister{}))

		req := httptest.NewRequest(http.MethodGet, "/investigations", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("returns paginated response", func(t *testing.T) {
		t1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 3, 1, 10, 1, 0, 0, time.UTC)
		lister := &fakeInvestigationLister{items: []models.InvestigationRead{
			{ID: uuid.New().String(), Title: "i1", CreatedAt: t1},
			{ID: uuid.New().String(), Title: "i2", CreatedAt: t2},
		}}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/investigations", listInvestigations(noopLogger{}, lister))

		req := httptest.NewRequest(http.MethodGet, "/investigations?limit=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		if lister.user != "alice" {
			t.Fatalf("expected user alice, got %q", lister.user)
		}
		if lister.opts.Limit != 1 {
			t.Fatalf("expected limit 1, got %d", lister.opts.Limit)
		}
	})
}

func TestGetInvestigations(t *testing.T) {
	getter := &fakeInvestigationGetter{err: errors.New("boom")}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("username", "alice")
		c.Next()
	})
	r.GET("/investigations/:id", getInvestigations(noopLogger{}, getter))

	req := httptest.NewRequest(http.MethodGet, "/investigations/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestVoteInvestigations(t *testing.T) {
	voter := &fakeInvestigationVoter{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("username", "alice")
		c.Next()
	})
	r.POST("/investigations/:id/vote", voteInvestigations(noopLogger{}, voter))

	req := httptest.NewRequest(http.MethodPost, "/investigations/123/vote?vote=p1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}
	if voter.id != "123" || voter.username != "alice" || voter.vote != "p1" {
		t.Fatalf("unexpected call args: id=%q user=%q vote=%q", voter.id, voter.username, voter.vote)
	}
}
