package server

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authentication token",
			})
			return
		}

		// Try API key first
		if auth.ValidateAPIKey(token) {
			c.Set("auth_method", "api_key")
			c.Next()
			return
		}

		// Try JWT
		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authentication token",
			})
			return
		}

		c.Set("auth_method", "jwt")
		c.Set("claims", claims)
		c.Next()
	}
}

// RateLimiter implements a simple sliding-window rate limiter
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerSecond,
		window:   time.Second,
	}

	// Without this the map grows one permanent entry per client IP ever seen.
	go rl.sweep(time.Minute)

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Drop timestamps that fell out of the window, in place.
	recent := rl.requests[key][:0]
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.requests[key] = recent
		return false
	}

	rl.requests[key] = append(recent, now)
	return true
}

// sweep drops clients that have gone quiet, so the map tracks active callers
// rather than every IP the agent has ever answered.
func (rl *RateLimiter) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-rl.window)

		rl.mu.Lock()
		for key, times := range rl.requests {
			if len(times) == 0 || times[len(times)-1].Before(cutoff) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

// LoggerMiddleware creates logging middleware
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		// Determine if authenticated
		authMethod, _ := c.Get("auth_method")
		authenticated := authMethod != nil

		log.Printf("[%s] %s %s | Status: %d | Latency: %v | Client: %s | Auth: %v",
			method, path, c.Request.URL.RawQuery, status, latency, clientIP, authenticated)
	}
}

// RecoveryMiddleware handles panics
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			break
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// The response body varies with Origin, so it must not be cached under
		// one origin and replayed to another.
		c.Header("Vary", "Origin")

		switch {
		case allowAll:
			// "*" and Allow-Credentials:true are mutually exclusive per the
			// CORS spec - browsers reject the pair outright. Echo the caller's
			// origin so credentialed requests actually work.
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		default:
			for _, allowed := range allowedOrigins {
				if origin != "" && origin == allowed {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LocalOnlyMiddleware rejects requests that did not arrive over loopback, a
// private LAN or the Tailscale CGNAT range.
//
// It fronts the unauthenticated /setup endpoints: in setup mode anyone who can
// reach the port can claim the agent by POSTing an API key, so the blast radius
// has to stay off the public internet even if a firewall rule is missed.
func LocalOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := net.ParseIP(c.ClientIP())
		if ip == nil || !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || isTailscaleIP(ip)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "setup is only available from loopback, LAN or Tailscale",
			})
			return
		}
		c.Next()
	}
}

// tailscaleCGNAT is 100.64.0.0/10, the range Tailscale assigns to tailnet nodes.
var tailscaleCGNAT = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

func isTailscaleIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		return tailscaleCGNAT.Contains(v4)
	}
	// Tailscale's IPv6 ULA range.
	return len(ip) == net.IPv6len && ip[0] == 0xfd && ip[1] == 0x7a
}
