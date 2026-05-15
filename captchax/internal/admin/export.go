package admin

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"captchax/internal/model"
	"captchax/internal/repository"

	"github.com/gin-gonic/gin"
)

type ExportService struct {
	captchaRepo *repository.CaptchaRepo
	db          *sql.DB
}

func NewExportService(captchaRepo *repository.CaptchaRepo, db *sql.DB) *ExportService {
	return &ExportService{
		captchaRepo: captchaRepo,
		db:          db,
	}
}

type ExportRequest struct {
	Format     string
	Type       string
	StartDate  string
	EndDate    string
	CaptchaType string
	Page       int
	PageSize   int
}

type ExportResult struct {
	Data       interface{}
	TotalCount int64
	FileName   string
	MimeType   string
}

type CaptchaExportRow struct {
	ID         int64  `json:"id" csv:"ID"`
	Type       string `json:"captcha_type" csv:"Type"`
	ClientID   string `json:"client_id" csv:"Client ID"`
	IP         string `json:"ip" csv:"IP"`
	UserAgent  string `json:"user_agent" csv:"User Agent"`
	Result     string `json:"result" csv:"Result"`
	Duration   int    `json:"duration" csv:"Duration (ms)"`
	RiskScore  int    `json:"risk_score" csv:"Risk Score"`
	CreatedAt  string `json:"created_at" csv:"Created At"`
}

type StatsExportRow struct {
	Metric      string  `json:"metric" csv:"Metric"`
	Value       float64 `json:"value" csv:"Value"`
	Description string  `json:"description" csv:"Description"`
}

type LogExportRow struct {
	ID          int64  `json:"id" csv:"ID"`
	Level       string `json:"level" csv:"Level"`
	Message     string `json:"message" csv:"Message"`
	Source      string `json:"source" csv:"Source"`
	IP          string `json:"ip" csv:"IP"`
	AdminID     string `json:"admin_id" csv:"Admin ID"`
	Action      string `json:"action" csv:"Action"`
	Details     string `json:"details" csv:"Details"`
	CreatedAt   string `json:"created_at" csv:"Created At"`
}

