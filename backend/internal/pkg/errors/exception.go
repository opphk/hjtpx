package errors

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ExceptionHandler interface {
	HandleException(exception *Exception) error
}

type Exception struct {
	Type         string                 `json:"type"`
	Message      string                 `json:"message"`
	StackTrace   string                 `json:"stack_trace"`
	Timestamp    time.Time              `json:"timestamp"`
	ErrorID      string                 `json:"error_id"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Recovered    bool                   `json:"recovered"`
	RetryCount   int                    `json:"retry_count"`
	Severity     ErrorSeverity          `json:"severity"`
}

func NewException(err error, context map[string]interface{}) *Exception {
	exception := &Exception{
		Type:       fmt.Sprintf("%T", err),
		Message:    err.Error(),
		Timestamp: time.Now(),
		ErrorID:   GenerateErrorID(),
		Context:   context,
		Severity:  SeverityError,
	}

	if appErr, ok := err.(*AppError); ok {
		exception.Type = "AppError"
		exception.Message = appErr.Message
		exception.Severity = appErr.Severity
	}

	exception.StackTrace = captureStackTrace(3)

	return exception
}

func captureStackTrace(skip int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	var buf bytes.Buffer

	for i := 0; i < n; i++ {
		pc := pcs[i]
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			fmt.Fprintf(&buf, "\n  at %s (%s:%d)", fn.Name(), file, line)
		}
	}

	return buf.String()
}

type ExceptionHandlerChain struct {
	handlers []ExceptionHandler
	mu       sync.RWMutex
}

var globalExceptionHandlerChain *ExceptionHandlerChain

func init() {
	globalExceptionHandlerChain = &ExceptionHandlerChain{
		handlers: make([]ExceptionHandler, 0),
	}
}

func GetExceptionHandlerChain() *ExceptionHandlerChain {
	return globalExceptionHandlerChain
}

func (c *ExceptionHandlerChain) AddHandler(handler ExceptionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

func (c *ExceptionHandlerChain) Handle(exception *Exception) {
	c.mu.RLock()
	handlers := make([]ExceptionHandler, len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.HandleException(exception); err != nil {
			fmt.Printf("Exception handler error: %v\n", err)
		}
	}
}

func (c *ExceptionHandlerChain) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = make([]ExceptionHandler, 0)
}

type LoggingExceptionHandler struct{}

func NewLoggingExceptionHandler() *LoggingExceptionHandler {
	return &LoggingExceptionHandler{}
}

func (h *LoggingExceptionHandler) HandleException(exception *Exception) error {
	stats := GetErrorStatisticsCollector()
	appErr := New(CodeInternalError, exception.Message)
	stats.RecordError(appErr.Code)

	fmt.Printf("[EXCEPTION] %s | %s | %s | %s\n",
		exception.ErrorID,
		exception.Severity.String(),
		exception.Type,
		exception.Message,
	)

	if len(exception.Context) > 0 {
		fmt.Printf("  Context: %v\n", exception.Context)
	}

	if exception.StackTrace != "" {
		fmt.Printf("  Stack: %s\n", exception.StackTrace)
	}

	return nil
}

type MetricsExceptionHandler struct {
	exceptionCounts map[string]*int64
	mu              sync.RWMutex
}

func NewMetricsExceptionHandler() *MetricsExceptionHandler {
	return &MetricsExceptionHandler{
		exceptionCounts: make(map[string]*int64),
	}
}

func (h *MetricsExceptionHandler) HandleException(exception *Exception) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.exceptionCounts[exception.Type]; !exists {
		h.exceptionCounts[exception.Type] = new(int64)
	}
	atomic.AddInt64(h.exceptionCounts[exception.Type], 1)

	return nil
}

func (h *MetricsExceptionHandler) GetCount(exceptionType string) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if count, exists := h.exceptionCounts[exceptionType]; exists {
		return atomic.LoadInt64(count)
	}
	return 0
}

func (h *MetricsExceptionHandler) GetAllCounts() map[string]int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make(map[string]int64)
	for t, count := range h.exceptionCounts {
		results[t] = atomic.LoadInt64(count)
	}
	return results
}

type RecoveryHandler struct {
	handlers      []func(interface{}) interface{}
	mu            sync.RWMutex
	maxRetries    int
	backoffBaseMs int64
}

func NewRecoveryHandler() *RecoveryHandler {
	return &RecoveryHandler{
		handlers:      make([]func(interface{}) interface{}, 0),
		maxRetries:    3,
		backoffBaseMs: 100,
	}
}

func (h *RecoveryHandler) AddRecoveryHandler(handler func(interface{}) interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers = append(h.handlers, handler)
}

func (h *RecoveryHandler) SetMaxRetries(maxRetries int) {
	h.maxRetries = maxRetries
}

func (h *RecoveryHandler) SetBackoff(baseMs int64) {
	h.backoffBaseMs = baseMs
}

func (h *RecoveryHandler) ExecuteWithRecovery(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			exception := &Exception{
				Type:       "panic",
				Message:    fmt.Sprintf("%v", r),
				Timestamp:  time.Now(),
				ErrorID:   GenerateErrorID(),
				Recovered: true,
				Severity:  SeverityCritical,
			}
			exception.StackTrace = captureStackTrace(3)

			GetExceptionHandlerChain().Handle(exception)

			var result interface{}
			h.mu.RLock()
			handlers := make([]func(interface{}) interface{}, len(h.handlers))
			copy(handlers, h.handlers)
			h.mu.RUnlock()

			for _, handler := range handlers {
				result = handler(r)
				if result == nil {
					break
				}
			}

			if result != nil {
				if resultErr, ok := result.(error); ok {
					err = resultErr
				} else {
					err = fmt.Errorf("%v", result)
				}
			}
		}
	}()

	return fn()
}

func (h *RecoveryHandler) ExecuteWithRetry(fn func() error) (err error) {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := h.backoffBaseMs * int64(attempt*attempt)
			time.Sleep(time.Duration(backoff) * time.Millisecond)
		}

		err = h.ExecuteWithRecovery(fn)
		if err == nil {
			return nil
		}

		if appErr, ok := err.(*AppError); ok {
			if !appErr.Retryable {
				return err
			}
		}

		lastErr = err
	}

	return lastErr
}

type ErrorContext struct {
	UserID       string                 `json:"user_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	Method       string                 `json:"method,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

type ContextAwareError struct {
	*AppError
	Context *ErrorContext
}

func WithContext(appErr *AppError, ctx *ErrorContext) *ContextAwareError {
	return &ContextAwareError{
		AppError: appErr,
		Context:  ctx,
	}
}

func (e *ContextAwareError) ToResponse() map[string]interface{} {
	resp := e.AppError.ToResponse()
	if e.Context != nil {
		resp["context"] = map[string]interface{}{
			"request_id": e.Context.RequestID,
			"endpoint":   e.Context.Endpoint,
			"method":     e.Context.Method,
			"timestamp":  e.Context.Timestamp,
		}
	}
	return resp
}

type ErrorAggregator struct {
	errors     map[Code][]*AppError
	mu         sync.RWMutex
	maxSize    int
	cleanupAge time.Duration
}

func NewErrorAggregator(maxSize int, cleanupAge time.Duration) *ErrorAggregator {
	agg := &ErrorAggregator{
		errors:     make(map[Code][]*AppError),
		maxSize:    maxSize,
		cleanupAge: cleanupAge,
	}

	go agg.cleanupLoop()
	return agg
}

func (a *ErrorAggregator) Add(err *AppError) {
	if err == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	err.HttpStatus = CodeToHTTPStatus(err.Code)

	a.errors[err.Code] = append(a.errors[err.Code], err)

	if len(a.errors[err.Code]) > a.maxSize {
		a.errors[err.Code] = a.errors[err.Code][len(a.errors[err.Code])-a.maxSize:]
	}
}

func (a *ErrorAggregator) Get(code Code) []*AppError {
	a.mu.RLock()
	defer a.mu.RUnlock()

	errors := make([]*AppError, len(a.errors[code]))
	copy(errors, a.errors[code])
	return errors
}

func (a *ErrorAggregator) GetCount(code Code) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.errors[code])
}

