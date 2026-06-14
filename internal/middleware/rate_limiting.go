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

// Redis Lua Script for atomic token bucket
var tokenBucketScript = redis.NewScript(`
	local key = KEYS[1]
	local capacity = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	
	local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
	local tokens = tonumber(bucket[1]) or capacity
	local last_refill = tonumber(bucket[2]) or now

	-- Refill tokens
	local delta = math.floor((now - last_refill) * refill_rate)
	if delta > 0 then
		tokens = math.min(capacity, tokens + delta)
		last_refill = now
	end

	-- Check limits
	if tokens >= 1 then
		tokens = tokens - 1
		redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
		redis.call('EXPIRE', key, math.ceil(capacity / refill_rate))
		return 1 -- Allowed
	end
	return 0 -- Rate Limited
`)

const idempotencyProcessing = "processing"

// Atomically claim the key or return its current value in one round trip.
// Returns {1, ""} acquired, {0, value} duplicate, {2, ""} lost race.
var idempotencyAcquireScript = redis.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])
	local marker = ARGV[2]

	if redis.call('SET', key, marker, 'EX', ttl, 'NX') then
		return {1, ''}
	end

	local val = redis.call('GET', key)
	if val == false then
		return {2, ''}
	end
	return {0, val}
`)

// Replace "processing" with the cached response only if we still own the key.
var idempotencyFinalizeScript = redis.NewScript(`
	local key = KEYS[1]
	local marker = ARGV[1]
	local payload = ARGV[2]
	local ttl = tonumber(ARGV[3])

	if redis.call('GET', key) == marker then
		redis.call('SET', key, payload, 'EX', ttl)
		return 1
	end
	return 0
`)

// Delete the key on failure only if it is still marked "processing".
var idempotencyReleaseScript = redis.NewScript(`
	local key = KEYS[1]
	local marker = ARGV[1]

	if redis.call('GET', key) == marker then
		redis.call('DEL', key)
		return 1
	end
	return 0
`)

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

		parts, ok := result.([]interface{})
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
