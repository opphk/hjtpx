package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xuri/excelize/v2"
)

type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

func (s *LogService) CreateVerificationLog(log *models.VerificationLog) error {
	return database.DB.Create(log).Error
}

func (s *LogService) GetLogByID(id uint) (*models.VerificationLog, error) {
	var log models.VerificationLog
	err := database.DB.Preload("Verification").
		Preload("Verification.BehaviorData").
		Preload("Application").
		First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

type LogQueryParams struct {
	Page          int
	PageSize      int
	ApplicationID uint
	Status        string
	CaptchaType   string
	SessionID     string
	StartDate     time.Time
	EndDate       time.Time
	MinRiskScore  float64
	MaxRiskScore  float64
	IPAddress     string
	UserAgent     string
}

type LogListResult struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
	Logs       []models.VerificationLog
}

func (s *LogService) QueryLogs(params LogQueryParams) (*LogListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	query := database.DB.Model(&models.VerificationLog{})

	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CaptchaType != "" {
		query = query.Where("captcha_type = ?", params.CaptchaType)
	}
	if params.SessionID != "" {
		query = query.Where("session_id LIKE ?", "%"+params.SessionID+"%")
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}
	if params.MinRiskScore > 0 {
		query = query.Where("risk_score >= ?", params.MinRiskScore)
	}
	if params.MaxRiskScore > 0 {
		query = query.Where("risk_score <= ?", params.MaxRiskScore)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []models.VerificationLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.Preload("Application").
		Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &LogListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Logs:       logs,
	}, nil
}

type LogExportParams struct {
	ApplicationID uint
	Status        string
	CaptchaType   string
	StartDate     time.Time
	EndDate       time.Time
	Format        string
	RiskLevel     string
	IPAddress     string
	UserAgent     string
	MinRiskScore  float64
	MaxRiskScore  float64
}

func (s *LogService) ExportLogs(params LogExportParams) ([]byte, string, error) {
	query := database.DB.Model(&models.VerificationLog{})

	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CaptchaType != "" {
		query = query.Where("captcha_type = ?", params.CaptchaType)
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at < ?", params.EndDate)
	}
	if params.IPAddress != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddress+"%")
	}
	if params.UserAgent != "" {
		query = query.Where("user_agent LIKE ?", "%"+params.UserAgent+"%")
	}
	if params.MinRiskScore > 0 {
		query = query.Where("risk_score >= ?", params.MinRiskScore)
	}
	if params.MaxRiskScore > 0 {
		query = query.Where("risk_score <= ?", params.MaxRiskScore)
	}

	var logs []models.VerificationLog
	if err := query.Preload("Application").
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, "", err
	}

	switch params.Format {
	case "xlsx", "excel":
		data, err := s.exportToExcel(logs)
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	case "pdf":
		data, err := s.exportToPDF(logs)
		return data, "application/pdf", err
	case "json":
		data, err := s.exportToJSON(logs)
		return data, "application/json", err
	default:
		data, err := s.exportToCSV(logs)
		return data, "text/csv", err
	}
}

