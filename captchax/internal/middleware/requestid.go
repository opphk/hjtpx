package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDKey       = "X-Request-ID"
	RequestIDCtxKey    = "request_id"
	RequestIDLength    = 16
	TraceIDLength      = 32
	SpanIDLength       = 8
)

type RequestIDConfig struct {
	HeaderName     string
	ContextKey    string
	Generator     func() string
	MaxLength     int
	ValidateFunc  func(string) bool
}

var defaultRequestIDConfig = &RequestIDConfig{
	HeaderName:  RequestIDKey,
	ContextKey:  RequestIDCtxKey,
	Generator:   GenerateRequestID,
	MaxLength:   64,
	ValidateFunc: ValidRequestID,
}

func RequestID() gin.HandlerFunc {
	return RequestIDWithConfig(defaultRequestIDConfig)
}

func RequestIDWithConfig(config *RequestIDConfig) gin.HandlerFunc {
	cfg := *config
	if cfg.HeaderName == "" {
		cfg.HeaderName = defaultRequestIDConfig.HeaderName
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = defaultRequestIDConfig.ContextKey
	}
	if cfg.Generator == nil {
		cfg.Generator = defaultRequestIDConfig.Generator
	}
	if cfg.MaxLength == 0 {
		cfg.MaxLength = defaultRequestIDConfig.MaxLength
	}
	if cfg.ValidateFunc == nil {
		cfg.ValidateFunc = defaultRequestIDConfig.ValidateFunc
	}

	return func(c *gin.Context) {
		requestID := c.GetHeader(cfg.HeaderName)

		if requestID == "" {
			requestID = cfg.Generator()
		}

		if !cfg.ValidateFunc(requestID) {
			requestID = cfg.Generator()
		}

		if len(requestID) > cfg.MaxLength {
			requestID = requestID[:cfg.MaxLength]
		}

		c.Set(cfg.ContextKey, requestID)
		c.Header(cfg.HeaderName, requestID)

		ctx := context.WithValue(c.Request.Context(), RequestIDCtxKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func GenerateRequestID() string {
	bytes := make([]byte, RequestIDLength)
	if _, err := rand.Read(bytes); err != nil {
		timestamp := time.Now().UnixNano()
		return fmt.Sprintf("%x-%d", uuid.New().String(), timestamp)
	}
	return hex.EncodeToString(bytes)
}

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateTraceID() string {
	bytes := make([]byte, TraceIDLength)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(bytes)
}

func GenerateSpanID() string {
	bytes := make([]byte, SpanIDLength)
	if _, err := rand.Read(bytes); err != nil {
		timestamp := time.Now().UnixNano()
		return fmt.Sprintf("%x", timestamp)
	}
	return hex.EncodeToString(bytes)
}

func ValidRequestID(requestID string) bool {
	if len(requestID) < 4 || len(requestID) > 64 {
		return false
	}

	for _, c := range requestID {
		if !isValidRequestIDChar(c) {
			return false
		}
	}

	return true
}

func isValidRequestIDChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDCtxKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	if requestID := c.GetHeader(RequestIDKey); requestID != "" {
		return requestID
	}
	return ""
}

func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDCtxKey).(string); ok {
		return requestID
	}
	return ""
}

type TraceInfo struct {
	TraceID string
	SpanID  string
	ParentID string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Spans     []*Span
	Metadata  map[string]interface{}
	mu        sync.RWMutex
}

type Span struct {
	SpanID   string
	ParentID string
	Name     string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Metadata  map[string]interface{}
}

var traceInfoPool = sync.Pool{
	New: func() interface{} {
		return &TraceInfo{
			Spans:    make([]*Span, 0, 10),
			Metadata: make(map[string]interface{}),
		}
	},
}

func NewTraceInfo() *TraceInfo {
	ti := traceInfoPool.Get().(*TraceInfo)
	ti.TraceID = GenerateTraceID()
	ti.SpanID = GenerateSpanID()
	ti.StartTime = time.Now()
	ti.EndTime = time.Time{}
	ti.Duration = 0
	ti.Spans = ti.Spans[:0]
	ti.Metadata = make(map[string]interface{})
	return ti
}

func (ti *TraceInfo) Release() {
	ti.Spans = ti.Spans[:0]
	ti.TraceID = ""
	ti.SpanID = ""
	ti.ParentID = ""
	ti.StartTime = time.Time{}
	ti.EndTime = time.Time{}
	ti.Duration = 0
	traceInfoPool.Put(ti)
}

func (ti *TraceInfo) StartSpan(name string) *Span {
	span := &Span{
		SpanID:   GenerateSpanID(),
		ParentID: ti.SpanID,
		Name:     name,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	return span
}

func (ti *TraceInfo) EndSpan(span *Span) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	ti.mu.Lock()
	ti.Spans = append(ti.Spans, span)
	ti.mu.Unlock()
}

func (ti *TraceInfo) End() {
	ti.EndTime = time.Now()
	ti.Duration = ti.EndTime.Sub(ti.StartTime)
}

