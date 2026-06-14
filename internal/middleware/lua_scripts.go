package middleware

import "github.com/go-redis/redis/v8"

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
