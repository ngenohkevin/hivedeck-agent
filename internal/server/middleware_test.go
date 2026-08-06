package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	auth := NewAuthService("test-api-key", "test-secret")

	router := gin.New()
	router.Use(AuthMiddleware(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_ValidJWT(t *testing.T) {
	auth := NewAuthService("test-api-key", "test-secret")
	token, _ := auth.GenerateToken("admin", time.Hour)

	router := gin.New()
	router.Use(AuthMiddleware(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	auth := NewAuthService("test-api-key", "test-secret")

	router := gin.New()
	router.Use(AuthMiddleware(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	auth := NewAuthService("test-api-key", "test-secret")

	router := gin.New()
	router.Use(AuthMiddleware(auth))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5) // 5 requests per second

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow("test-client"))
	}

	// 6th request should be denied
	assert.False(t, limiter.Allow("test-client"))

	// Different client should be allowed
	assert.True(t, limiter.Allow("another-client"))
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2) // 2 requests per second

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(CORSMiddleware([]string{"*"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// With credentials enabled the wildcard has to be reflected back as the
	// concrete origin - browsers reject "*" alongside Allow-Credentials.
	assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestCORSMiddleware_NoOrigin(t *testing.T) {
	router := gin.New()
	router.Use(CORSMiddleware([]string{"*"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// A non-browser caller (curl, the dashboard's server side) sends no Origin.
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_SpecificOrigins(t *testing.T) {
	router := gin.New()
	router.Use(CORSMiddleware([]string{"http://allowed.com", "http://also-allowed.com"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://allowed.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, "http://allowed.com", w.Header().Get("Access-Control-Allow-Origin"))

	// Not allowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://not-allowed.com")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestLocalOnlyMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(LocalOnlyMiddleware())
	router.GET("/setup", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name       string
		remoteAddr string
		wantStatus int
	}{
		{"loopback", "127.0.0.1:5000", http.StatusOK},
		{"private LAN", "192.168.1.20:5000", http.StatusOK},
		{"tailscale", "100.85.91.122:5000", http.StatusOK},
		{"public", "203.0.113.7:5000", http.StatusForbidden},
		{"public near tailscale range", "100.128.0.1:5000", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/setup", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(RecoveryMiddleware())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
