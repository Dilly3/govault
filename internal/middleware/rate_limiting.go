package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const idempotencyTTL = 24 * time.Hour

type cachedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type responseCapture struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

const idempotencyProcessing = "processing"

func replayCachedResponse(c *gin.Context, val string) bool {
	var cached cachedResponse
	if json.Unmarshal([]byte(val), &cached) != nil {
		return false
	}
	for k, v := range cached.Headers {
		c.Header(k, v)
	}
	contentType := cached.Headers["Content-Type"]
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(cached.Status, contentType, []byte(cached.Body))
	c.Abort()
	return true
}

func handleDuplicateRequest(c *gin.Context, val string) {
	if val == idempotencyProcessing {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "transaction in progress"})
		return
	}
	if replayCachedResponse(c, val) {
		return
	}
	c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "transaction already processed"})
}

func RateLimiter(rdb *redis.Client, capacity int, refillRate float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rate_limit:" + ip
		now := time.Now().Unix()

		allowed, err := tokenBucketScript.Run(c.Request.Context(), rdb, []string{key}, capacity, refillRate, now).Result()

		if err != nil || allowed.(int64) == 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
			return
		}
		ctx := c.Request.Context()
		redisKey := "idemp:" + key
		ttlSeconds := int(idempotencyTTL.Seconds())

		result, err := idempotencyAcquireScript.Run(ctx, rdb, []string{redisKey}, ttlSeconds, idempotencyProcessing).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "idempotency store unavailable"})
			return
		}

		parts, ok := result.([]any)
		if !ok || len(parts) != 2 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "idempotency store unavailable"})
			return
		}

		state, _ := parts[0].(int64)
		switch state {
		case 0:
			val, _ := parts[1].(string)
			handleDuplicateRequest(c, val)
			return
		case 2:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "transaction in progress"})
			return
		}

		capture := &responseCapture{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = capture
		c.Next()

		if len(c.Errors) > 0 || capture.status >= http.StatusInternalServerError {
			_, _ = idempotencyReleaseScript.Run(ctx, rdb, []string{redisKey}, idempotencyProcessing).Result()
			return
		}

		contentType := capture.Header().Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		payload, err := json.Marshal(cachedResponse{
			Status:  capture.status,
			Headers: map[string]string{"Content-Type": contentType},
			Body:    capture.body.String(),
		})
		if err != nil {
			_, _ = idempotencyReleaseScript.Run(ctx, rdb, []string{redisKey}, idempotencyProcessing).Result()
			return
		}
		_, _ = idempotencyFinalizeScript.Run(ctx, rdb, []string{redisKey}, idempotencyProcessing, payload, ttlSeconds).Result()
	}
}
