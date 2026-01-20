package server

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"`
}

// AuthService handles authentication
type AuthService struct {
	apiKey    string
	jwtSecret []byte
}

// NewAuthService creates a new auth service
func NewAuthService(apiKey, jwtSecret string) *AuthService {
	return &AuthService{
		apiKey:    apiKey,
		jwtSecret: []byte(jwtSecret),
	}
}

// ValidateAPIKey validates an API key
func (a *AuthService) ValidateAPIKey(key string) bool {
	return key != "" && key == a.apiKey
}

// GenerateToken generates a new JWT token
func (a *AuthService) GenerateToken(role string, duration time.Duration) (string, error) {
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "hivedeck-agent",
		},
		Role: role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateToken validates a JWT token
func (a *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractToken extracts the token from the Authorization header
func ExtractToken(c *gin.Context) string {
	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// Raw token
		return authHeader
	}

	// Check query parameter
	if token := c.Query("token"); token != "" {
		return token
	}

	return ""
}
