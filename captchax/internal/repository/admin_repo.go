package repository

import (
	"context"
	"database/sql"
	"fmt"

	"captchax/internal/model"
)

type AdminRepo struct {
	db *sql.DB
}

func NewAdminRepo(db *sql.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) Create(ctx context.Context, admin *model.Admin) (int64, error) {
	query := `
		INSERT INTO admins (username, password_hash, role, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, admin.Username, admin.PasswordHash, admin.Role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create admin: %w", err)
	}
	return id, nil
}

func (r *AdminRepo) GetByID(ctx context.Context, id int64) (*model.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at FROM admins WHERE id = $1`
	admin := &model.Admin{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at FROM admins WHERE username = $1`
	admin := &model.Admin{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin by username: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) List(ctx context.Context, page, pageSize int) ([]*model.Admin, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `
		SELECT id, username, password_hash, role, created_at 
		FROM admins 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	var admins []*model.Admin
	for rows.Next() {
		admin := &model.Admin{}
		err := rows.Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.Role, &admin.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}
		admins = append(admins, admin)
	}
	return admins, nil
}

func (r *AdminRepo) Update(ctx context.Context, id int64, passwordHash, role string) error {
	if passwordHash != "" && role != "" {
		query := `UPDATE admins SET password_hash = $1, role = $2 WHERE id = $3`
		_, err := r.db.ExecContext(ctx, query, passwordHash, role, id)
		if err != nil {
			return fmt.Errorf("failed to update admin: %w", err)
		}
	} else if passwordHash != "" {
		query := `UPDATE admins SET password_hash = $1 WHERE id = $2`
		_, err := r.db.ExecContext(ctx, query, passwordHash, id)
		if err != nil {
			return fmt.Errorf("failed to update admin password: %w", err)
		}
	} else if role != "" {
		query := `UPDATE admins SET role = $1 WHERE id = $2`
		_, err := r.db.ExecContext(ctx, query, role, id)
		if err != nil {
			return fmt.Errorf("failed to update admin role: %w", err)
		}
	}
	return nil
}

func (r *AdminRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM admins WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("admin not found")
	}
	return nil
}

func (r *AdminRepo) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM admins`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count admins: %w", err)
	}
	return count, nil
}

func (r *AdminRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM admins WHERE username = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	return exists, nil
}
