package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"captchax/internal/model"
	"captchax/internal/repository"

	"github.com/gin-gonic/gin"
)

type RBACService struct {
	roleRepo       *repository.RoleRepo
	permRepo       *repository.PermissionRepo
	adminRoleRepo  *repository.AdminRoleRepo
	adminRepo      *repository.AdminRepo
}

func NewRBACService(
	roleRepo *repository.RoleRepo,
	permRepo *repository.PermissionRepo,
	adminRoleRepo *repository.AdminRoleRepo,
	adminRepo *repository.AdminRepo,
) *RBACService {
	return &RBACService{
		roleRepo:      roleRepo,
		permRepo:      permRepo,
		adminRoleRepo: adminRoleRepo,
		adminRepo:     adminRepo,
	}
}

func (s *RBACService) ListRoles(ctx context.Context, filter *model.RoleFilter) ([]*model.Role, int64, error) {
	roles, err := s.roleRepo.ListRoles(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.roleRepo.CountRoles(ctx)
	if err != nil {
		return nil, 0, err
	}

	for _, role := range roles {
		permissions, err := s.roleRepo.GetPermissionsByRoleID(ctx, int64(role.ID))
		if err != nil {
			return nil, 0, err
		}
		perms := make([]model.Permission, len(permissions))
		for i, p := range permissions {
			perms[i] = *p
		}
		role.Permissions = perms
	}

	return roles, count, nil
}

func (s *RBACService) GetRole(ctx context.Context, id int64) (*model.Role, error) {
	return s.roleRepo.GetRoleWithPermissions(ctx, id)
}

func (s *RBACService) CreateRole(ctx context.Context, req *model.CreateRoleRequest) (*model.Role, error) {
	role := &model.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
	}

	id, err := s.roleRepo.CreateRole(ctx, role)
	if err != nil {
		return nil, err
	}
	role.ID = uint(id)

	if len(req.PermissionIDs) > 0 {
		if err := s.roleRepo.SetRolePermissions(ctx, int64(role.ID), req.PermissionIDs); err != nil {
			return nil, err
		}
	}

	return s.roleRepo.GetRoleWithPermissions(ctx, int64(role.ID))
}

func (s *RBACService) UpdateRole(ctx context.Context, id int64, req *model.UpdateRoleRequest) (*model.Role, error) {
	existingRole, err := s.roleRepo.GetRoleByID(ctx, id)
	if err != nil || existingRole == nil {
		return nil, fmt.Errorf("role not found")
	}

	if existingRole.IsSystem {
		return nil, fmt.Errorf("cannot modify system role")
	}

	existingRole.Name = req.Name
	existingRole.Description = req.Description

	if err := s.roleRepo.UpdateRole(ctx, id, existingRole); err != nil {
		return nil, err
	}

	if req.PermissionIDs != nil {
		if err := s.roleRepo.SetRolePermissions(ctx, id, req.PermissionIDs); err != nil {
			return nil, err
		}
	}

	return s.roleRepo.GetRoleWithPermissions(ctx, id)
}

func (s *RBACService) DeleteRole(ctx context.Context, id int64) error {
	existingRole, err := s.roleRepo.GetRoleByID(ctx, id)
	if err != nil || existingRole == nil {
		return fmt.Errorf("role not found")
	}

	if existingRole.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	return s.roleRepo.DeleteRole(ctx, id)
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	return s.permRepo.ListPermissions(ctx)
}

