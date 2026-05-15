package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"captchax/internal/model"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuditHandlers struct {
	db *sql.DB
}

func NewAuditHandlers(db *sql.DB) *AuditHandlers {
	return &AuditHandlers{db: db}
}

func (h *AuditHandlers) GetAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")
	username := c.Query("username")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	listQuery := `SELECT id, user_id, username, action, detail, ip_address, user_agent, created_at FROM audit_logs WHERE 1=1`
	args := make([]interface{}, 0)

	if action != "" {
		argIdx := len(args) + 1
		countQuery += ` AND action = $` + strconv.Itoa(argIdx)
		listQuery += ` AND action = $` + strconv.Itoa(argIdx)
		args = append(args, action)
	}

	if username != "" {
		argIdx := len(args) + 1
		countQuery += ` AND username ILIKE $` + strconv.Itoa(argIdx)
		listQuery += ` AND username ILIKE $` + strconv.Itoa(argIdx)
		args = append(args, "%"+username+"%")
	}

	if startDate != "" {
		argIdx := len(args) + 1
		countQuery += ` AND created_at >= $` + strconv.Itoa(argIdx)
		listQuery += ` AND created_at >= $` + strconv.Itoa(argIdx)
		args = append(args, startDate)
	}

	if endDate != "" {
		argIdx := len(args) + 1
		countQuery += ` AND created_at <= $` + strconv.Itoa(argIdx)
		listQuery += ` AND created_at <= $` + strconv.Itoa(argIdx)
		args = append(args, endDate)
	}

	var total int64
	err := h.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		response.InternalError(c, "failed to count audit logs")
		return
	}

	listQuery += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	listArgs := append(args, pageSize, offset)

	rows, err := h.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		response.InternalError(c, "failed to list audit logs")
		return
	}
	defer rows.Close()

	logs := make([]*model.AuditLogDTO, 0)
	for rows.Next() {
		var log model.AuditLog
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action, &log.Detail,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			response.InternalError(c, "failed to scan audit log")
			return
		}
		logs = append(logs, log.ToDTO())
	}

	if logs == nil {
		logs = make([]*model.AuditLogDTO, 0)
	}

	response.Success(c, gin.H{
		"items":       logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *AuditHandlers) ExportAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	action := c.Query("action")
	username := c.Query("username")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := `SELECT id, user_id, username, action, detail, ip_address, user_agent, created_at FROM audit_logs WHERE 1=1`
	args := make([]interface{}, 0)

	if action != "" {
		argIdx := len(args) + 1
		query += ` AND action = $` + strconv.Itoa(argIdx)
		args = append(args, action)
	}

	if username != "" {
		argIdx := len(args) + 1
		query += ` AND username ILIKE $` + strconv.Itoa(argIdx)
		args = append(args, "%"+username+"%")
	}

	if startDate != "" {
		argIdx := len(args) + 1
		query += ` AND created_at >= $` + strconv.Itoa(argIdx)
		args = append(args, startDate)
	}

	if endDate != "" {
		argIdx := len(args) + 1
		query += ` AND created_at <= $` + strconv.Itoa(argIdx)
		args = append(args, endDate)
	}

	query += ` ORDER BY created_at DESC LIMIT 10000`

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		response.InternalError(c, "failed to export audit logs")
		return
	}
	defer rows.Close()

	var csvBuilder strings.Builder
	csvBuilder.WriteString("ID,User ID,Username,Action,Detail,IP Address,User Agent,Created At\n")

	for rows.Next() {
		var log model.AuditLog
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action, &log.Detail,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			continue
		}
		detail := strings.ReplaceAll(log.Detail, `"`, `""`)
		userAgent := strings.ReplaceAll(log.UserAgent, `"`, `""`)
		csvBuilder.WriteString(fmt.Sprintf(`%d,%d,"%s","%s","%s","%s","%s","%s"`+"\n",
			log.ID, log.UserID, log.Username, log.Action, detail, log.IPAddress, userAgent, log.CreatedAt.Format(time.RFC3339)))
	}

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, csvBuilder.String())
}

func (h *AuditHandlers) ShowAuditPage(c *gin.Context) {
	c.HTML(http.StatusOK, "audit.html", gin.H{
		"title": "CaptchaX Audit Logs",
	})
}

func (h *AuditHandlers) LogAction(ctx context.Context, userID uint, username, action, detail, ip, userAgent string) {
	if username == "" {
		username = "system"
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	query := `INSERT INTO audit_logs (user_id, username, action, detail, ip_address, user_agent, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`
	_, _ = h.db.ExecContext(ctx, query, userID, username, action, detail, ip, userAgent)
}