func (s *ExportService) ExportCaptchas(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
		endDate = time.Now()
	}

	filter := &model.CaptchaLogFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Type:      req.CaptchaType,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 1000
	}
	if filter.PageSize > 10000 {
		filter.PageSize = 10000
	}

	logs, err := s.captchaRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch captcha logs: %w", err)
	}

	rows := make([]CaptchaExportRow, 0, len(logs))
	for _, log := range logs {
		row := CaptchaExportRow{
			ID:        log.ID,
			Type:      log.Type,
			ClientID:  log.ClientID,
			IP:        log.IP,
			Result:    boolToString(log.Result),
			Duration:  log.Duration,
			RiskScore: log.RiskScore,
			CreatedAt: log.CreatedAt.Format(time.RFC3339),
		}
		if log.UserAgent.Valid {
			row.UserAgent = log.UserAgent.String
		}
		rows = append(rows, row)
	}

	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM captcha_logs WHERE created_at >= $1 AND created_at <= $2`
	args := []interface{}{startDate, endDate}
	if req.CaptchaType != "" {
		countQuery += " AND captcha_type = $3"
		args = append(args, req.CaptchaType)
	}
	s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)

	prefix := "captchas"
	if req.CaptchaType != "" {
		prefix = req.CaptchaType + "_" + prefix
	}
	fileName := fmt.Sprintf("%s_%s_%s", prefix, req.StartDate, req.EndDate)

	return &ExportResult{
		Data:       rows,
		TotalCount: totalCount,
		FileName:   fileName,
		MimeType:   getMimeType(req.Format),
	}, nil
}

func (s *ExportService) ExportStats(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
		endDate = time.Now()
	}

	stats, err := s.captchaRepo.GetStats(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	rows := []StatsExportRow{
		{Metric: "Total Verifications", Value: float64(stats.TotalCount), Description: "Total number of captcha verifications"},
		{Metric: "Success Count", Value: float64(stats.SuccessCount), Description: "Number of successful verifications"},
		{Metric: "Fail Count", Value: float64(stats.FailCount), Description: "Number of failed verifications"},
		{Metric: "Success Rate", Value: stats.SuccessRate, Description: "Percentage of successful verifications"},
		{Metric: "Average Duration", Value: stats.AvgDuration, Description: "Average verification duration in ms"},
		{Metric: "Average Risk Score", Value: stats.AvgRiskScore, Description: "Average risk score (0-100)"},
	}

	for captchaType, count := range stats.ByType {
		rows = append(rows, StatsExportRow{
			Metric:      fmt.Sprintf("Type: %s", captchaType),
			Value:       float64(count),
			Description: fmt.Sprintf("Count for %s type", captchaType),
		})
	}

	fileName := fmt.Sprintf("stats_%s_%s", req.StartDate, req.EndDate)

	return &ExportResult{
		Data:       rows,
		TotalCount: int64(len(rows)),
		FileName:   fileName,
		MimeType:   getMimeType(req.Format),
	}, nil
}

func (s *ExportService) ExportLogs(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
		endDate = time.Now()
	}

	query := `
		SELECT id, COALESCE(level, 'info') as level, message, source, 
		       COALESCE(ip, '') as ip, COALESCE(admin_id, 0) as admin_id,
		       COALESCE(action, '') as action, COALESCE(details, '') as details,
		       created_at
		FROM admin_logs
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 1000
	}
	if pageSize > 10000 {
		pageSize = 10000
	}
	offset := (page - 1) * pageSize

	rows, err := s.db.QueryContext(ctx, query, startDate, endDate, pageSize, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			return &ExportResult{
				Data:       []LogExportRow{},
				TotalCount: 0,
				FileName:   fmt.Sprintf("logs_%s_%s", req.StartDate, req.EndDate),
				MimeType:   getMimeType(req.Format),
			}, nil
		}
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	defer rows.Close()

	var logRows []LogExportRow
	for rows.Next() {
		var row LogExportRow
		var adminID int64
		if err := rows.Scan(&row.ID, &row.Level, &row.Message, &row.Source, &row.IP, &adminID, &row.Action, &row.Details, &row.CreatedAt); err != nil {
			continue
		}
		row.AdminID = strconv.FormatInt(adminID, 10)
		logRows = append(logRows, row)
	}

	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM admin_logs WHERE created_at >= $1 AND created_at <= $2`
	s.db.QueryRowContext(ctx, countQuery, startDate, endDate).Scan(&totalCount)

	fileName := fmt.Sprintf("logs_%s_%s", req.StartDate, req.EndDate)

	return &ExportResult{
		Data:       logRows,
		TotalCount: totalCount,
		FileName:   fileName,
		MimeType:   getMimeType(req.Format),
	}, nil
}

func (s *ExportService) ExportCaptchasBatch(ctx context.Context, req *ExportRequest, writer io.Writer) error {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
		endDate = time.Now()
	}

	batchSize := 5000
	offset := 0

	if req.Format == "csv" {
		csvWriter := csv.NewWriter(writer)
		headers := []string{"ID", "Type", "Client ID", "IP", "User Agent", "Result", "Duration (ms)", "Risk Score", "Created At"}
		csvWriter.Write(headers)

		for {
			filter := &model.CaptchaLogFilter{
				StartDate: &startDate,
				EndDate:   &endDate,
				Type:      req.CaptchaType,
				Page:      1,
				PageSize:  batchSize,
			}
			filter.PageSize = batchSize

			logs, err := s.captchaRepo.List(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to fetch batch: %w", err)
			}

			if len(logs) == 0 {
				break
			}

			for _, log := range logs {
				ua := ""
				if log.UserAgent.Valid {
					ua = log.UserAgent.String
				}
				row := []string{
					strconv.FormatInt(log.ID, 10),
					log.Type,
					log.ClientID,
					log.IP,
					ua,
					boolToString(log.Result),
					strconv.Itoa(log.Duration),
					strconv.Itoa(log.RiskScore),
					log.CreatedAt.Format(time.RFC3339),
				}
				csvWriter.Write(row)
			}

			offset += len(logs)
			if len(logs) < batchSize {
				break
			}
		}

		csvWriter.Flush()
		return csvWriter.Error()
	}

	csvWriter := csv.NewWriter(writer)
	headers := []string{"ID", "Type", "Client ID", "IP", "User Agent", "Result", "Duration (ms)", "Risk Score", "Created At"}
	csvWriter.Write(headers)

	for {
		filter := &model.CaptchaLogFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
			Type:      req.CaptchaType,
			Page:      1,
			PageSize:  batchSize,
		}
		filter.PageSize = batchSize

		logs, err := s.captchaRepo.List(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to fetch batch: %w", err)
		}

		if len(logs) == 0 {
			break
		}

		for _, log := range logs {
			ua := ""
			if log.UserAgent.Valid {
				ua = log.UserAgent.String
			}
			row := []string{
				strconv.FormatInt(log.ID, 10),
				log.Type,
				log.ClientID,
				log.IP,
				ua,
				boolToString(log.Result),
				strconv.Itoa(log.Duration),
				strconv.Itoa(log.RiskScore),
				log.CreatedAt.Format(time.RFC3339),
			}
			csvWriter.Write(row)
		}

		offset += len(logs)
		if len(logs) < batchSize {
			break
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (s *ExportService) GetExportCount(ctx context.Context, exportType string, req *ExportRequest) (int64, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" && req.EndDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
		endDate = time.Now()
	}

	switch exportType {
	case "captchas":
		query := `SELECT COUNT(*) FROM captcha_logs WHERE created_at >= $1 AND created_at <= $2`
		args := []interface{}{startDate, endDate}
		if req.CaptchaType != "" {
			query += " AND captcha_type = $3"
			args = append(args, req.CaptchaType)
		}
		var count int64
		err = s.db.QueryRowContext(ctx, query, args...).Scan(&count)
		return count, err

	case "stats":
		return 1, nil

	case "logs":
		query := `SELECT COUNT(*) FROM admin_logs WHERE created_at >= $1 AND created_at <= $2`
		var count int64
		err = s.db.QueryRowContext(ctx, query, startDate, endDate).Scan(&count)
		return count, err

	default:
		return 0, fmt.Errorf("unknown export type: %s", exportType)
	}
}

func SerializeToFormat(data interface{}, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	case "csv":
		return convertToCSV(data)
	default:
		return json.MarshalIndent(data, "", "  ")
	}
}

func convertToCSV(data interface{}) ([]byte, error) {
	var rows [][]string

	switch v := data.(type) {
	case []CaptchaExportRow:
		rows = make([][]string, 0, len(v)+1)
		rows = append(rows, []string{"ID", "Type", "Client ID", "IP", "User Agent", "Result", "Duration (ms)", "Risk Score", "Created At"})
		for _, row := range v {
			rows = append(rows, []string{
				strconv.FormatInt(row.ID, 10),
				row.Type,
				row.ClientID,
				row.IP,
				row.UserAgent,
				row.Result,
				strconv.Itoa(row.Duration),
				strconv.Itoa(row.RiskScore),
				row.CreatedAt,
			})
		}
	case []StatsExportRow:
		rows = make([][]string, 0, len(v)+1)
		rows = append(rows, []string{"Metric", "Value", "Description"})
		for _, row := range v {
			rows = append(rows, []string{
				row.Metric,
				strconv.FormatFloat(row.Value, 'f', 2, 64),
				row.Description,
			})
		}
	case []LogExportRow:
		rows = make([][]string, 0, len(v)+1)
		rows = append(rows, []string{"ID", "Level", "Message", "Source", "IP", "Admin ID", "Action", "Details", "Created At"})
		for _, row := range v {
			rows = append(rows, []string{
				strconv.FormatInt(row.ID, 10),
				row.Level,
				row.Message,
				row.Source,
				row.IP,
				row.AdminID,
				row.Action,
				row.Details,
				row.CreatedAt,
			})
		}
	default:
		return nil, fmt.Errorf("unsupported data type for CSV export")
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.WriteAll(rows)
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func boolToString(b bool) string {
	if b {
		return "success"
	}
	return "fail"
}

func getMimeType(format string) string {
	switch format {
	case "csv":
		return "text/csv"
	case "json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func ParseExportRequest(c *gin.Context) *ExportRequest {
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "json" {
		format = "csv"
	}

	exportType := c.DefaultQuery("type", "captchas")
	if exportType != "captchas" && exportType != "stats" && exportType != "logs" {
		exportType = "captchas"
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "1000"))

	return &ExportRequest{
		Format:      format,
		Type:        exportType,
		StartDate:   c.Query("start_date"),
		EndDate:     c.Query("end_date"),
		CaptchaType: c.Query("captcha_type"),
		Page:        page,
		PageSize:    pageSize,
	}
}
