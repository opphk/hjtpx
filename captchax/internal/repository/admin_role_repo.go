package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"captchax/internal/model"
)

type RoleRepo struct {
	db *sql.DB
}

func NewRoleRepo(db *sql.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) CreateRole(ctx context.Context, role *model.Role) (int64, error) {
	query := `
		INSERT INTO roles (code, name, description, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, role.Code, role.Name, role.Description, role.IsSystem).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return 0, fmt.Errorf("role code already exists")
		}
		return 0, fmt.Errorf("failed to create role: %w", err)
	}
	return id, nil
}

func (r *RoleRepo) GetRoleByID(ctx context.Context, id int64) (*model.Role, error) {
	query := `
		SELECT id, code, name, description, is_system, created_at, updated_at
		FROM roles WHERE id = $1
	`
	role := &model.Role{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return role, nil
}

func (r *RoleRepo) GetRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	query := `
		SELECT id, code, name, description, is_system, created_at, updated_at
		FROM roles WHERE code = $1
	`
	role := &model.Role{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by code: %w", err)
	}
	return role, nil
}

func (r *RoleRepo) ListRoles(ctx context.Context, filter *model.RoleFilter) ([]*model.Role, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	query := `
		SELECT id, code, name, description, is_system, created_at, updated_at
		FROM roles
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filter.Code != "" {
		argCount++
		query += fmt.Sprintf(" AND code LIKE $%d", argCount)
		args = append(args, "%"+filter.Code+"%")
	}
	if filter.IsSystem != nil {
		argCount++
		query += fmt.Sprintf(" AND is_system = $%d", argCount)
		args = append(args, *filter.IsSystem)
	}

	argCount++
	query += fmt.Sprintf(" ORDER BY is_system DESC, created_at DESC LIMIT $%d", argCount)
	args = append(args, filter.PageSize)
	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		role := &model.Role{}
		err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *RoleRepo) UpdateRole(ctx context.Context, id int64, role *model.Role) error {
	query := `
		UPDATE roles SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND is_system = FALSE
	`
	result, err := r.db.ExecContext(ctx, query, role.Name, role.Description, id)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("role not found or is a system role")
	}
	return nil
}

func (r *RoleRepo) DeleteRole(ctx context.Context, id int64) error {
	query := `DELETE FROM roles WHERE id = $1 AND is_system = FALSE`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("role not found or is a system role")
	}
	return nil
}

