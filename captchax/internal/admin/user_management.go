package admin

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"captchax/internal/model"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserManagementHandlers struct {
	db *sql.DB
}

func NewUserManagementHandlers(db *sql.DB) *UserManagementHandlers {
	return &UserManagementHandlers{db: db}
}

func (h *UserManagementHandlers) GetUsers(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	role := c.Query("role")
	status := c.Query("status")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var countQuery string
	var listQuery string
	var args []interface{}

	countQuery = `SELECT COUNT(*) FROM admins WHERE 1=1`
	listQuery = `SELECT id, username, email, nickname, phone, avatar, role, status, department, notes, last_login_at, last_login_ip, login_count, created_at, updated_at FROM admins WHERE 1=1`

	if search != "" {
		searchPattern := "%" + search + "%"
		countQuery += ` AND (username LIKE $1 OR email LIKE $1 OR nickname LIKE $1)`
		listQuery += ` AND (username LIKE $1 OR email LIKE $1 OR nickname LIKE $1)`
		args = append(args, searchPattern)
	}

	if role != "" {
		argIdx := len(args) + 1
		countQuery += ` AND role = $` + strconv.Itoa(argIdx)
		listQuery += ` AND role = $` + strconv.Itoa(argIdx)
		args = append(args, role)
	}

	if status != "" {
		argIdx := len(args) + 1
		countQuery += ` AND status = $` + strconv.Itoa(argIdx)
		listQuery += ` AND status = $` + strconv.Itoa(argIdx)
		args = append(args, status)
	}

	var total int64
	err := h.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		response.InternalError(c, "failed to count users")
		return
	}

	listQuery += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	listArgs := append(args, pageSize, offset)

	rows, err := h.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		response.InternalError(c, "failed to list users")
		return
	}
	defer rows.Close()

	users := make([]*model.AdminDTO, 0)
	for rows.Next() {
		var admin model.Admin
		err := rows.Scan(
			&admin.ID, &admin.Username, &admin.Email, &admin.Nickname, &admin.Phone,
			&admin.Avatar, &admin.Role, &admin.Status, &admin.Department, &admin.Notes,
			&admin.LastLoginAt, &admin.LastLoginIP, &admin.LoginCount, &admin.CreatedAt, &admin.UpdatedAt,
		)
		if err != nil {
			response.InternalError(c, "failed to scan user")
			return
		}
		users = append(users, admin.ToDTO())
	}

	if users == nil {
		users = make([]*model.AdminDTO, 0)
	}

	response.Success(c, gin.H{
		"items":       users,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *UserManagementHandlers) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	ctx := c.Request.Context()
	query := `SELECT id, username, email, nickname, phone, avatar, role, status, department, notes, last_login_at, last_login_ip, login_count, created_at, updated_at FROM admins WHERE id = $1`

	var admin model.Admin
	err = h.db.QueryRowContext(ctx, query, id).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.Nickname, &admin.Phone,
		&admin.Avatar, &admin.Role, &admin.Status, &admin.Department, &admin.Notes,
		&admin.LastLoginAt, &admin.LastLoginIP, &admin.LoginCount, &admin.CreatedAt, &admin.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		response.NotFound(c, "user not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to get user")
		return
	}

	response.Success(c, admin.ToDTO())
}

func (h *UserManagementHandlers) CreateUser(c *gin.Context) {
	var req model.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	var exists bool
	err := h.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM admins WHERE username = $1)`, req.Username).Scan(&exists)
	if err != nil {
		response.InternalError(c, "failed to check username")
		return
	}
	if exists {
		response.Error(c, http.StatusConflict, "username already exists")
		return
	}

	if req.Email != "" {
		err = h.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM admins WHERE email = $1)`, req.Email).Scan(&exists)
		if err != nil {
			response.InternalError(c, "failed to check email")
			return
		}
		if exists {
			response.Error(c, http.StatusConflict, "email already exists")
			return
		}
	}

	admin := &model.Admin{
		Username:   req.Username,
		Email:      req.Email,
		Nickname:   req.Nickname,
		Phone:      req.Phone,
		Role:       req.Role,
		Status:     1,
		Department: req.Department,
		Notes:      req.Notes,
	}
	if err := admin.SetPassword(req.Password); err != nil {
		response.InternalError(c, "failed to hash password")
		return
	}

	query := `INSERT INTO admins (username, password_hash, email, nickname, phone, role, status, department, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()) RETURNING id`

	var newID int64
	err = h.db.QueryRowContext(ctx, query, admin.Username, admin.PasswordHash, admin.Email, admin.Nickname, admin.Phone, admin.Role, admin.Status, admin.Department, admin.Notes).Scan(&newID)
	if err != nil {
		response.InternalError(c, "failed to create user")
		return
	}

	h.logAudit(ctx, 0, c.GetString("username"), "create_user", "Created user: "+admin.Username, c.ClientIP(), c.Request.UserAgent())

	response.Success(c, gin.H{"id": newID})
}