func (ti *TraceInfo) ToMap() map[string]interface{} {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	spans := make([]map[string]interface{}, len(ti.Spans))
	for i, span := range ti.Spans {
		spans[i] = map[string]interface{}{
			"span_id":   span.SpanID,
			"parent_id": span.ParentID,
			"name":      span.Name,
			"duration":  span.Duration.String(),
			"start":     span.StartTime.Format(time.RFC3339Nano),
			"end":       span.EndTime.Format(time.RFC3339Nano),
		}
	}

	return map[string]interface{}{
		"trace_id": ti.TraceID,
		"span_id":  ti.SpanID,
		"duration": ti.Duration.String(),
		"start":    ti.StartTime.Format(time.RFC3339Nano),
		"end":      ti.EndTime.Format(time.RFC3339Nano),
		"spans":    spans,
	}
}

type TracingContext struct {
	traces map[string]*TraceInfo
	mu     sync.RWMutex
}

var defaultTracingContext = &TracingContext{
	traces: make(map[string]*TraceInfo),
}

func GetTracingContext() *TracingContext {
	return defaultTracingContext
}

func (tc *TracingContext) StartTrace(requestID string) *TraceInfo {
	ti := NewTraceInfo()
	tc.mu.Lock()
	tc.traces[requestID] = ti
	tc.mu.Unlock()
	return ti
}

func (tc *TracingContext) GetTrace(requestID string) (*TraceInfo, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	ti, ok := tc.traces[requestID]
	return ti, ok
}

func (tc *TracingContext) EndTrace(requestID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if ti, ok := tc.traces[requestID]; ok {
		ti.End()
		ti.Release()
		delete(tc.traces, requestID)
	}
}

func (tc *TracingContext) Cleanup(timeout time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	now := time.Now()
	for requestID, ti := range tc.traces {
		if now.Sub(ti.StartTime) > timeout {
			ti.Release()
			delete(tc.traces, requestID)
		}
	}
}

type requestLog struct {
	RequestID    string        `json:"request_id"`
	TraceID      string        `json:"trace_id"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	StatusCode   int           `json:"status_code"`
	Duration     time.Duration `json:"duration"`
	ClientIP     string        `json:"client_ip"`
	UserAgent    string        `json:"user_agent"`
	ErrorMessage string        `json:"error,omitempty"`
	StackTrace   string        `json:"stack_trace,omitempty"`
}

func (rl *requestLog) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"request_id":  rl.RequestID,
		"trace_id":    rl.TraceID,
		"method":      rl.Method,
		"path":        rl.Path,
		"status_code": rl.StatusCode,
		"duration":   rl.Duration.String(),
		"client_ip":   rl.ClientIP,
		"user_agent":  rl.UserAgent,
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := GetRequestID(c)

		c.Next()

		duration := time.Since(start)

		log := &requestLog{
			RequestID:  requestID,
			TraceID:    GetRequestIDFromContext(c.Request.Context()),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Duration:  duration,
			ClientIP:  c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
		}

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.ErrorMessage = err.Error()
			if c.Err() != nil {
				log.StackTrace = string(debug.Stack())
			}
		}

		_ = log
	}
}

func RecoveryWithRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(c)
				stack := debug.Stack()

				c.AbortWithStatusJSON(500, gin.H{
					"error":      "internal server error",
					"request_id": requestID,
					"message":    fmt.Sprintf("panic recovered: %v", err),
				})

				_ = fmt.Sprintf("request_id=%s error=%v stack=%s", requestID, err, string(stack))
			}
		}()
		c.Next()
	}
}

func RequestIDValidator() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDKey)

		if requestID != "" && !ValidRequestID(requestID) {
			c.AbortWithStatusJSON(400, gin.H{
				"error":      "invalid request id",
				"request_id": requestID,
			})
			return
		}

		c.Next()
	}
}

type RequestIDMiddleware struct {
	config    *RequestIDConfig
	tracing   *TracingContext
	cleanupInterval time.Duration
	cleanupTimeout  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{
		config:          defaultRequestIDConfig,
		tracing:         defaultTracingContext,
		cleanupInterval: 5 * time.Minute,
		cleanupTimeout:  30 * time.Minute,
		stopChan:        make(chan struct{}),
	}
}

func (m *RequestIDMiddleware) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.tracing.Cleanup(m.cleanupTimeout)
			case <-m.stopChan:
				return
			}
		}
	}()
}

func (m *RequestIDMiddleware) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

func (m *RequestIDMiddleware) Handler() gin.HandlerFunc {
	return RequestIDWithConfig(m.config)
}

func (m *RequestIDMiddleware) TracingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GetRequestID(c)
		trace, _ := m.tracing.GetTrace(requestID)

		if trace == nil {
			trace = m.tracing.StartTrace(requestID)
		}

		c.Set("trace", trace)

		c.Next()

		m.tracing.EndTrace(requestID)
	}
}
