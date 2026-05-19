package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComplianceService_GetComplianceStatus(t *testing.T) {
	service := NewComplianceService()
	
	result, err := service.GetComplianceStatus()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Contains(t, result, "gdpr")
	assert.Contains(t, result, "soc2")
	assert.Contains(t, result, "iso27001")
	assert.Contains(t, result, "ccpa")
}

func TestComplianceService_GenerateSecurityComplianceReport(t *testing.T) {
	service := NewComplianceService()
	
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()
	
	result, err := service.GenerateSecurityComplianceReport(startDate, endDate)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Equal(t, "Security Compliance", result["report_type"])
	assert.Contains(t, result, "summary")
	assert.Contains(t, result, "security_metrics")
	assert.Contains(t, result, "compliance_status")
}

func TestComplianceService_ExportComplianceReport(t *testing.T) {
	service := NewComplianceService()
	
	params := map[string]interface{}{
		"start_date": "2024-01-01",
		"end_date":   "2024-01-31",
	}
	
	result, err := service.ExportComplianceReport("security", params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)
}

func TestComplianceService_ExportComplianceReport_InvalidType(t *testing.T) {
	service := NewComplianceService()
	
	result, err := service.ExportComplianceReport("invalid", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}