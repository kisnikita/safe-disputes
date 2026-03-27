package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddleware(t *testing.T) {
	t.Setenv("TELEGRAM_SECRET_TOKEN", "test-token")

	newRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/ping", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		return r
	}

	t.Run("returns bad request when authorization header missing", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("returns bad request for invalid header format", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("Authorization", "Bearer token")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("returns unauthorized for invalid initData", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("Authorization", "tma invalid-init-data")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}
