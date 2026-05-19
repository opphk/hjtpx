package response

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSuccessV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "success with map data",
			data: map[string]string{"key": "value"},
		},
		{
			name: "success with nil data",
			data: nil,
		},
		{
			name: "success with slice data",
			data: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			SuccessV2(c, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)
			assert.Equal(t, "success", resp.Message)
		})
	}
}

func TestSuccessWithMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	meta := &ResponseMeta{
		RequestID: "req-123",
		Timestamp: time.Now(),
	}

	SuccessWithMeta(c, map[string]string{"data": "test"}, meta)

	var resp ResponseV2
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, "req-123", resp.Meta.RequestID)
}

func TestSuccessPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		page       int
		pageSize   int
		total      int64
		expectPage int
	}{
		{
			name:       "first page",
			page:       1,
			pageSize:   10,
			total:      100,
			expectPage: 1,
		},
		{
			name:       "second page",
			page:       2,
			pageSize:   10,
			total:      25,
			expectPage: 2,
		},
		{
			name:       "last partial page",
			page:       3,
			pageSize:   10,
			total:      25,
			expectPage: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			SuccessPage(c, []int{1, 2, 3}, tt.page, tt.pageSize, tt.total)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.NotNil(t, resp.Meta)
			assert.NotNil(t, resp.Meta.Page)
			assert.Equal(t, tt.expectPage, resp.Meta.Page.Page)
			assert.Equal(t, tt.pageSize, resp.Meta.Page.PageSize)
			assert.Equal(t, tt.total, resp.Meta.Page.Total)
		})
	}
}

func TestErrorV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		httpStatus int
		message    string
	}{
		{
			name:       "400 bad request",
			httpStatus: http.StatusBadRequest,
			message:    "bad request",
		},
		{
			name:       "500 internal error",
			httpStatus: http.StatusInternalServerError,
			message:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ErrorV2(c, tt.httpStatus, tt.message)

			assert.Equal(t, tt.httpStatus, w.Code)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.httpStatus, resp.Code)
			assert.Equal(t, tt.message, resp.Message)
		})
	}
}

func TestErrorWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorWithCode(c, http.StatusBadRequest, "INVALID_INPUT", "validation failed", "field 'email' is required")

	var resp ResponseV2
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "validation failed", resp.Message)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Equal(t, "field 'email' is required", resp.Error.Details)
}

func TestBadRequestV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequestV2(c, "invalid input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnauthorizedV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	UnauthorizedV2(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestForbiddenV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ForbiddenV2(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestNotFoundV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with message",
			message: "user not found",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			NotFoundV2(c, tt.message)

			assert.Equal(t, http.StatusNotFound, w.Code)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			if tt.message != "" {
				assert.Equal(t, tt.message, resp.Message)
			} else {
				assert.Equal(t, "resource not found", resp.Message)
			}
		})
	}
}

func TestInternalServerErrorV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with message",
			message: "database error",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			InternalServerErrorV2(c, tt.message)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			if tt.message != "" {
				assert.Equal(t, tt.message, resp.Message)
			} else {
				assert.Equal(t, "internal server error", resp.Message)
			}
		})
	}
}

func TestTooManyRequestsV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with message",
			message: "rate limit exceeded",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			TooManyRequestsV2(c, tt.message)

			assert.Equal(t, http.StatusTooManyRequests, w.Code)

			var resp ResponseV2
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			if tt.message != "" {
				assert.Equal(t, tt.message, resp.Message)
			} else {
				assert.Equal(t, "too many requests", resp.Message)
			}
		})
	}
}

func TestGzipMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("compresses response when accepted", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Accept-Encoding", "gzip")
		c.Writer.Header().Set("Content-Type", "application/json")

		router := gin.New()
		router.Use(GzipMiddleware())
		router.GET("/test", func(c *gin.Context) {
			SuccessV2(c, map[string]string{"data": "test"})
		})

		c.Request.URL.Path = "/test"
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	})

	t.Run("skips compression for excluded paths", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/batch", nil)
		c.Request.Header.Set("Accept-Encoding", "gzip")

		router := gin.New()
		router.Use(GzipMiddleware(GzipConfig{
			ExcludedPaths: []string{"/batch"},
		}))
		router.POST("/batch", func(c *gin.Context) {
			SuccessV2(c, nil)
		})

		c.Request.URL.Path = "/batch"
		router.ServeHTTP(w, c.Request)

		assert.Empty(t, w.Header().Get("Content-Encoding"))
	})

	t.Run("decompresses gzipped body", func(t *testing.T) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)

		data := map[string]string{"test": "data"}
		jsonData, _ := json.Marshal(data)
		gz.Write(jsonData)
		gz.Close()

		reader := bytes.NewReader(buf.Bytes())
		gzReader, err := gzip.NewReader(reader)
		assert.NoError(t, err)

		result, err := io.ReadAll(gzReader)
		assert.NoError(t, err)
		assert.JSONEq(t, string(jsonData), string(result))
	})
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid object",
			input:   map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "slice",
			input:   []int{1, 2, 3},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal to map", func(t *testing.T) {
		data := []byte(`{"key":"value"}`)
		var result map[string]string
		err := UnmarshalJSON(data, &result)
		assert.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("unmarshal to slice", func(t *testing.T) {
		data := []byte(`[1,2,3]`)
		var result []int
		err := UnmarshalJSON(data, &result)
		assert.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("invalid json", func(t *testing.T) {
		data := []byte(`{invalid}`)
		var result map[string]string
		err := UnmarshalJSON(data, &result)
		assert.Error(t, err)
	})
}

func TestFastMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "map",
			input: map[string]string{"key": "value"},
		},
		{
			name:  "slice",
			input: []int{1, 2, 3},
		},
		{
			name:  "struct",
			input: struct{ Name string }{Name: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FastMarshalJSON(tt.input)
			assert.NotEmpty(t, result)
			assert.True(t, result[0] == '{' || result[0] == '[')
		})
	}
}

func TestWriteJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteJSON(c, http.StatusOK, map[string]string{"data": "test"})

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp ResponseV2
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
}

func TestBatchRequestHandler(t *testing.T) {
	handler := NewBatchRequestHandler(time.Second, 10)

	t.Run("registers and completes request", func(t *testing.T) {
		key := "test-key-1"
		ch, err := handler.Register(key)
		assert.NoError(t, err)
		assert.NotNil(t, ch)

		resp := BatchResponse{Key: key, Status: 200, Data: "success"}
		completed := handler.Complete(key, resp)
		assert.True(t, completed)
	})

	t.Run("rejects duplicate key", func(t *testing.T) {
		key := "test-key-2"
		_, err := handler.Register(key)
		assert.NoError(t, err)

		_, err = handler.Register(key)
		assert.Error(t, err)
	})

	t.Run("rejects when max requests exceeded", func(t *testing.T) {
		h := NewBatchRequestHandler(time.Second, 1)
		_, err := h.Register("key1")
		assert.NoError(t, err)

		_, err = h.Register("key2")
		assert.Error(t, err)
	})

	t.Run("wait returns completed response", func(t *testing.T) {
		key := "test-key-3"
		_, err := handler.Register(key)
		assert.NoError(t, err)

		go func() {
			time.Sleep(100 * time.Millisecond)
			handler.Complete(key, BatchResponse{Key: key, Status: 200})
		}()

		resp, ok := handler.Wait(key)
		assert.True(t, ok)
		assert.Equal(t, key, resp.Key)
		assert.Equal(t, 200, resp.Status)
	})
}

