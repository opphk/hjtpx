package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestSuccessResponseWithNilData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

func TestSuccessResponseWithComplexData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{
		"users": []map[string]string{
			{"id": "1", "name": "Alice"},
			{"id": "2", "name": "Bob"},
		},
		"total": 2,
	}

	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

func TestBadRequestResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "invalid parameters")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "invalid parameters", resp.Message)
}

func TestUnauthorizedResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Equal(t, "unauthorized", resp.Message)
}

func TestForbiddenResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, "forbidden", resp.Message)
}

func TestNotFoundResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "user not found")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "user not found", resp.Message)
}

func TestNotFoundResponseWithEmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "resource not found", resp.Message)
}

func TestInternalServerErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalServerError(c, "database error")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "database error", resp.Message)
}

func TestInternalServerErrorResponseWithEmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalServerError(c, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "internal server error", resp.Message)
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 418, "I'm a teapot")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 418, resp.Code)
	assert.Equal(t, "I'm a teapot", resp.Message)
}

func TestErrorResponseWithDifferentCodes(t *testing.T) {
	testCodes := []int{400, 401, 403, 404, 500, 503}

	for _, code := range testCodes {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		Error(c, code, "test message")

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, code, resp.Code, "Code should be %d", code)
	}
}

func TestResponseStructure(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"key": "value"},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded Response
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Code, decoded.Code)
	assert.Equal(t, resp.Message, decoded.Message)
}

func TestResponseWithoutData(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), `"code":0`)
	assert.Contains(t, string(jsonData), `"message":"success"`)
}

func TestResponseJSONOmitsEmptyData(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	assert.NotContains(t, string(jsonData), `"data":null`)
	assert.Contains(t, string(jsonData), `"code":0`)
	assert.Contains(t, string(jsonData), `"message":"success"`)
}