func (s *RBACService) ListAdmins(ctx context.Context, filter *model.AdminFilter) ([]*model.Admin, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	offset := (filter.Page - 1) * filter.PageSize

	query := `
		SELECT id, username, email, nickname, phone, avatar, role, status, department,
			   notes, last_login_at, last_login_ip, login_count, created_at, updated_at
		FROM admins
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filter.Username != "" {
		argCount++
		query += fmt.Sprintf(" AND username LIKE $%d", argCount)
		args = append(args, "%"+filter.Username+"%")
	}
	if filter.Email != "" {
		argCount++
		query += fmt.Sprintf(" AND email LIKE $%d", argCount)
		args = append(args, "%"+filter.Email+"%")
	}
	if filter.Role != "" {
		argCount++
		query += fmt.Sprintf(" AND role = $%d", argCount)
		args = append(args, filter.Role)
	}
	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	whereIdx := strings.Index(query, "WHERE")
	countQuery := "SELECT COUNT(*) FROM admins " + query[whereIdx+5:]
	var total int64
	if err := s.adminRepo.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	argCount++
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argCount)
	args = append(args, filter.PageSize)
	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := s.adminRepo.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var admins []*model.Admin
	for rows.Next() {
		admin := &model.Admin{}
		var lastLoginAt sql.NullTime
		var lastLoginIP sql.NullString
		var email, nickname, phone, avatar, department, notes sql.NullString

		err := rows.Scan(
			&admin.ID, &admin.Username, &email, &nickname, &phone, &avatar,
			&admin.Role, &admin.Status, &department,
			&notes, &lastLoginAt, &lastLoginIP, &admin.LoginCount, &admin.CreatedAt, &admin.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		admin.Email = email.String
		admin.Nickname = nickname.String
		admin.Phone = phone.String
		admin.Avatar = avatar.String
		admin.Department = department.String
		admin.Notes = notes.String
		if lastLoginAt.Valid {
			admin.LastLoginAt = &lastLoginAt.Time
		}
		if lastLoginIP.Valid {
			admin.LastLoginIP = lastLoginIP.String
		}

		roles, err := s.adminRoleRepo.GetAdminRoles(ctx, int64(admin.ID))
		if err == nil {
			adminRoles := make([]model.Role, len(roles))
			for i, r := range roles {
				adminRoles[i] = *r
			}
			admin.Roles = adminRoles
		}

		admins = append(admins, admin)
	}

	return admins, total, nil
}

func (s *RBACService) GetAdmin(ctx context.Context, id int64) (*model.Admin, error) {
	admin, err := s.adminRepo.GetByID(ctx, id)
	if err != nil || admin == nil {
		return nil, fmt.Errorf("admin not found")
	}

	roles, err := s.adminRoleRepo.GetAdminRoles(ctx, id)
	if err == nil {
		for _, r := range roles {
			admin.Roles = append(admin.Roles, *r)
		}
	}

	return admin, nil
}

func (s *RBACService) CreateAdmin(ctx context.Context, req *model.CreateAdminRequest) (*model.Admin, error) {
	exists, err := s.adminRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("username already exists")
	}

	admin := &model.Admin{
		Username:   req.Username,
		Email:      req.Email,
		Nickname:   req.Nickname,
		Phone:      req.Phone,
		Role:       req.Role,
		Department: req.Department,
		Notes:      req.Notes,
		Status:     1,
	}

	if err := admin.SetPassword(req.Password); err != nil {
		return nil, err
	}

	id, err := s.adminRepo.Create(ctx, admin)
	if err != nil {
		return nil, err
	}
	admin.ID = uint(id)

	role, err := s.roleRepo.GetRoleByCode(ctx, req.Role)
	if err == nil && role != nil {
		if err := s.adminRoleRepo.AssignRole(ctx, int64(admin.ID), int64(role.ID)); err != nil {
			return nil, err
		}
		admin.Roles = append(admin.Roles, *role)
	}

	return admin, nil
}

func (s *RBACService) UpdateAdmin(ctx context.Context, id int64, req *model.UpdateAdminRequest) (*model.Admin, error) {
	admin, err := s.adminRepo.GetByID(ctx, id)
	if err != nil || admin == nil {
		return nil, fmt.Errorf("admin not found")
	}

	if req.Email != "" {
		admin.Email = req.Email
	}
	if req.Nickname != "" {
		admin.Nickname = req.Nickname
	}
	if req.Phone != "" {
		admin.Phone = req.Phone
	}
	if req.Status != nil {
		admin.Status = *req.Status
	}
	if req.Department != "" {
		admin.Department = req.Department
	}
	if req.Notes != "" {
		admin.Notes = req.Notes
	}

	if err := s.adminRepo.Update(ctx, id, "", ""); err != nil {
		return nil, err
	}

	return s.GetAdmin(ctx, id)
}

func (s *RBACService) DeleteAdmin(ctx context.Context, id int64) error {
	admin, err := s.adminRepo.GetByID(ctx, id)
	if err != nil || admin == nil {
		return fmt.Errorf("admin not found")
	}

	if admin.Role == "super_admin" {
		return fmt.Errorf("cannot delete super admin")
	}

	return s.adminRepo.Delete(ctx, id)
}

func (s *RBACService) UpdateAdminRoles(ctx context.Context, adminID int64, req *model.AssignRolesRequest) (*model.Admin, error) {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil || admin == nil {
		return nil, fmt.Errorf("admin not found")
	}

	if err := s.adminRoleRepo.SetAdminRoles(ctx, adminID, req.RoleIDs); err != nil {
		return nil, err
	}

	return s.GetAdmin(ctx, adminID)
}

func (s *RBACService) CheckPermission(c *gin.Context, permissionCode string) bool {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return false
	}

	hasPermission, err := s.adminRoleRepo.HasPermission(context.Background(), int64(adminID.(uint)), permissionCode)
	if err != nil {
		return false
	}

	return hasPermission
}

func (s *RBACService) CheckRole(c *gin.Context, roleCode string) bool {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return false
	}

	hasRole, err := s.adminRoleRepo.HasRole(context.Background(), int64(adminID.(uint)), roleCode)
	if err != nil {
		return false
	}

	return hasRole
}

func (s *RBACService) GetAdminPermissions(ctx context.Context, adminID int64) ([]*model.Permission, error) {
	return s.adminRoleRepo.GetAdminPermissions(ctx, adminID)
}

func (s *RBACService) LogOperation(ctx context.Context, adminID uint, username, action, resourceType, resourceID, details, ip, userAgent string) error {
	return nil
}
