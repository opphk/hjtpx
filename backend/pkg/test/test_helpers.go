package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func MakeTestRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	var req *http.Request
	if len(reqBody) > 0 {
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func MakeTestRequestWithHeaders(r *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	var req *http.Request
	if len(reqBody) > 0 {
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func AssertEqual(t TestingT, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func AssertStatusCode(t TestingT, w *httptest.ResponseRecorder, expectedCode int) {
	if w.Code != expectedCode {
		t.Errorf("Expected status code %d, got %d", expectedCode, w.Code)
	}
}

func AssertContains(t TestingT, str, substr string) {
	if !containsString(str, substr) {
		t.Errorf("Expected string '%s' to contain '%s'", str, substr)
	}
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > 0 && containsSubstring(str, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func ParseJSONResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func ParseJSONArrayResponse(w *httptest.ResponseRecorder) []interface{} {
	var response []interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

type TestingT interface {
	Errorf(format string, args ...interface{})
}