func (s *LogService) exportToExcel(logs []models.VerificationLog) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheetName := "Verification Logs"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"ID", "Session ID", "Application", "Captcha Type",
		"Status", "IP Address", "Risk Score", "Risk Level",
		"Duration (ms)", "User Agent", "Created At",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
	})
	_ = f.SetCellStyle(sheetName, "A1", fmt.Sprintf("K1"), headerStyle)

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheetName, cell, header)
	}

	for rowIdx, log := range logs {
		rowNum := rowIdx + 2
		riskLevel := calculateRiskLevelFromScore(log.RiskScore)

		values := []interface{}{
			log.ID,
			log.SessionID,
			getApplicationName(log),
			log.CaptchaType,
			log.Status,
			log.IPAddress,
			log.RiskScore,
			riskLevel,
			log.Duration,
			log.UserAgent,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		for colIdx, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, value)
		}
	}

	for i := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetName, colName, colName, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *LogService) exportToPDF(logs []models.VerificationLog) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Verification Logs Report")
	pdf.Ln(15)

	pdf.SetFont("Arial", "B", 8)
	colWidths := []float64{15, 30, 25, 20, 18, 30, 20, 18, 25, 30}
	headers := []string{"ID", "Session", "App", "Type", "Status", "IP", "Risk", "Level", "Duration", "Created"}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 7)
	for _, log := range logs {
		riskLevel := calculateRiskLevelFromScore(log.RiskScore)

		row := []string{
			fmt.Sprintf("%d", log.ID),
			truncateString(log.SessionID, 15),
			truncateString(getApplicationName(log), 12),
			truncateString(log.CaptchaType, 10),
			truncateString(log.Status, 9),
			truncateString(log.IPAddress, 15),
			fmt.Sprintf("%.1f", log.RiskScore),
			truncateString(riskLevel, 9),
			fmt.Sprintf("%dms", log.Duration),
			truncateString(log.CreatedAt.Format("2006-01-02 15:04"), 15),
		}

		for i, cell := range row {
			pdf.CellFormat(colWidths[i], 5, cell, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.SetFont("Arial", "I", 8)
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Generated at: %s | Total records: %d", time.Now().Format("2006-01-02 15:04:05"), len(logs)))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *LogService) exportToJSON(logs []models.VerificationLog) ([]byte, error) {
	type LogExport struct {
		ID            uint      `json:"id"`
		SessionID     string    `json:"session_id"`
		Application   string    `json:"application"`
		CaptchaType   string    `json:"captcha_type"`
		Status        string    `json:"status"`
		IPAddress     string    `json:"ip_address"`
		RiskScore     float64   `json:"risk_score"`
		RiskLevel     string    `json:"risk_level"`
		Duration      int64     `json:"duration"`
		UserAgent     string    `json:"user_agent"`
		CreatedAt     time.Time `json:"created_at"`
	}

	exportLogs := make([]LogExport, len(logs))
	for i, log := range logs {
		exportLogs[i] = LogExport{
			ID:          log.ID,
			SessionID:   log.SessionID,
			Application: getApplicationName(log),
			CaptchaType: log.CaptchaType,
			Status:      log.Status,
			IPAddress:   log.IPAddress,
			RiskScore:   log.RiskScore,
			RiskLevel:   calculateRiskLevelFromScore(log.RiskScore),
			Duration:    log.Duration,
			UserAgent:   log.UserAgent,
			CreatedAt:   log.CreatedAt,
		}
	}

	exportData := map[string]interface{}{
		"export_time": time.Now().Format(time.RFC3339),
		"total_count": len(logs),
		"logs":        exportLogs,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

func calculateRiskLevelFromScore(score float64) string {
	if score >= 80 {
		return "critical"
	} else if score >= 60 {
		return "high"
	} else if score >= 30 {
		return "medium"
	}
	return "low"
}

func getApplicationName(log models.VerificationLog) string {
	if log.Application.Name != "" {
		return log.Application.Name
	}
	if log.ApplicationID > 0 {
		return fmt.Sprintf("App-%d", log.ApplicationID)
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (s *LogService) exportToCSV(logs []models.VerificationLog) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"ID",
		"Session ID",
		"Application Name",
		"Captcha Type",
		"Status",
		"IP Address",
		"Risk Score",
		"Risk Level",
		"Duration (ms)",
		"Created At",
		"User Agent",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, log := range logs {
		applicationName := ""
		if log.Application.Name != "" {
			applicationName = log.Application.Name
		}

		row := []string{
			fmt.Sprintf("%d", log.ID),
			log.SessionID,
			applicationName,
			log.CaptchaType,
			log.Status,
			log.IPAddress,
			fmt.Sprintf("%.2f", log.RiskScore),
			calculateRiskLevelFromScore(log.RiskScore),
			fmt.Sprintf("%d", log.Duration),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.UserAgent,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *LogService) DeleteOldLogs(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	result := database.DB.Where("created_at < ?", cutoffDate).Delete(&models.VerificationLog{})
	return result.RowsAffected, result.Error
}

func (s *LogService) GetLogsBySessionID(sessionID string) ([]models.VerificationLog, error) {
	var logs []models.VerificationLog
	err := database.DB.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *LogService) GetLogCountByStatus(status string) (int64, error) {
	var count int64
	err := database.DB.Model(&models.VerificationLog{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

func (s *LogService) GetLogCountByDateRange(start, end time.Time) (int64, error) {
	var count int64
	err := database.DB.Model(&models.VerificationLog{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&count).Error
	return count, err
}
