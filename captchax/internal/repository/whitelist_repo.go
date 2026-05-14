package repository

import (
	"context"
	"database/sql"
	"fmt"

	"captchax/internal/model"
)

type WhitelistRepo struct {
	db *sql.DB
}

func NewWhitelistRepo(db *sql.DB) *WhitelistRepo {
	return &WhitelistRepo{db: db}
}

func (r *WhitelistRepo) Create(ctx context.Context, w *model.Whitelist) (int64, error) {
	query := `
		INSERT INTO whitelist (ip, domain, reason, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (ip, domain) DO NOTHING
		RETURNING id
	`
	var domain, reason interface{}
	if w.Domain.Valid {
		domain = w.Domain.String
	}
	if w.Reason.Valid {
		reason = w.Reason.String
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query, w.IP, domain, reason).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("whitelist entry already exists")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to create whitelist: %w", err)
	}
	return id, nil
}

func (r *WhitelistRepo) GetByID(ctx context.Context, id int64) (*model.Whitelist, error) {
	query := `SELECT id, ip, domain, reason, created_at FROM whitelist WHERE id = $1`
	w := &model.Whitelist{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&w.ID, &w.IP, &w.Domain, &w.Reason, &w.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist: %w", err)
	}
	return w, nil
}

func (r *WhitelistRepo) GetByIP(ctx context.Context, ip string) (*model.Whitelist, error) {
	query := `SELECT id, ip, domain, reason, created_at FROM whitelist WHERE ip = $1`
	w := &model.Whitelist{}
	err := r.db.QueryRowContext(ctx, query, ip).Scan(
		&w.ID, &w.IP, &w.Domain, &w.Reason, &w.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist by IP: %w", err)
	}
	return w, nil
}

func (r *WhitelistRepo) IsWhitelisted(ctx context.Context, ip string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM whitelist WHERE ip = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, ip).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check whitelist: %w", err)
	}
	return exists, nil
}

func (r *WhitelistRepo) List(ctx context.Context, filter *model.WhitelistFilter) ([]*model.Whitelist, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.IP != "" {
		conditions = append(conditions, fmt.Sprintf("ip LIKE $%d", argIdx))
		args = append(args, "%"+filter.IP+"%")
		argIdx++
	}
	if filter.Domain != "" {
		conditions = append(conditions, fmt.Sprintf("domain LIKE $%d", argIdx))
		args = append(args, "%"+filter.Domain+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	query := fmt.Sprintf(`
		SELECT id, ip, domain, reason, created_at
		FROM whitelist %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.Limit(), filter.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list whitelist: %w", err)
	}
	defer rows.Close()

	var list []*model.Whitelist
	for rows.Next() {
		w := &model.Whitelist{}
		err := rows.Scan(&w.ID, &w.IP, &w.Domain, &w.Reason, &w.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan whitelist: %w", err)
		}
		list = append(list, w)
	}
	return list, nil
}

func (r *WhitelistRepo) Update(ctx context.Context, id int64, domain, reason string) error {
	query := `UPDATE whitelist SET domain = $1, reason = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, domain, reason, id)
	if err != nil {
		return fmt.Errorf("failed to update whitelist: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("whitelist entry not found")
	}
	return nil
}

func (r *WhitelistRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM whitelist WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete whitelist: %w", err)
	}
	return nil
}

func (r *WhitelistRepo) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM whitelist`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count whitelist: %w", err)
	}
	return count, nil
}