func (r *RoleRepo) CountRoles(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM roles`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count roles: %w", err)
	}
	return count, nil
}

func (r *RoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID int64) ([]*model.Permission, error) {
	query := `
		SELECT p.id, p.code, p.name, p.description, p.category, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.category, p.code
	`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		perm := &model.Permission{}
		err := rows.Scan(&perm.ID, &perm.Code, &perm.Name, &perm.Description, &perm.Category, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

func (r *RoleRepo) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return fmt.Errorf("failed to delete existing permissions: %w", err)
	}

	if len(permissionIDs) > 0 {
		valueStrings := make([]string, 0, len(permissionIDs))
		valueArgs := make([]interface{}, 0, len(permissionIDs)*2)
		for i, permID := range permissionIDs {
			valueStrings = append(valueStrings, fmt.Sprintf("($1, $%d, NOW())", i+2))
			valueArgs = append(valueArgs, roleID, permID)
		}
		query := `INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES ` + strings.Join(valueStrings, ", ")
		_, err = tx.ExecContext(ctx, query, valueArgs...)
		if err != nil {
			return fmt.Errorf("failed to add permissions: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *RoleRepo) GetRoleWithPermissions(ctx context.Context, id int64) (*model.Role, error) {
	role, err := r.GetRoleByID(ctx, id)
	if err != nil || role == nil {
		return role, err
	}

	permissions, err := r.GetPermissionsByRoleID(ctx, id)
	if err != nil {
		return nil, err
	}
	perms := make([]model.Permission, len(permissions))
	for i, p := range permissions {
		perms[i] = *p
	}
	role.Permissions = perms
	return role, nil
}

type PermissionRepo struct {
	db *sql.DB
}

func NewPermissionRepo(db *sql.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	query := `
		SELECT id, code, name, description, category, created_at, updated_at
		FROM permissions
		ORDER BY category, code
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		perm := &model.Permission{}
		err := rows.Scan(&perm.ID, &perm.Code, &perm.Name, &perm.Description, &perm.Category, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

func (r *PermissionRepo) GetPermissionByCode(ctx context.Context, code string) (*model.Permission, error) {
	query := `
		SELECT id, code, name, description, category, created_at, updated_at
		FROM permissions WHERE code = $1
	`
	perm := &model.Permission{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&perm.ID, &perm.Code, &perm.Name, &perm.Description, &perm.Category, &perm.CreatedAt, &perm.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return perm, nil
}

func (r *PermissionRepo) GetPermissionsByCodes(ctx context.Context, codes []string) ([]*model.Permission, error) {
	if len(codes) == 0 {
		return []*model.Permission{}, nil
	}

	placeholders := make([]string, len(codes))
	args := make([]interface{}, len(codes))
	for i, code := range codes {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = code
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, description, category, created_at, updated_at
		FROM permissions
		WHERE code IN (%s)
		ORDER BY category, code
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		perm := &model.Permission{}
		err := rows.Scan(&perm.ID, &perm.Code, &perm.Name, &perm.Description, &perm.Category, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

type AdminRoleRepo struct {
	db *sql.DB
}

func NewAdminRoleRepo(db *sql.DB) *AdminRoleRepo {
	return &AdminRoleRepo{db: db}
}

func (r *AdminRoleRepo) AssignRole(ctx context.Context, adminID, roleID int64) error {
	query := `INSERT INTO admin_roles (admin_id, role_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, adminID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

func (r *AdminRoleRepo) RemoveRole(ctx context.Context, adminID, roleID int64) error {
	query := `DELETE FROM admin_roles WHERE admin_id = $1 AND role_id = $2`
	_, err := r.db.ExecContext(ctx, query, adminID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	return nil
}

func (r *AdminRoleRepo) SetAdminRoles(ctx context.Context, adminID int64, roleIDs []uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM admin_roles WHERE admin_id = $1`, adminID)
	if err != nil {
		return fmt.Errorf("failed to delete existing roles: %w", err)
	}

	if len(roleIDs) > 0 {
		valueStrings := make([]string, 0, len(roleIDs))
		valueArgs := make([]interface{}, 0, len(roleIDs)*2)
		for i, roleID := range roleIDs {
			valueStrings = append(valueStrings, fmt.Sprintf("($1, $%d, NOW())", i+2))
			valueArgs = append(valueArgs, adminID, roleID)
		}
		query := `INSERT INTO admin_roles (admin_id, role_id, created_at) VALUES ` + strings.Join(valueStrings, ", ")
		_, err = tx.ExecContext(ctx, query, valueArgs...)
		if err != nil {
			return fmt.Errorf("failed to assign roles: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *AdminRoleRepo) GetAdminRoles(ctx context.Context, adminID int64) ([]*model.Role, error) {
	query := `
		SELECT r.id, r.code, r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN admin_roles ar ON r.id = ar.role_id
		WHERE ar.admin_id = $1
		ORDER BY r.is_system DESC, r.name
	`
	rows, err := r.db.QueryContext(ctx, query, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin roles: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		role := &model.Role{}
		err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *AdminRoleRepo) GetAdminPermissions(ctx context.Context, adminID int64) ([]*model.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.code, p.name, p.description, p.category, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN admin_roles ar ON rp.role_id = ar.role_id
		WHERE ar.admin_id = $1
		ORDER BY p.category, p.code
	`
	rows, err := r.db.QueryContext(ctx, query, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		perm := &model.Permission{}
		err := rows.Scan(&perm.ID, &perm.Code, &perm.Name, &perm.Description, &perm.Category, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

func (r *AdminRoleRepo) HasPermission(ctx context.Context, adminID int64, permissionCode string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM permissions p
			INNER JOIN role_permissions rp ON p.id = rp.permission_id
			INNER JOIN admin_roles ar ON rp.role_id = ar.role_id
			WHERE ar.admin_id = $1 AND p.code = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, adminID, permissionCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return exists, nil
}

func (r *AdminRoleRepo) HasRole(ctx context.Context, adminID int64, roleCode string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM roles r
			INNER JOIN admin_roles ar ON r.id = ar.role_id
			WHERE ar.admin_id = $1 AND r.code = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, adminID, roleCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}
	return exists, nil
}
