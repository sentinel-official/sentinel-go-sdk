package middlewares

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
)

// RateLimiterOptions defines configuration for the rate limiter middleware.
type RateLimiterOptions struct {
	Limit           float64       // Allowed requests per second
	Burst           int           // Maximum tokens that can accumulate
	InactiveTimeout time.Duration // Remove clients inactive for this duration
	CleanupInterval time.Duration // How often to run cleanup
}

// RateLimiter returns a Gin middleware that limits requests per IP.
// limit = requests per second; burst = max tokens in bucket.
func RateLimiter(opts *RateLimiterOptions) gin.HandlerFunc {
	// Apply defaults if values are zero
	if opts == nil {
		opts = &RateLimiterOptions{}
	}
	if opts.Limit <= 0 {
		opts.Limit = 1
	}
	if opts.Burst <= 0 {
		opts.Burst = 5
	}
	if opts.InactiveTimeout <= 0 {
		opts.InactiveTimeout = 15 * time.Minute
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = time.Minute
	}

	type Client struct {
		limiter   *rate.Limiter // Token bucket for this client
		timestamp time.Time     // Last request time
	}

	m := safe.NewMap[string, *Client]() // Thread-safe client storage

	// Background cleanup: remove clients inactive for >15m
	go func() {
		ticker := time.NewTicker(opts.CleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			m.RangeDelete(func(_ string, v *Client) (bool, bool) {
				return time.Since(v.timestamp) > opts.InactiveTimeout, false
			})
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		timestamp := time.Now()

		// Atomically create or update the client's limiter
		v := m.Update(ip, func(v *Client, found bool) *Client {
			if !found {
				v = &Client{
					limiter: rate.NewLimiter(rate.Limit(opts.Limit), opts.Burst),
				}
			}

			v.timestamp = timestamp
			return v
		})

		// Try to reserve 1 token at current time
		r := v.limiter.ReserveN(timestamp, 1)
		if !r.OK() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		// Calculate delay, remaining tokens, and reset time
		delay := r.DelayFrom(timestamp)
		remaining := math.Floor(v.limiter.TokensAt(timestamp))
		if remaining < 0 {
			remaining = 0
		}

		reset := math.Ceil((float64(opts.Burst) - remaining) / opts.Limit)
		after := math.Ceil(delay.Seconds())

		// Send RFC 9239 RateLimit headers
		c.Header("RateLimit-Limit", strconv.Itoa(opts.Burst))
		c.Header("RateLimit-Remaining", strconv.Itoa(int(remaining)))
		c.Header("RateLimit-Reset", strconv.Itoa(int(reset)))

		// If a delay is required, reject with retry info
		if delay > 0 {
			c.Header("RateLimit-After", strconv.Itoa(int(after)))
			c.AbortWithStatus(http.StatusTooManyRequests)

			// Free the reserved token
			r.CancelAt(timestamp)

			return
		}

		c.Next()
	}
}
