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

type fakeUserGetter struct {
	user       models.User
	users      []models.User
	errByUser  error
	errGetTop  error
	gotUser    string
	gotTop     int
	getMeCalls int
}

func (f *fakeUserGetter) GetByUsername(_ context.Context, username string) (models.User, error) {
	f.getMeCalls++
	f.gotUser = username
	if f.errByUser != nil {
		return models.User{}, f.errByUser
	}
	return f.user, nil
}

func (f *fakeUserGetter) GetTop(_ context.Context, limit int) ([]models.User, error) {
	f.gotTop = limit
	if f.errGetTop != nil {
		return nil, f.errGetTop
	}
	return f.users, nil
}

type fakeUserUpdater struct {
	err    error
	called bool
	opts   models.UserUpdateOpts
}

func (f *fakeUserUpdater) UpdateByUsername(_ context.Context, opts models.UserUpdateOpts) error {
	f.called = true
	f.opts = opts
	return f.err
}

func TestGetMe(t *testing.T) {
	t.Run("returns unauthorized when username missing", func(t *testing.T) {
		r := gin.New()
		r.GET("/me", getMe(noopLogger{}, &fakeUserGetter{}))

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("returns internal server error on getter failure", func(t *testing.T) {
		getter := &fakeUserGetter{errByUser: errors.New("boom")}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/me", getMe(noopLogger{}, getter))

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("returns user on success", func(t *testing.T) {
		id := uuid.New()
		getter := &fakeUserGetter{user: models.User{ID: id, Username: "alice"}}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/me", getMe(noopLogger{}, getter))

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		if getter.gotUser != "alice" {
			t.Fatalf("expected username alice, got %q", getter.gotUser)
		}

		jsonBody := decodeJSONMap(t, rr)
		data := jsonBody["data"].(map[string]any)
		if data["username"] != "alice" {
			t.Fatalf("expected username alice in response, got %#v", data["username"])
		}
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("returns bad request for invalid json", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.PATCH("/me", updateUser(noopLogger{}, &fakeUserUpdater{}))

		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("sets username from context and returns no content", func(t *testing.T) {
		updater := &fakeUserUpdater{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.PATCH("/me", updateUser(noopLogger{}, updater))

		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"notificationEnabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}
		if !updater.called {
			t.Fatal("expected updater to be called")
		}
		if updater.opts.Username != "alice" {
			t.Fatalf("expected username alice, got %q", updater.opts.Username)
		}
	})

	t.Run("passes investigation readiness to updater", func(t *testing.T) {
		updater := &fakeUserUpdater{}
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.PATCH("/me", updateUser(noopLogger{}, updater))

		req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"investigationReadiness":false}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
		}
		if updater.opts.InvestigationReadiness == nil || *updater.opts.InvestigationReadiness {
			t.Fatalf("expected investigationReadiness=false, got %#v", updater.opts.InvestigationReadiness)
		}
	})
}

func TestGetTop(t *testing.T) {
	getter := &fakeUserGetter{users: []models.User{{Username: "alice"}}}
	r := gin.New()
	r.GET("/top", getTop(noopLogger{}, getter))

	req := httptest.NewRequest(http.MethodGet, "/top", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if getter.gotTop != 100 {
		t.Fatalf("expected limit 100, got %d", getter.gotTop)
	}
}
