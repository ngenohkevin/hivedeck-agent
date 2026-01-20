package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_ValidateAPIKey(t *testing.T) {
	auth := NewAuthService("my-api-key", "my-secret")

	assert.True(t, auth.ValidateAPIKey("my-api-key"))
	assert.False(t, auth.ValidateAPIKey("wrong-key"))
	assert.False(t, auth.ValidateAPIKey(""))
}

func TestAuthService_GenerateAndValidateToken(t *testing.T) {
	auth := NewAuthService("api-key", "jwt-secret")

	token, err := auth.GenerateToken("admin", time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "hivedeck-agent", claims.Issuer)
}

func TestAuthService_ExpiredToken(t *testing.T) {
	auth := NewAuthService("api-key", "jwt-secret")

	// Generate token that expires immediately
	token, err := auth.GenerateToken("admin", -time.Hour)
	require.NoError(t, err)

	_, err = auth.ValidateToken(token)
	assert.Error(t, err)
}

func TestAuthService_InvalidToken(t *testing.T) {
	auth := NewAuthService("api-key", "jwt-secret")

	_, err := auth.ValidateToken("invalid.token.here")
	assert.Error(t, err)

	_, err = auth.ValidateToken("")
	assert.Error(t, err)
}

func TestAuthService_WrongSecret(t *testing.T) {
	auth1 := NewAuthService("api-key", "secret1")
	auth2 := NewAuthService("api-key", "secret2")

	token, err := auth1.GenerateToken("admin", time.Hour)
	require.NoError(t, err)

	_, err = auth2.ValidateToken(token)
	assert.Error(t, err)
}

func TestExtractToken_BearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer my-token")

	token := ExtractToken(c)
	assert.Equal(t, "my-token", token)
}

func TestExtractToken_RawHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "raw-token")

	token := ExtractToken(c)
	assert.Equal(t, "raw-token", token)
}

func TestExtractToken_QueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?token=query-token", nil)

	token := ExtractToken(c)
	assert.Equal(t, "query-token", token)
}

func TestExtractToken_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)

	token := ExtractToken(c)
	assert.Empty(t, token)
}

func TestExtractToken_HeaderPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?token=query-token", nil)
	c.Request.Header.Set("Authorization", "Bearer header-token")

	// Header should take priority
	token := ExtractToken(c)
	assert.Equal(t, "header-token", token)
}