func (h *UserManagementHandlers) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req model.UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	var current model.Admin
	err = h.db.QueryRowContext(ctx, `SELECT id, username, email FROM admins WHERE id = $1`, id).Scan(&current.ID, &current.Username, &current.Email)
	if err == sql.ErrNoRows {
		response.NotFound(c, "user not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to get user")
		return
	}

	if req.Email != "" && req.Email != current.Email {
		var exists bool
		err = h.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM admins WHERE email = $1 AND id != $2)`, req.Email, id).Scan(&exists)
		if err != nil {
			response.InternalError(c, "failed to check email")
			return
		}
		if exists {
			response.Error(c, http.StatusConflict, "email already exists")
			return
		}
	}

	setClauses := ""
	args := make([]interface{}, 0)
	argIdx := 1

	if req.Email != "" {
		setClauses += "email = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, req.Email)
		argIdx++
	}
	if req.Nickname != "" {
		setClauses += "nickname = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, req.Nickname)
		argIdx++
	}
	if req.Phone != "" {
		setClauses += "phone = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, req.Phone)
		argIdx++
	}
	if req.Status != nil {
		setClauses += "status = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Department != "" {
		setClauses += "department = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, req.Department)
		argIdx++
	}
	if req.Notes != "" {
		setClauses += "notes = $" + strconv.Itoa(argIdx) + ", "
		args = append(args, req.Notes)
		argIdx++
	}

	if len(args) == 0 {
		response.BadRequest(c, "no fields to update")
		return
	}

	setClauses += "updated_at = NOW()"
	query := `UPDATE admins SET ` + setClauses + ` WHERE id = $` + strconv.Itoa(argIdx)
	args = append(args, id)

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		response.InternalError(c, "failed to update user")
		return
	}

	h.logAudit(ctx, 0, c.GetString("username"), "update_user", "Updated user ID: "+idStr, c.ClientIP(), c.Request.UserAgent())

	response.SuccessWithMessage(c, "user updated successfully", nil)
}

func (h *UserManagementHandlers) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	ctx := c.Request.Context()

	var username string
	err = h.db.QueryRowContext(ctx, `SELECT username FROM admins WHERE id = $1`, id).Scan(&username)
	if err == sql.ErrNoRows {
		response.NotFound(c, "user not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to get user")
		return
	}

	_, err = h.db.ExecContext(ctx, `DELETE FROM admins WHERE id = $1`, id)
	if err != nil {
		response.InternalError(c, "failed to delete user")
		return
	}

	h.logAudit(ctx, 0, c.GetString("username"), "delete_user", "Deleted user: "+username, c.ClientIP(), c.Request.UserAgent())

	response.SuccessWithMessage(c, "user deleted successfully", nil)
}

func (h *UserManagementHandlers) UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	_, err = h.db.ExecContext(ctx, `UPDATE admins SET role = $1, updated_at = NOW() WHERE id = $2`, req.Role, id)
	if err != nil {
		response.InternalError(c, "failed to update user role")
		return
	}

	h.logAudit(ctx, 0, c.GetString("username"), "update_role", "Updated role for user ID "+idStr+" to "+req.Role, c.ClientIP(), c.Request.UserAgent())

	response.SuccessWithMessage(c, "role updated successfully", nil)
}

func (h *UserManagementHandlers) UpdateUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	_, err = h.db.ExecContext(ctx, `UPDATE admins SET status = $1, updated_at = NOW() WHERE id = $2`, req.Status, id)
	if err != nil {
		response.InternalError(c, "failed to update user status")
		return
	}

	statusLabel := "disabled"
	if req.Status == 1 {
		statusLabel = "enabled"
	}
	h.logAudit(ctx, 0, c.GetString("username"), "update_status", "Updated status for user ID "+idStr+" to "+statusLabel, c.ClientIP(), c.Request.UserAgent())

	response.SuccessWithMessage(c, "status updated successfully", nil)
}

func (h *UserManagementHandlers) ShowUsersPage(c *gin.Context) {
	c.HTML(http.StatusOK, "users.html", gin.H{
		"title": "CaptchaX User Management",
	})
}

func (h *UserManagementHandlers) logAudit(ctx context.Context, userID uint, username, action, detail, ip, userAgent string) {
	if username == "" {
		username = "system"
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	query := `INSERT INTO audit_logs (user_id, username, action, detail, ip_address, user_agent, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, _ = h.db.ExecContext(ctx, query, userID, username, action, detail, ip, userAgent, time.Now())
}