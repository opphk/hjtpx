package middleware

import (
	"captchax/pkg/response"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DeduplicationConfig struct {
	Enabled           bool
	WindowDuration    time.Duration
	MaxRequestsPerKey int
	StorageType       string
}

var defaultDeduplicationConfig = &DeduplicationConfig{
	Enabled:           true,
	WindowDuration:    60 * time.Second,
	MaxRequestsPerKey: 1,
	StorageType:      "memory",
}

type deduplicationStore struct {
	requests map[string]*dedupEntry
	mu       sync.RWMutex
}

type dedupEntry struct {
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

var memoryStore = &deduplicationStore{
	requests: make(map[string]*dedupEntry),
}

func Deduplication() gin.HandlerFunc {
	return DeduplicationWithConfig(defaultDeduplicationConfig)
}

func DeduplicationWithConfig(config *DeduplicationConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultDeduplicationConfig
	}

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		dedupKey := generateDedupKey(c, config)
		dedupID := c.GetHeader("X-Deduplication-ID")

		if dedupID == "" {
			dedupID = c.GetHeader("X-Request-ID")
		}

		if dedupID == "" {
			dedupID = generateDedupID()
		}
		c.Set("dedup_id", dedupID)
		c.Header("X-Deduplication-ID", dedupID)

		if isDuplicate(dedupKey, config) {
			c.Header("X-Duplicate-Request", "true")
			c.Header("Retry-After", fmt.Sprintf("%d", int(config.WindowDuration.Seconds())))

			response.ErrorWithStatus(c, http.StatusConflict, 409,
				fmt.Sprintf("duplicate request detected, retry after %d seconds", int(config.WindowDuration.Seconds())))
			c.Abort()
			return
		}

		c.Next()
	}
}

func generateDedupKey(c *gin.Context, config *DeduplicationConfig) string {
	var components []string

	components = append(components, c.ClientIP())

	if appID := c.GetHeader("X-App-ID"); appID != "" {
		components = append(components, appID)
	}

	method := c.Request.Method
	path := c.Request.URL.Path
	components = append(components, method, path)

	if body := extractBodyHash(c); body != "" {
		components = append(components, body)
	}

	dedupID := c.GetHeader("X-Deduplication-ID")
	if dedupID != "" {
		return fmt.Sprintf("dedup:%s", dedupID)
	}

	rawKey := ""
	for _, comp := range components {
		rawKey += comp + ":"
	}

	hash := sha256.Sum256([]byte(rawKey))
	return fmt.Sprintf("dedup:%s", hex.EncodeToString(hash[:]))
}

func extractBodyHash(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	bodyBytes, err := c.GetRawData()
	if err != nil {
		return ""
	}
	c.Request.Body = newBodyCloser(bodyBytes)

	if len(bodyBytes) == 0 {
		return ""
	}

	hash := sha256.Sum256(bodyBytes)
	return hex.EncodeToString(hash[:])
}

type bodyCloser struct {
	data []byte
	pos  int
}

func newBodyCloser(data []byte) *bodyCloser {
	return &bodyCloser{data: data, pos: 0}
}

func (b *bodyCloser) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bodyCloser) Close() error {
	return nil
}

func isDuplicate(key string, config *DeduplicationConfig) bool {
	now := time.Now()
	windowStart := now.Add(-config.WindowDuration)

	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	entry, exists := memoryStore.requests[key]
	if !exists {
		memoryStore.requests[key] = &dedupEntry{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
		go cleanupOldEntries(windowStart)
		return false
	}

	if now.Sub(entry.FirstSeen) > config.WindowDuration {
		memoryStore.requests[key] = &dedupEntry{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
		return false
	}

	if entry.Count >= config.MaxRequestsPerKey {
		return true
	}

	entry.Count++
	entry.LastSeen = now
	return false
}

func cleanupOldEntries(windowStart time.Time) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	for key, entry := range memoryStore.requests {
		if entry.LastSeen.Before(windowStart) {
			delete(memoryStore.requests, key)
		}
	}
}

func generateDedupID() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%x", timestamp)
}

type DeduplicationMiddleware struct {
	config     *DeduplicationConfig
	store      *deduplicationStore
	stopChan   chan struct{}
	cleanupInt time.Duration
}

func NewDeduplicationMiddleware(config *DeduplicationConfig) *DeduplicationMiddleware {
	if config == nil {
		config = defaultDeduplicationConfig
	}

	return &DeduplicationMiddleware{
		config:     config,
		store:      memoryStore,
		stopChan:   make(chan struct{}),
		cleanupInt: config.WindowDuration,
	}
}

func (m *DeduplicationMiddleware) Start() {
	go m.periodicCleanup()
}

func (m *DeduplicationMiddleware) Stop() {
	close(m.stopChan)
}

func (m *DeduplicationMiddleware) periodicCleanup() {
	ticker := time.NewTicker(m.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			windowStart := time.Now().Add(-m.config.WindowDuration)
			cleanupOldEntries(windowStart)
		case <-m.stopChan:
			return
		}
	}
}

func (m *DeduplicationMiddleware) Handler() gin.HandlerFunc {
	return DeduplicationWithConfig(m.config)
}

func (m *DeduplicationMiddleware) Stats() map[string]interface{} {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	totalEntries := len(m.store.requests)
	oldestEntry := time.Time{}
	newestEntry := time.Time{}

	for _, entry := range m.store.requests {
		if oldestEntry.IsZero() || entry.FirstSeen.Before(oldestEntry) {
			oldestEntry = entry.FirstSeen
		}
		if newestEntry.IsZero() || entry.LastSeen.After(newestEntry) {
			newestEntry = entry.LastSeen
		}
	}

	return map[string]interface{}{
		"total_entries": totalEntries,
		"oldest_entry":  oldestEntry,
		"newest_entry":  newestEntry,
		"window":        m.config.WindowDuration.String(),
	}
}