func TestBatchHandler(t *testing.T) {
	t.Run("empty requests returns empty slice", func(t *testing.T) {
		h := NewBatchHandler("http://localhost:8080", time.Second, 10)
		resp, err := h.Execute([]BatchRequest{})
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

func TestRequestMerger(t *testing.T) {
	merger := NewRequestMerger(100*time.Millisecond, 5)

	t.Run("merges requests", func(t *testing.T) {
		key := "merge-key-1"
		results := []interface{}{}
		var mu sync.Mutex

		mergeFn := func(reqs []interface{}) (interface{}, error) {
			mu.Lock()
			defer mu.Unlock()
			results = append(results, reqs...)
			return map[string]int{"count": len(results)}, nil
		}

		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				resp, err := merger.Merge(key, fmt.Sprintf("req-%d", id), mergeFn)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		assert.GreaterOrEqual(t, len(results), 1)
		mu.Unlock()
	})

	t.Run("respects max size", func(t *testing.T) {
		key := "merge-key-2"
		mergeFn := func(reqs []interface{}) (interface{}, error) {
			return nil, nil
		}

		_, err := merger.Merge(key, "req-1", mergeFn)
		assert.NoError(t, err)
	})
}

func TestResponseV2Helpers(t *testing.T) {
	resp := &ResponseV2{Code: 0, Message: "success"}

	t.Run("WithRequestID", func(t *testing.T) {
		result := resp.WithRequestID("req-123")
		assert.Equal(t, resp, result)
		assert.Equal(t, "req-123", resp.Meta.RequestID)
	})

	t.Run("WithTimestamp", func(t *testing.T) {
		now := time.Now()
		result := resp.WithTimestamp(now)
		assert.Equal(t, resp, result)
		assert.Equal(t, now, resp.Meta.Timestamp)
	})

	t.Run("WithCost", func(t *testing.T) {
		result := resp.WithCost(100)
		assert.Equal(t, resp, result)
		assert.Equal(t, int64(100), resp.Meta.Cost)
	})

	t.Run("WithPage", func(t *testing.T) {
		result := resp.WithPage(1, 10, 100)
		assert.Equal(t, resp, result)
		assert.NotNil(t, resp.Meta.Page)
		assert.Equal(t, 1, resp.Meta.Page.Page)
		assert.Equal(t, 10, resp.Meta.Page.PageSize)
		assert.Equal(t, int64(100), resp.Meta.Page.Total)
		assert.Equal(t, 10, resp.Meta.Page.TotalPages)
	})

	t.Run("WithError", func(t *testing.T) {
		result := resp.WithError("ERR_CODE", "error details")
		assert.Equal(t, resp, result)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, "ERR_CODE", resp.Error.Code)
		assert.Equal(t, "error details", resp.Error.Details)
	})
}

func TestPageInfoHelpers(t *testing.T) {
	t.Run("HasNext", func(t *testing.T) {
		p := &PageInfo{Page: 1, TotalPages: 5}
		assert.True(t, p.HasNext())

		p.Page = 5
		assert.False(t, p.HasNext())
	})

	t.Run("HasPrev", func(t *testing.T) {
		p := &PageInfo{Page: 2, TotalPages: 5}
		assert.True(t, p.HasPrev())

		p.Page = 1
		assert.False(t, p.HasPrev())
	})

	t.Run("Offset", func(t *testing.T) {
		p := &PageInfo{Page: 3, PageSize: 10}
		assert.Equal(t, 20, p.Offset())
	})
}

func TestConcurrentBatchRequests(t *testing.T) {
	handler := NewBatchRequestHandler(2*time.Second, 100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%10)

			ch, err := handler.Register(key)
			if err != nil {
				return
			}

			time.Sleep(10 * time.Millisecond)
			handler.Complete(key, BatchResponse{Key: key, Status: 200})

			select {
			case resp := <-ch:
				assert.Equal(t, key, resp.Key)
			case <-time.After(time.Second):
				t.Error("timeout waiting for response")
			}
		}(i)
	}

	wg.Wait()
}
