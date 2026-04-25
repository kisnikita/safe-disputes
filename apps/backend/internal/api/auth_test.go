package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeUserCreator struct {
	err         error
	called      bool
	gotUsername string
}

func (f *fakeUserCreator) CreateIfNotExist(_ context.Context, username string, _ *string) error {
	f.called = true
	f.gotUsername = username
	return f.err
}

func TestTelegramAuth(t *testing.T) {
	t.Run("returns unauthorized when username missing", func(t *testing.T) {
		r := gin.New()
		r.GET("/auth", telegramAuth(noopLogger{}, &fakeUserCreator{}))

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})

	t.Run("returns bad request when username has invalid type", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", 42)
			c.Next()
		})
		r.GET("/auth", telegramAuth(noopLogger{}, &fakeUserCreator{}))

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("returns internal error when service fails", func(t *testing.T) {
		svc := &fakeUserCreator{err: errors.New("boom")}

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/auth", telegramAuth(noopLogger{}, svc))

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		if !svc.called {
			t.Fatal("expected CreateIfNotExist to be called")
		}
	})

	t.Run("returns ok and calls service", func(t *testing.T) {
		svc := &fakeUserCreator{}

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("username", "alice")
			c.Next()
		})
		r.GET("/auth", telegramAuth(noopLogger{}, svc))

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		if svc.gotUsername != "alice" {
			t.Fatalf("expected username alice, got %q", svc.gotUsername)
		}
	})
}
