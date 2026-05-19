package response

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

var jsonConfig = jsoniter.Config{
	EscapeHTML:             false,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

var defaultGzipWriter = &gzipWriterPool{
	pool: sync.Pool{
		New: func() interface{} {
			return &gzipWriter{
				buf: new(bytes.Buffer),
			}
		},
	},
}

type gzipWriterPool struct {
	pool sync.Pool
}

func (p *gzipWriterPool) Get() *gzipWriter {
	w := p.pool.Get().(*gzipWriter)
	w.buf.Reset()
	return w
}

func (p *gzipWriterPool) Put(w *gzipWriter) {
	p.pool.Put(w)
}

type gzipWriter struct {
	w       gzip.Writer
	buf     *bytes.Buffer
	encoded []byte
}

func (gw *gzipWriter) Write(p []byte) (int, error) {
	gw.buf.Write(p)
	return len(p), nil
}

func (gw *gzipWriter) Close() error {
	return gw.w.Close()
}

func (gw *gzipWriter) Reset(w io.Writer) {
	gw.w.Reset(w)
	gw.buf.Reset()
}

func (gw *gzipWriter) encode(dst io.Writer, level int) error {
	buf := new(bytes.Buffer)
	gz, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return err
	}
	_, err = gz.Write(gw.buf.Bytes())
	if err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	_, err = dst.Write(buf.Bytes())
	return err
}

type PageInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type ResponseV2 struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data,omitempty"`
	Meta    *ResponseMeta   `json:"meta,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

type ResponseMeta struct {
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Page      *PageInfo `json:"page,omitempty"`
	Cost      int64     `json:"cost_ms,omitempty"`
}

type ResponseError struct {
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type BatchRequest struct {
	Key      string      `json:"key"`
	Endpoint string      `json:"endpoint"`
	Method   string      `json:"method"`
	Data     interface{} `json:"data,omitempty"`
}

type BatchResponse struct {
	Key    string      `json:"key"`
	Status int         `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type BatchRequestHandler struct {
	timeout    time.Duration
	maxReqs    int
	mu         sync.RWMutex
	pendingReq map[string]chan BatchResponse
}

func NewBatchRequestHandler(timeout time.Duration, maxRequests int) *BatchRequestHandler {
	return &BatchRequestHandler{
		timeout:    timeout,
		maxReqs:     maxRequests,
		pendingReq:  make(map[string]chan BatchResponse),
	}
}

func (h *BatchRequestHandler) Register(key string) (chan BatchResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.pendingReq) >= h.maxReqs {
		return nil, fmt.Errorf("max pending requests exceeded: %d", h.maxReqs)
	}

	if _, exists := h.pendingReq[key]; exists {
		return nil, fmt.Errorf("request key already exists: %s", key)
	}

	ch := make(chan BatchResponse, 1)
	h.pendingReq[key] = ch
	return ch, nil
}

func (h *BatchRequestHandler) Complete(key string, resp BatchResponse) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch, exists := h.pendingReq[key]
	if !exists {
		return false
	}

	select {
	case ch <- resp:
		delete(h.pendingReq, key)
		return true
	default:
		delete(h.pendingReq, key)
		return false
	}
}

