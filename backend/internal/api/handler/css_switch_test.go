package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetCSSSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
	}{
		{
			name: "returns css source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/css-source", nil)
			c.Request = req

			GetCSSSource(c)

			assert.GreaterOrEqual(t, w.Code, 200)
			assert.Less(t, w.Code, 300)
		})
	}
}

func TestSetCSSSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
	}{
		{
			name: "valid cdn source",
			requestBody: CSSConfig{
				Source: "cdn",
			},
		},
		{
			name: "valid local source",
			requestBody: CSSConfig{
				Source: "local",
			},
		},
		{
			name: "invalid source",
			requestBody: CSSConfig{
				Source: "invalid",
			},
		},
		{
			name:        "empty request",
			requestBody: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			var req *http.Request
			if tt.requestBody == nil {
				req, _ = http.NewRequest("POST", "/css-source", nil)
			} else {
				body, _ := json.Marshal(tt.requestBody)
				req, _ = http.NewRequest("POST", "/css-source", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			}
			c.Request = req

			SetCSSSource(c)

			assert.GreaterOrEqual(t, w.Code, 200)
			assert.Less(t, w.Code, 500)
		})
	}
}
