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

func TestInitWebhookService(t *testing.T) {
	InitWebhookService()
	assert.NotNil(t, webhookServiceInstance)
}

func TestHandleWebhook(t *testing.T) {
	InitWebhookService()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/webhook", HandleWebhook)

	tests := []struct {
		name         string
		body         interface{}
		setupHeaders func(*http.Request)
		wantCode     int
	}{
		{
			name: "valid webhook request",
			body: WebhookRequest{
				Event: "test-event",
				Data:  map[string]interface{}{"key": "value"},
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/json")
			},
			wantCode: http.StatusOK,
		},
		{
			name: "invalid JSON",
			body: "invalid json",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/json")
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing event field",
			body: map[string]interface{}{
				"data": "test",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Content-Type", "application/json")
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error
			
			if str, ok := tt.body.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.body)
				assert.NoError(t, err)
			}

			req, _ := http.NewRequest("POST", "/api/v1/webhook", bytes.NewBuffer(reqBody))
			tt.setupHeaders(req)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestHandleWebhook_WithSignature(t *testing.T) {
	InitWebhookService()
	webhookServiceInstance.SetSignatureVerifier("test-secret")
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/webhook", HandleWebhook)

	reqBody := WebhookRequest{
		Event: "test-event",
		Data:  "test",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "invalid-signature")
	req.Header.Set("X-Webhook-Timestamp", "1234567890")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOAuth2Endpoints(t *testing.T) {
	InitWebhookService()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.POST("/api/v1/oauth2/providers", RegisterOAuth2Provider)
	router.GET("/api/v1/oauth2/providers", ListOAuth2Providers)
	router.POST("/api/v1/oauth2/initiate", InitiateOAuth2)
	router.POST("/api/v1/oauth2/refresh", RefreshOAuth2Token)
	router.GET("/api/v1/oauth2/callback/:provider", HandleOAuth2Callback)

	t.Run("register oauth2 provider", func(t *testing.T) {
		reqBody := RegisterOAuth2ProviderRequest{
			Name:         "test-provider",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURI:  "https://example.com/callback",
			AuthURL:      "https://auth.example.com/authorize",
			TokenURL:     "https://auth.example.com/token",
			Scope:        []string{"read", "write"},
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/oauth2/providers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("list oauth2 providers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/providers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("initiate oauth2 - provider not found", func(t *testing.T) {
		reqBody := OAuth2InitiateRequest{
			Provider: "nonexistent",
			State:    "test-state",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/oauth2/initiate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("initiate oauth2 - valid", func(t *testing.T) {
		// 先注册 provider
		registerReq := RegisterOAuth2ProviderRequest{
			Name:         "github",
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURI:  "https://example.com/callback",
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
		}
		registerBody, _ := json.Marshal(registerReq)
		registerReqHTTP, _ := http.NewRequest("POST", "/api/v1/oauth2/providers", bytes.NewBuffer(registerBody))
		registerReqHTTP.Header.Set("Content-Type", "application/json")
		registerW := httptest.NewRecorder()
		router.ServeHTTP(registerW, registerReqHTTP)

		// 然后发起授权
		initiateReq := OAuth2InitiateRequest{
			Provider: "github",
			State:    "test-state-123",
		}
		initiateBody, _ := json.Marshal(initiateReq)
		req, _ := http.NewRequest("POST", "/api/v1/oauth2/initiate", bytes.NewBuffer(initiateBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("oauth2 callback - provider not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/oauth2/callback/nonexistent?code=test-code&state=test-state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("refresh token - provider not found", func(t *testing.T) {
		reqBody := OAuth2RefreshRequest{
			Provider:     "nonexistent",
			RefreshToken: "test-refresh-token",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/oauth2/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSetWebhookSecret(t *testing.T) {
	InitWebhookService()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/webhook/secret", SetWebhookSecret)

	tests := []struct {
		name     string
		body     SetWebhookSecretRequest
		wantCode int
	}{
		{
			name: "valid secret",
			body: SetWebhookSecretRequest{
				Secret: "my-secret-key",
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "missing secret",
			body:     SetWebhookSecretRequest{},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", "/api/v1/webhook/secret", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestTestWebhookSignature(t *testing.T) {
	InitWebhookService()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/webhook/test-signature", TestWebhookSignature)

	req, _ := http.NewRequest("POST", "/api/v1/webhook/test-signature", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