func (h *BatchRequestHandler) Wait(key string) (BatchResponse, bool) {
	h.mu.RLock()
	ch, exists := h.pendingReq[key]
	h.mu.RUnlock()

	if !exists {
		return BatchResponse{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	select {
	case resp := <-ch:
		return resp, true
	case <-ctx.Done():
		return BatchResponse{
			Key:    key,
			Status: http.StatusGatewayTimeout,
			Error:  "request timeout",
		}, false
	}
}

func SuccessV2(c *gin.Context, data interface{}) {
	SuccessWithMeta(c, data, nil)
}

func SuccessWithMeta(c *gin.Context, data interface{}, meta *ResponseMeta) {
	resp := ResponseV2{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
		Meta:    meta,
	}
	c.JSON(http.StatusOK, resp)
}

func SuccessPage(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	meta := &ResponseMeta{
		Page: &PageInfo{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
	SuccessWithMeta(c, data, meta)
}

func ErrorV2(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, ResponseV2{
		Code:    httpStatus,
		Message: message,
	})
}

func ErrorWithCode(c *gin.Context, httpStatus int, code, message, details string) {
	resp := ResponseV2{
		Code:    httpStatus,
		Message: message,
		Error: &ResponseError{
			Code:    code,
			Details: details,
		},
	}
	c.JSON(httpStatus, resp)
}

func BadRequestV2(c *gin.Context, message string) {
	ErrorV2(c, http.StatusBadRequest, message)
}

func UnauthorizedV2(c *gin.Context) {
	ErrorV2(c, http.StatusUnauthorized, "unauthorized")
}

func ForbiddenV2(c *gin.Context) {
	ErrorV2(c, http.StatusForbidden, "forbidden")
}

func NotFoundV2(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	ErrorV2(c, http.StatusNotFound, message)
}

func InternalServerErrorV2(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	ErrorV2(c, http.StatusInternalServerError, message)
}

func TooManyRequestsV2(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}
	ErrorV2(c, http.StatusTooManyRequests, message)
}

type GzipConfig struct {
	Level           int
	MinContentSize  int
	ExcludedPaths   []string
	IncludedTypes   []string
}

var defaultGzipConfig = GzipConfig{
	Level:          gzip.DefaultCompression,
	MinContentSize: 1024,
	IncludedTypes:  []string{"application/json", "text/plain"},
}

func GzipMiddleware(config ...GzipConfig) gin.HandlerFunc {
	cfg := defaultGzipConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !shouldCompress(c, cfg) {
			c.Next()
			return
		}

		if c.Request.Header.Get("Accept-Encoding") != "gzip" &&
			c.Request.Header.Get("TE") != "gzip" {
			c.Next()
			return
		}

		gz, err := gzip.NewWriterLevel(c.Writer, cfg.Level)
		if err != nil {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Vary", "Accept-Encoding")

		originalWriter := c.Writer
		c.Writer = &gzipResponseWriter{
			ResponseWriter: originalWriter,
			Writer:         gz,
		}

		c.Next()

		gz.Close()
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	Writer *gzip.Writer
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.Writer.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Writer.Write([]byte(s))
}

func (w *gzipResponseWriter) Close() error {
	return w.Writer.Close()
}

func shouldCompress(c *gin.Context, cfg GzipConfig) bool {
	if c.Request.Method == "POST" && c.Request.URL.Path == "/batch" {
		return false
	}

	for _, path := range cfg.ExcludedPaths {
		if c.Request.URL.Path == path {
			return false
		}
	}

	contentType := c.Writer.Header().Get("Content-Type")
	for _, t := range cfg.IncludedTypes {
		if len(contentType) >= len(t) && contentType[:len(t)] == t {
			return true
		}
	}

	return false
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return jsonConfig.Marshal(v)
}

func UnmarshalJSON(data []byte, v interface{}) error {
	return jsonConfig.Unmarshal(data, v)
}

func FastMarshalJSON(v interface{}) string {
	if b, err := jsonConfig.Marshal(v); err == nil {
		return string(b)
	}
	return "{}"
}

func WriteJSON(c *gin.Context, status int, v interface{}) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(status)

	if data, err := jsonConfig.Marshal(v); err == nil {
		c.Writer.Write(data)
		return
	}

	c.Writer.WriteString("{}")
}

func WriteGzipJSON(c *gin.Context, status int, v interface{}) bool {
	if c.Writer.Header().Get("Content-Encoding") != "gzip" {
		WriteJSON(c, status, v)
		return false
	}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(status)

	if data, err := jsonConfig.Marshal(v); err == nil {
		gz := defaultGzipWriter.Get()
		defer defaultGzipWriter.Put(gz)

		if err := gz.encode(c.Writer, gzip.DefaultCompression); err == nil {
			gz.buf.Write(data)
			return true
		}
	}

	return false
}

type BatchHandler struct {
	client  *http.Client
	baseURL string
	handler *BatchRequestHandler
}

func NewBatchHandler(baseURL string, timeout time.Duration, maxRequests int) *BatchHandler {
	return &BatchHandler{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		handler: NewBatchRequestHandler(timeout, maxRequests),
	}
}

func (h *BatchHandler) Execute(reqs []BatchRequest) ([]BatchResponse, error) {
	if len(reqs) == 0 {
		return []BatchResponse{}, nil
	}

	var wg sync.WaitGroup
	responses := make([]BatchResponse, len(reqs))
	errChan := make(chan error, len(reqs))

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r BatchRequest) {
			defer wg.Done()

			resp, err := h.executeSingle(r)
			if err != nil {
				errChan <- fmt.Errorf("request %s failed: %w", r.Key, err)
				responses[idx] = BatchResponse{
					Key:    r.Key,
					Status: http.StatusInternalServerError,
					Error:  err.Error(),
				}
				return
			}
			responses[idx] = resp
		}(i, req)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return responses, fmt.Errorf("batch execution had %d errors", len(errs))
	}

	return responses, nil
}

