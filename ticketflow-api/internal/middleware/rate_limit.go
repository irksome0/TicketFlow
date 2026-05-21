package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	Name    string
	Limit   int
	Window  time.Duration
	KeyFunc func(*gin.Context) string
}

type rateBucket struct {
	count     int
	resetTime time.Time
}

func RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	limiter := &rateLimiter{
		name:    cfg.Name,
		limit:   cfg.Limit,
		window:  cfg.Window,
		keyFunc: cfg.KeyFunc,
		buckets: make(map[string]*rateBucket),
	}

	if limiter.name == "" {
		limiter.name = "default"
	}
	if limiter.limit <= 0 {
		limiter.limit = 60
	}
	if limiter.window <= 0 {
		limiter.window = time.Minute
	}
	if limiter.keyFunc == nil {
		limiter.keyFunc = ClientIPKey
	}

	return limiter.middleware
}

func ClientIPKey(c *gin.Context) string {
	return c.ClientIP()
}

func ClientIPRouteKey(c *gin.Context) string {
	return fmt.Sprintf("%s:%s:%s", c.ClientIP(), c.Request.Method, c.FullPath())
}

type rateLimiter struct {
	mu      sync.Mutex
	name    string
	limit   int
	window  time.Duration
	keyFunc func(*gin.Context) string
	buckets map[string]*rateBucket
}

func (r *rateLimiter) middleware(c *gin.Context) {
	now := time.Now()
	key := r.name + ":" + r.keyFunc(c)

	r.mu.Lock()
	r.cleanupExpired(now)

	bucket, exists := r.buckets[key]
	if !exists || now.After(bucket.resetTime) {
		bucket = &rateBucket{
			count:     0,
			resetTime: now.Add(r.window),
		}
		r.buckets[key] = bucket
	}

	bucket.count++
	remaining := r.limit - bucket.count
	retryAfter := time.Until(bucket.resetTime)
	allowed := bucket.count <= r.limit
	r.mu.Unlock()

	if retryAfter < 0 {
		retryAfter = 0
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(r.limit))
	if remaining < 0 {
		remaining = 0
	}
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(bucket.resetTime.Unix(), 10))

	if !allowed {
		c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "перевищено ліміт запитів, спробуйте пізніше",
		})
		return
	}

	c.Next()
}

func (r *rateLimiter) cleanupExpired(now time.Time) {
	for key, bucket := range r.buckets {
		if now.After(bucket.resetTime.Add(r.window)) {
			delete(r.buckets, key)
		}
	}
}
