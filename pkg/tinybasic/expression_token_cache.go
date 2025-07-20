package tinybasic

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// ExpressionTokenCache provides fast caching for tokenized expressions
type ExpressionTokenCache struct {
	// Primary cache storage
	cache map[uint64]*CachedTokens
	mutex sync.RWMutex
	
	// Cache configuration
	maxSize     int
	maxAge      time.Duration
	
	// Performance metrics
	hits        int64
	misses      int64
	evictions   int64
	
	// Cache cleanup
	lastCleanup int64 // Unix timestamp
	cleanupInterval time.Duration
}

// CachedTokens represents cached tokenization result
type CachedTokens struct {
	tokens    []token
	hash      uint64
	createdAt int64 // Unix timestamp for atomic operations
	hitCount  int64
	lastUsed  int64
}

// NewExpressionTokenCache creates a new expression token cache
func NewExpressionTokenCache(maxSize int, maxAge time.Duration) *ExpressionTokenCache {
	return &ExpressionTokenCache{
		cache:           make(map[uint64]*CachedTokens),
		maxSize:         maxSize,
		maxAge:          maxAge,
		cleanupInterval: maxAge / 4, // Cleanup every quarter of max age
		lastCleanup:     time.Now().Unix(),
	}
}

// generateExpressionHash creates a fast hash for the expression string
func generateExpressionHash(expr string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(expr))
	return h.Sum64()
}

// Get attempts to retrieve cached tokens for an expression
func (c *ExpressionTokenCache) Get(expr string) ([]token, bool) {
	hash := generateExpressionHash(expr)
	
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	cached, exists := c.cache[hash]
	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}
	
	// Check if cache entry is too old
	now := time.Now().Unix()
	if now-cached.createdAt > int64(c.maxAge.Seconds()) {
		// Don't delete here (we have read lock), just return miss
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}
	
	// Update access statistics
	atomic.AddInt64(&c.hits, 1)
	atomic.AddInt64(&cached.hitCount, 1)
	atomic.StoreInt64(&cached.lastUsed, now)
	
	// Return a copy of tokens to prevent mutation
	tokensCopy := make([]token, len(cached.tokens))
	copy(tokensCopy, cached.tokens)
	
	return tokensCopy, true
}

// Put stores tokenized expression in the cache
func (c *ExpressionTokenCache) Put(expr string, tokens []token) {
	if len(tokens) == 0 {
		return // Don't cache empty token lists
	}
	
	hash := generateExpressionHash(expr)
	now := time.Now().Unix()
	
	// Create a copy of tokens for storage
	tokensCopy := make([]token, len(tokens))
	copy(tokensCopy, tokens)
	
	cached := &CachedTokens{
		tokens:    tokensCopy,
		hash:      hash,
		createdAt: now,
		lastUsed:  now,
		hitCount:  0,
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Check if we need to evict entries
	if len(c.cache) >= c.maxSize {
		c.evictLeastRecentlyUsed()
	}
	
	c.cache[hash] = cached
	
	// Periodic cleanup of expired entries
	if now-c.lastCleanup > int64(c.cleanupInterval.Seconds()) {
		go c.cleanupExpired() // Run cleanup in background
		c.lastCleanup = now
	}
}

// evictLeastRecentlyUsed removes the least recently used cache entry
func (c *ExpressionTokenCache) evictLeastRecentlyUsed() {
	if len(c.cache) == 0 {
		return
	}
	
	var oldestHash uint64
	var oldestTime int64 = time.Now().Unix()
	
	// Find the least recently used entry
	for hash, cached := range c.cache {
		lastUsed := atomic.LoadInt64(&cached.lastUsed)
		if lastUsed < oldestTime {
			oldestTime = lastUsed
			oldestHash = hash
		}
	}
	
	// Remove the oldest entry
	if oldestHash != 0 {
		delete(c.cache, oldestHash)
		atomic.AddInt64(&c.evictions, 1)
	}
}

// cleanupExpired removes expired cache entries (runs in background)
func (c *ExpressionTokenCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now().Unix()
	maxAge := int64(c.maxAge.Seconds())
	
	for hash, cached := range c.cache {
		if now-cached.createdAt > maxAge {
			delete(c.cache, hash)
			atomic.AddInt64(&c.evictions, 1)
		}
	}
}

// GetStats returns cache performance statistics
func (c *ExpressionTokenCache) GetStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	evictions := atomic.LoadInt64(&c.evictions)
	
	var hitRatio float64
	if hits+misses > 0 {
		hitRatio = float64(hits) / float64(hits+misses)
	}
	
	return CacheStats{
		Hits:      hits,
		Misses:    misses,
		Evictions: evictions,
		HitRatio:  hitRatio,
		Size:      len(c.cache),
		MaxSize:   c.maxSize,
	}
}

// CacheStats represents cache performance metrics
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	HitRatio  float64
	Size      int
	MaxSize   int
}

// Clear removes all entries from the cache
func (c *ExpressionTokenCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[uint64]*CachedTokens)
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
	atomic.StoreInt64(&c.evictions, 0)
}

// SetMaxSize updates the maximum cache size
func (c *ExpressionTokenCache) SetMaxSize(maxSize int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.maxSize = maxSize
	
	// Evict entries if current size exceeds new max size
	for len(c.cache) > maxSize {
		c.evictLeastRecentlyUsed()
	}
}

// IsExpired checks if a cache entry is expired
func (c *ExpressionTokenCache) IsExpired(cached *CachedTokens) bool {
	now := time.Now().Unix()
	return now-cached.createdAt > int64(c.maxAge.Seconds())
}

// GetTopExpressions returns the most frequently used cached expressions
func (c *ExpressionTokenCache) GetTopExpressions(limit int) []CachedExpressionInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	type expressionWithCount struct {
		hash     uint64
		hitCount int64
		tokens   []token
	}
	
	var expressions []expressionWithCount
	for hash, cached := range c.cache {
		expressions = append(expressions, expressionWithCount{
			hash:     hash,
			hitCount: atomic.LoadInt64(&cached.hitCount),
			tokens:   cached.tokens,
		})
	}
	
	// Sort by hit count (bubble sort for simplicity)
	for i := 0; i < len(expressions); i++ {
		for j := i + 1; j < len(expressions); j++ {
			if expressions[i].hitCount < expressions[j].hitCount {
				expressions[i], expressions[j] = expressions[j], expressions[i]
			}
		}
	}
	
	// Return top expressions
	result := make([]CachedExpressionInfo, 0, limit)
	for i := 0; i < len(expressions) && i < limit; i++ {
		result = append(result, CachedExpressionInfo{
			Hash:      expressions[i].hash,
			HitCount:  expressions[i].hitCount,
			TokenCount: len(expressions[i].tokens),
		})
	}
	
	return result
}

// CachedExpressionInfo represents information about a cached expression
type CachedExpressionInfo struct {
	Hash       uint64
	HitCount   int64
	TokenCount int
}