func (h *BatchHandler) executeSingle(req BatchRequest) (BatchResponse, error) {
	url := h.baseURL + req.Endpoint

	var body io.Reader
	if req.Data != nil {
		data, err := MarshalJSON(req.Data)
		if err != nil {
			return BatchResponse{}, err
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequest(req.Method, url, body)
	if err != nil {
		return BatchResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return BatchResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchResponse{}, err
	}

	var respData interface{}
	if len(respBody) > 0 {
		UnmarshalJSON(respBody, &respData)
	}

	return BatchResponse{
		Key:    req.Key,
		Status: resp.StatusCode,
		Data:   respData,
	}, nil
}

type RequestMerger struct {
	mu       sync.RWMutex
	pending  map[string]*MergeRequest
	timeout  time.Duration
	maxSize  int
}

type MergeRequest struct {
	Key      string
	Reqs     []interface{}
	Done     chan struct{}
	Response interface{}
	Err      error
	mu       sync.Mutex
}

func NewRequestMerger(timeout time.Duration, maxSize int) *RequestMerger {
	return &RequestMerger{
		pending: make(map[string]*MergeRequest),
		timeout: timeout,
		maxSize: maxSize,
	}
}

func (m *RequestMerger) Merge(key string, req interface{}, mergeFn func([]interface{}) (interface{}, error)) (interface{}, error) {
	m.mu.Lock()

	existing, exists := m.pending[key]
	if exists {
		if len(existing.Reqs) >= m.maxSize {
			m.mu.Unlock()
			return nil, fmt.Errorf("max merge size exceeded for key: %s", key)
		}
		existing.Reqs = append(existing.Reqs, req)
		m.mu.Unlock()
		<-existing.Done
		return existing.Response, existing.Err
	}

	mergeReq := &MergeRequest{
		Key:      key,
		Reqs:     []interface{}{req},
		Done:     make(chan struct{}),
	}
	m.pending[key] = mergeReq
	m.mu.Unlock()

	go func() {
		time.Sleep(m.timeout)
		m.executeAndNotify(key, mergeReq, mergeFn)
	}()

	<-mergeReq.Done
	return mergeReq.Response, mergeReq.Err
}

func (m *RequestMerger) AddAndWait(key string, req interface{}, mergeFn func([]interface{}) (interface{}, error)) (interface{}, error) {
	m.mu.Lock()
	existing, exists := m.pending[key]
	if !exists {
		m.mu.Unlock()
		return m.Merge(key, req, mergeFn)
	}

	if len(existing.Reqs) >= m.maxSize {
		m.mu.Unlock()
		return nil, fmt.Errorf("max merge size exceeded")
	}

	existing.Reqs = append(existing.Reqs, req)
	reqDone := make(chan struct{})
	m.mu.Unlock()

	select {
	case <-existing.Done:
		return existing.Response, existing.Err
	case <-reqDone:
		return existing.Response, existing.Err
	}
}

func (m *RequestMerger) executeAndNotify(key string, mergeReq *MergeRequest, mergeFn func([]interface{}) (interface{}, error)) {
	mergeReq.mu.Lock()
	reqs := make([]interface{}, len(mergeReq.Reqs))
	copy(reqs, mergeReq.Reqs)

	resp, err := mergeFn(reqs)
	mergeReq.Response = resp
	mergeReq.Err = err
	mergeReq.mu.Unlock()

	m.mu.Lock()
	delete(m.pending, key)
	m.mu.Unlock()

	close(mergeReq.Done)
}

type mergeRequestInternal struct {
	*MergeRequest
}

func (r *ResponseV2) WithRequestID(id string) *ResponseV2 {
	if r.Meta == nil {
		r.Meta = &ResponseMeta{}
	}
	r.Meta.RequestID = id
	return r
}

func (r *ResponseV2) WithTimestamp(t time.Time) *ResponseV2 {
	if r.Meta == nil {
		r.Meta = &ResponseMeta{}
	}
	r.Meta.Timestamp = t
	return r
}

func (r *ResponseV2) WithCost(costMs int64) *ResponseV2 {
	if r.Meta == nil {
		r.Meta = &ResponseMeta{}
	}
	r.Meta.Cost = costMs
	return r
}

func (r *ResponseV2) WithPage(page, pageSize int, total int64) *ResponseV2 {
	if r.Meta == nil {
		r.Meta = &ResponseMeta{}
	}
	r.Meta.Page = &PageInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: int(total) / pageSize,
	}
	if int(total)%pageSize > 0 {
		r.Meta.Page.TotalPages++
	}
	return r
}

func (r *ResponseV2) WithError(code, details string) *ResponseV2 {
	r.Error = &ResponseError{
		Code:    code,
		Details: details,
	}
	return r
}

func (p *PageInfo) HasNext() bool {
	return p.Page < p.TotalPages
}

func (p *PageInfo) HasPrev() bool {
	return p.Page > 1
}

func (p *PageInfo) Offset() int {
	return (p.Page - 1) * p.PageSize
}
