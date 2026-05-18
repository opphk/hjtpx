package export

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetExporter(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		expectType interface{}
	}{
		{
			name:       "get CSV exporter",
			format:     FormatCSV,
			expectType: &CSVExporter{},
		},
		{
			name:       "get Excel exporter",
			format:     FormatExcel,
			expectType: &ExcelExporter{},
		},
		{
			name:       "get PDF exporter",
			format:     FormatPDF,
			expectType: &PDFExporter{},
		},
		{
			name:       "get JSON exporter",
			format:     FormatJSON,
			expectType: &JSONExporter{},
		},
		{
			name:       "default to CSV exporter",
			format:     "unknown",
			expectType: &CSVExporter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := GetExporter(tt.format)
			assert.IsType(t, tt.expectType, exporter)
		})
	}
}

func TestConvertLogsToExportData(t *testing.T) {
	now := time.Now()
	logs := []models.VerificationLog{
		{
			SessionID: "session-1",
			Application: models.Application{
				Name: "Test App",
			},
			CaptchaType: "slider",
			Status:      "success",
			IPAddress:   "192.168.1.1",
			RiskScore:   0.1,
			Duration:    1000,
			CreatedAt:   now,
			UserAgent:   "Mozilla/5.0",
		},
		{
			SessionID: "session-2",
			Application: models.Application{
				Name: "Test App",
			},
			CaptchaType: "click",
			Status:      "failed",
			IPAddress:   "192.168.1.2",
			RiskScore:   0.8,
			Duration:    2000,
			CreatedAt:   now.Add(-time.Hour),
			UserAgent:   "Chrome/120.0",
		},
	}

	title := "Test Export"
	data := ConvertLogsToExportData(logs, title)

	assert.Equal(t, title, data.Title)
	assert.Greater(t, len(data.Headers), 0)
	assert.Len(t, data.Rows, 2)
	assert.NotEmpty(t, data.Metadata)
	assert.Equal(t, "2", data.Metadata["Record Count"])
}

func TestCSVExporter_Export(t *testing.T) {
	exporter := NewCSVExporter()
	data := ExportData{
		Title:   "Test CSV",
		Headers: []string{"ID", "Name"},
		Rows: [][]interface{}{
			{1, "Test 1"},
			{2, "Test 2"},
		},
		Metadata: map[string]string{"Key": "Value"},
	}

	result, err := exporter.Export(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, string(result), "Test CSV")
	assert.Contains(t, string(result), "ID")
	assert.Contains(t, string(result), "Test 1")
}

func TestJSONExporter_Export(t *testing.T) {
	exporter := NewJSONExporter()
	data := ExportData{
		Title:   "Test JSON",
		Headers: []string{"ID", "Name"},
		Rows: [][]interface{}{
			{1, "Test 1"},
		},
		Metadata: map[string]string{"Key": "Value"},
	}

	result, err := exporter.Export(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, string(result), "Test JSON")
	assert.Contains(t, string(result), "exported_at")
}

func TestExcelExporter_Export(t *testing.T) {
	exporter := NewExcelExporter()
	data := ExportData{
		Title:   "Test Excel",
		Headers: []string{"ID", "Name", "Value"},
		Rows: [][]interface{}{
			{1, "Test 1", 100},
			{2, "Test 2", 200},
		},
		Metadata: map[string]string{"Generated": "Now"},
	}

	result, err := exporter.Export(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	// Excel files should start with specific magic bytes
	assert.Greater(t, len(result), 0)
}

func TestPDFExporter_Export(t *testing.T) {
	exporter := NewPDFExporter()
	data := ExportData{
		Title:   "Test PDF",
		Headers: []string{"ID", "Name"},
		Rows: [][]interface{}{
			{1, "Test 1"},
		},
		Metadata: map[string]string{"Source": "Test"},
	}

	result, err := exporter.Export(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGenerateLogVisualization(t *testing.T) {
	logs := []models.VerificationLog{
		{Status: "success", CaptchaType: "slider", CreatedAt: time.Now()},
		{Status: "success", CaptchaType: "slider", CreatedAt: time.Now()},
		{Status: "failed", CaptchaType: "click", CreatedAt: time.Now()},
		{Status: "pending", CaptchaType: "jigsaw", CreatedAt: time.Now().Add(-24 * time.Hour)},
	}

	vizData := GenerateLogVisualization(logs, "Test Visualization")

	assert.Equal(t, "Test Visualization", vizData.Title)
	assert.Len(t, vizData.Charts, 3) // status, type, and date charts

	// Check status chart
	statusChart := vizData.Charts[0]
	assert.Equal(t, "Verification Status Distribution", statusChart.Title)
	assert.Equal(t, "pie", statusChart.Type)

	// Check type chart
	typeChart := vizData.Charts[1]
	assert.Equal(t, "Captcha Type Distribution", typeChart.Title)
	assert.Equal(t, "bar", typeChart.Type)
}

func TestVisualizationExporter_ExportHTML(t *testing.T) {
	exporter := NewVisualizationExporter()
	vizData := VisualizationData{
		Title: "Test Visualization Report",
		Charts: []ChartData{
			{
				Title:  "Test Chart",
				Type:   "bar",
				Labels: []string{"A", "B", "C"},
				Values: []interface{}{10, 20, 30},
			},
		},
	}

	result, err := exporter.ExportHTML(vizData)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, string(result), "Test Visualization Report")
	assert.Contains(t, string(result), "echarts")
	assert.Contains(t, string(result), "Test Chart")
}

func TestCreateGoEChartsBar(t *testing.T) {
	data := map[string]int{
		"Success": 100,
		"Failed":  20,
		"Pending": 10,
	}

	chart := CreateGoEChartsBar("Test Chart", data)
	assert.NotNil(t, chart)
}

func TestCreateGoEChartsPie(t *testing.T) {
	data := map[string]int{
		"Type A": 50,
		"Type B": 30,
		"Type C": 20,
	}

	chart := CreateGoEChartsPie("Test Pie Chart", data)
	assert.NotNil(t, chart)
}

func TestNewExporterFunctions(t *testing.T) {
	assert.IsType(t, &CSVExporter{}, NewCSVExporter())
	assert.IsType(t, &ExcelExporter{}, NewExcelExporter())
	assert.IsType(t, &PDFExporter{}, NewPDFExporter())
	assert.IsType(t, &JSONExporter{}, NewJSONExporter())
	assert.IsType(t, &VisualizationExporter{}, NewVisualizationExporter())
}

func TestExportData_Structure(t *testing.T) {
	data := ExportData{
		Title:   "Structure Test",
		Headers: []string{"Col1", "Col2"},
		Rows: [][]interface{}{
			{"Val1", 1},
			{"Val2", 2},
		},
		Metadata: map[string]string{
			"Author": "Test",
			"Date":   "Today",
		},
	}

	assert.Equal(t, "Structure Test", data.Title)
	assert.Len(t, data.Headers, 2)
	assert.Len(t, data.Rows, 2)
	assert.Len(t, data.Metadata, 2)
}

func TestEmptyDataExport(t *testing.T) {
	exporters := []Exporter{
		NewCSVExporter(),
		NewExcelExporter(),
		NewPDFExporter(),
		NewJSONExporter(),
	}

	emptyData := ExportData{
		Title:   "Empty Test",
		Headers: []string{"ID"},
		Rows:    [][]interface{}{},
	}

	for _, exporter := range exporters {
		t.Run("", func(t *testing.T) {
			result, err := exporter.Export(emptyData)
			assert.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}
