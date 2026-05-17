package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEncryptionMiddlewareCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EncryptionMiddleware())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestEncryptionMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EncryptionMiddleware(EncryptionConfig{Enabled: false}))

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestEncryptionMiddlewareExcludedPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EncryptionMiddleware(EncryptionConfig{
		ExcludePaths: []string{"/health"},
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponseEncryptorCreation(t *testing.T) {
	encryptor := NewResponseEncryptor(nil)
	if encryptor == nil {
		t.Fatal("Expected response encryptor to be created")
	}
}

func TestResponseEncryptorEncryptResponse(t *testing.T) {
	encryptor := NewResponseEncryptor([]byte("test-key-1234567890"))

	data := map[string]interface{}{
		"message": "Hello, World! 你好世界！",
		"number":   42,
	}

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	encrypted, err := encryptor.EncryptResponse(c, data)
	if err != nil {
		t.Fatalf("EncryptResponse failed: %v", err)
	}

	if encrypted == "" {
		t.Error("Encrypted response should not be empty")
	}
}

func TestResponseEncryptorWithDisabledEncryption(t *testing.T) {
	encryptor := NewResponseEncryptor([]byte("test-key-1234567890"))
	encryptor.SetConfig(EncryptionConfig{Enabled: true, EncryptResponse: false})

	data := map[string]interface{}{
		"message": "Hello",
	}

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	encrypted, err := encryptor.EncryptResponse(c, data)
	if err != nil {
		t.Fatalf("EncryptResponse failed: %v", err)
	}

	if !json.Valid([]byte(encrypted)) {
		t.Error("Response should be valid JSON when encryption is disabled")
	}
}

func TestRequestDecryptorCreation(t *testing.T) {
	decryptor := NewRequestDecryptor(nil)
	if decryptor == nil {
		t.Fatal("Expected request decryptor to be created")
	}
}

func TestRequestDecryptorValidateDecryptedParams(t *testing.T) {
	decryptor := NewRequestDecryptor(nil)

	params := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
		"email":    "test@example.com",
	}

	err := decryptor.ValidateDecryptedParams(params, []string{"username", "password"})
	if err != nil {
		t.Error("Validation should pass for valid params")
	}

	err = decryptor.ValidateDecryptedParams(params, []string{"username", "missing_field"})
	if err == nil {
		t.Error("Validation should fail for missing field")
	}
}

func TestSecureRequestHandlerCreation(t *testing.T) {
	handler := NewSecureRequestHandler(nil)
	if handler == nil {
		t.Fatal("Expected secure request handler to be created")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("GenerateSecureToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	if len(token) < 20 {
		t.Error("Token should be reasonably long")
	}
}

func TestMaskSensitiveFields(t *testing.T) {
	data := map[string]interface{}{
		"username": "testuser",
		"password": "secretpass",
		"email":    "test@example.com",
		"token":    "abc123token",
	}

	masked := MaskSensitiveFields(data, []string{"password", "token"})

	if masked["username"] != "testuser" {
		t.Error("Username should not be masked")
	}

	if masked["password"] != "[REDACTED]" {
		t.Error("Password should be masked")
	}

	if masked["token"] != "[REDACTED]" {
		t.Error("Token should be masked")
	}

	if masked["email"] != "test@example.com" {
		t.Error("Email should not be masked")
	}
}

func TestValidateEncryptedData(t *testing.T) {
	validBase64 := "SGVsbG8sIFdvcmxkIQ=="
	if !ValidateEncryptedData(validBase64) {
		t.Error("Valid base64 should pass validation")
	}

	tooShort := "abc"
	if ValidateEncryptedData(tooShort) {
		t.Error("Too short data should fail validation")
	}

	invalidBase64 := "not-valid-base64!!!"
	if ValidateEncryptedData(invalidBase64) {
		t.Error("Invalid base64 should fail validation")
	}
}

func TestGetEncryptionInfo(t *testing.T) {
	info := GetEncryptionInfo()

	if !info.Enabled {
		t.Error("Encryption should be enabled")
	}

	if info.Algorithm != "AES-256-GCM" {
		t.Error("Algorithm should be AES-256-GCM")
	}

	if info.KeySize != 32 {
		t.Errorf("Key size should be 32, got %d", info.KeySize)
	}

	if info.NonceSize != 12 {
		t.Errorf("Nonce size should be 12, got %d", info.NonceSize)
	}
}

func TestRequireEncryption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireEncryption())

	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireEncryptionWithKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("custom-encryption-key-12345")
	router := gin.New()
	router.Use(RequireEncryptionWithKey(key))

	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("Strict-Transport-Security header should be set")
	}

	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Error("X-Content-Type-Options header should be set")
	}

	if w.Header().Get("X-Frame-Options") == "" {
		t.Error("X-Frame-Options header should be set")
	}
}

func TestSecurityHeadersMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders(SecurityHeadersMiddleware{Enabled: false}))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("Headers should not be set when disabled")
	}
}

func TestWrapResponse(t *testing.T) {
	encryptor := NewResponseEncryptor(nil)

	data := map[string]interface{}{
		"key": "value",
	}

	response := encryptor.WrapResponse(data, true, "Success")

	if !response["success"].(bool) {
		t.Error("Success should be true")
	}

	if response["message"] != "Success" {
		t.Error("Message should match")
	}

	if response["data"] == nil {
		t.Error("Data should be included")
	}
}

func TestConvertToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		hasData  bool
	}{
		{
			name:    "map input",
			input:   map[string]interface{}{"key": "value"},
			hasData: false,
		},
		{
			name:    "struct input",
			input:   struct{ Name string }{Name: "test"},
			hasData: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := convertToMap(tc.input)
			if result == nil {
				t.Error("Result should not be nil")
			}
		})
	}
}

func TestIsPathExcluded(t *testing.T) {
	tests := []struct {
		path     string
		excluded []string
		expected bool
	}{
		{"/health", []string{"/health", "/metrics"}, true},
		{"/api/v1/test", []string{"/health"}, false},
		{"/api/health/check", []string{"/health"}, true},
	}

	for _, tc := range tests {
		result := isPathExcluded(tc.path, tc.excluded)
		if result != tc.expected {
			t.Errorf("isPathExcluded(%q, %v) = %v, expected %v",
				tc.path, tc.excluded, result, tc.expected)
		}
	}
}

func TestIsPathIncluded(t *testing.T) {
	tests := []struct {
		path     string
		included []string
		expected bool
	}{
		{"/api/test", []string{"/api"}, true},
		{"/other/test", []string{"/api"}, false},
		{"/api/test", []string{}, true},
	}

	for _, tc := range tests {
		result := isPathIncluded(tc.path, tc.included)
		if result != tc.expected {
			t.Errorf("isPathIncluded(%q, %v) = %v, expected %v",
				tc.path, tc.included, result, tc.expected)
		}
	}
}

func TestFullEncryptedRequestResponseCycle(t *testing.T) {
	secretKey := []byte("test-secret-key-12345678")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EncryptionMiddleware(EncryptionConfig{
		SecretKey:        secretKey,
		DecryptRequest:   true,
		EncryptResponse:  true,
		ExcludePaths:     []string{"/health"},
	}))

	router.POST("/api/test", func(c *gin.Context) {
		params, exists := c.Get("decrypted_params")
		if !exists {
			c.JSON(http.StatusOK, gin.H{"status": "no params"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"received": params,
			"status":   "ok",
		})
	})

	testData := map[string]interface{}{
		"username": "testuser",
		"password": "secretpass",
	}

	decryptor := NewRequestDecryptor(secretKey)
	encryptedData, err := decryptor.cryptoService.EncryptParams(testData)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	jsonData, _ := json.Marshal(map[string]interface{}{"data": encryptedData})

	req := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Encrypted", "true")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !json.Valid(w.Body.Bytes()) {
		t.Error("Response should be valid JSON")
	}
}