func (a *ErrorAggregator) GetAll() map[Code]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	counts := make(map[Code]int)
	for code, errs := range a.errors {
		counts[code] = len(errs)
	}
	return counts
}

func (a *ErrorAggregator) cleanupLoop() {
	ticker := time.NewTicker(a.cleanupAge)
	defer ticker.Stop()

	for range ticker.C {
		a.cleanup()
	}
}

func (a *ErrorAggregator) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	cutoff := time.Now().Add(-a.cleanupAge)
	for code, errs := range a.errors {
		var validErrors []*AppError
		for _, err := range errs {
			if err.Timestamp.After(cutoff) {
				validErrors = append(validErrors, err)
			}
		}
		if len(validErrors) == 0 {
			delete(a.errors, code)
		} else {
			a.errors[code] = validErrors
		}
	}
}

func (a *ErrorAggregator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errors = make(map[Code][]*AppError)
}

type PanicRecorder struct {
	panicCount int64
	panics     []*Exception
	mu         sync.RWMutex
	maxSize    int
}

func NewPanicRecorder(maxSize int) *PanicRecorder {
	return &PanicRecorder{
		panics:  make([]*Exception, 0),
		maxSize: maxSize,
	}
}

func (r *PanicRecorder) Record(p interface{}, stack []byte) {
	atomic.AddInt64(&r.panicCount, 1)

	exception := &Exception{
		Type:        "panic",
		Message:     fmt.Sprintf("%v", p),
		StackTrace:  string(stack),
		Timestamp:   time.Now(),
		ErrorID:    GenerateErrorID(),
		Recovered:   true,
		Severity:    SeverityCritical,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.panics = append(r.panics, exception)
	if len(r.panics) > r.maxSize {
		r.panics = r.panics[len(r.panics)-r.maxSize:]
	}
}

func (r *PanicRecorder) GetCount() int64 {
	return atomic.LoadInt64(&r.panicCount)
}

func (r *PanicRecorder) GetRecent() []*Exception {
	r.mu.RLock()
	defer r.mu.RUnlock()

	panics := make([]*Exception, len(r.panics))
	copy(panics, r.panics)
	return panics
}

func (r *PanicRecorder) Reset() {
	atomic.StoreInt64(&r.panicCount, 0)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.panics = make([]*Exception, 0)
}

var globalPanicRecorder *PanicRecorder

func init() {
	globalPanicRecorder = NewPanicRecorder(100)
}

func GetGlobalPanicRecorder() *PanicRecorder {
	return globalPanicRecorder
}

func RecordPanic(p interface{}, stack []byte) {
	globalPanicRecorder.Record(p, stack)
}

func init() {
	GetExceptionHandlerChain().AddHandler(NewLoggingExceptionHandler())
	GetExceptionHandlerChain().AddHandler(NewMetricsExceptionHandler())
}
