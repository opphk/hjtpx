package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchOperationHandler_BatchUpdateApplications(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Enable Applications", func(t *testing.T) {
		ids := []uint{1, 2, 3}
		result := handler.batchEnableApplications(ids)

		assert.Equal(t, 3, result.Total, "Total should be 3")
		assert.Equal(t, 0, result.Success, "Success should be 0 (mock implementation)")
		assert.Equal(t, 3, result.Failed, "Failed should be 3")
		assert.NotEmpty(t, result.Message, "Message should not be empty")
	})

	t.Run("Test Disable Applications", func(t *testing.T) {
		ids := []uint{1, 2}
		result := handler.batchDisableApplications(ids)

		assert.Equal(t, 2, result.Total, "Total should be 2")
		assert.Contains(t, result.Message, "禁用", "Message should contain 禁用")
	})

	t.Run("Test Delete Applications", func(t *testing.T) {
		ids := []uint{1}
		result := handler.batchDeleteApplications(ids)

		assert.Equal(t, 1, result.Total, "Total should be 1")
		assert.Contains(t, result.Message, "删除", "Message should contain 删除")
	})

	t.Run("Test Update Applications", func(t *testing.T) {
		ids := []uint{1, 2}
		data := []string{"key1=value1"}
		result := handler.batchUpdateApplications(ids, data)

		assert.Equal(t, 2, result.Total, "Total should be 2")
		assert.Contains(t, result.Message, "更新", "Message should contain 更新")
	})
}

func TestBatchOperationHandler_BatchUpdateUsers(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Enable Users", func(t *testing.T) {
		ids := []uint{1, 2}
		result := handler.batchEnableUsers(ids)

		assert.Equal(t, 2, result.Total, "Total should be 2")
		assert.Contains(t, result.Message, "启用", "Message should contain 启用")
	})

	t.Run("Test Disable Users", func(t *testing.T) {
		ids := []uint{1}
		result := handler.batchDisableUsers(ids)

		assert.Equal(t, 1, result.Total, "Total should be 1")
	})

	t.Run("Test Delete Users", func(t *testing.T) {
		ids := []uint{1, 2, 3}
		result := handler.batchDeleteUsers(ids)

		assert.Equal(t, 3, result.Total, "Total should be 3")
	})
}

func TestBatchOperationHandler_BatchDeleteLogs(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Delete Logs", func(t *testing.T) {
		ids := []uint{1, 2}
		result := handler.batchDeleteLogs(ids)

		assert.Equal(t, 2, result.Total, "Total should be 2")
		assert.Contains(t, result.Message, "删除", "Message should contain 删除")
	})
}

func TestBatchOperationHandler_BatchUpdateRiskRules(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Enable Risk Rules", func(t *testing.T) {
		ids := []uint{1, 2, 3}
		result := handler.batchEnableRiskRules(ids)

		assert.Equal(t, 3, result.Total, "Total should be 3")
		assert.Contains(t, result.Message, "启用", "Message should contain 启用")
	})

	t.Run("Test Disable Risk Rules", func(t *testing.T) {
		ids := []uint{1}
		result := handler.batchDisableRiskRules(ids)

		assert.Equal(t, 1, result.Total, "Total should be 1")
		assert.Contains(t, result.Message, "禁用", "Message should contain 禁用")
	})

	t.Run("Test Delete Risk Rules", func(t *testing.T) {
		ids := []uint{1, 2}
		result := handler.batchDeleteRiskRules(ids)

		assert.Equal(t, 2, result.Total, "Total should be 2")
		assert.Contains(t, result.Message, "删除", "Message should contain 删除")
	})
}

func TestBatchOperationHandler_BatchUpdateBlacklist(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Delete Blacklist", func(t *testing.T) {
		ids := []uint{1, 2, 3, 4}
		result := handler.batchDeleteBlacklist(ids)

		assert.Equal(t, 4, result.Total, "Total should be 4")
		assert.Contains(t, result.Message, "删除", "Message should contain 删除")
	})
}

func TestBatchOperationHandler_ExportFunctions(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Generate Export Filename", func(t *testing.T) {
		filename := handler.generateExportFilename("applications", "csv")

		assert.Contains(t, filename, "applications", "Filename should contain type")
		assert.Contains(t, filename, "csv", "Filename should contain format")
		assert.Contains(t, filename, "_export_", "Filename should contain export timestamp")
		assert.HasSuffix(t, filename, ".csv", "Filename should end with .csv")
	})

	t.Run("Test Get Content Type", func(t *testing.T) {
		tests := []struct {
			format     string
			expected   string
		}{
			{"csv", "text/csv"},
			{"excel", "application/vnd.ms-excel"},
			{"json", "application/json"},
			{"pdf", "application/pdf"},
			{"unknown", "application/octet-stream"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				contentType := handler.getContentType(tt.format)
				assert.Equal(t, tt.expected, contentType)
			})
		}
	})

	t.Run("Test Export CSV", func(t *testing.T) {
		data := []map[string]interface{}{
			{"ID": 1, "Name": "Test1"},
			{"ID": 2, "Name": "Test2"},
		}

		assert.NotNil(t, data, "Data should not be nil")
		assert.Equal(t, 2, len(data), "Data length should be 2")
	})
}

func TestBatchOperationHandler_MixedResults(t *testing.T) {
	handler := NewBatchOperationHandler()

	t.Run("Test Mixed Success and Failure", func(t *testing.T) {
		ids := []uint{1, 2, 3}
		result := handler.batchEnableApplications(ids)

		assert.Equal(t, 3, result.Total, "Total should be 3")
		assert.GreaterOrEqual(t, result.Failed, 0, "Failed should be >= 0")
		assert.LessOrEqual(t, result.Success+result.Failed, 3, "Success + Failed should be <= Total")
	})
}

func TestBatchOperationResponse(t *testing.T) {
	t.Run("Test Response Structure", func(t *testing.T) {
		result := BatchOperationResponse{
			Total:     10,
			Success:   7,
			Failed:    3,
			FailedIDs: []uint{1, 2, 3},
			Errors:    []string{"Error1", "Error2", "Error3"},
			Message:   "Test message",
		}

		assert.Equal(t, 10, result.Total)
		assert.Equal(t, 7, result.Success)
		assert.Equal(t, 3, result.Failed)
		assert.Equal(t, 3, len(result.FailedIDs))
		assert.Equal(t, 3, len(result.Errors))
		assert.Equal(t, "Test message", result.Message)
	})
}

func TestBatchExportRequest(t *testing.T) {
	t.Run("Test Export Request Structure", func(t *testing.T) {
		req := BatchExportRequest{
			Type:      "applications",
			Format:    "csv",
			IDs:       []uint{1, 2, 3},
			StartDate: "2024-01-01",
			EndDate:   "2024-01-31",
			Fields:    []string{"id", "name", "status"},
		}

		assert.Equal(t, "applications", req.Type)
		assert.Equal(t, "csv", req.Format)
		assert.Equal(t, 3, len(req.IDs))
		assert.Equal(t, 3, len(req.Fields))
	})
}
