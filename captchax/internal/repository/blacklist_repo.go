package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"captchax/internal/model"
)

type BlacklistRepo struct {
	db *sql.DB
}

func NewBlacklistRepo(db *sql.DB) *BlacklistRepo {
	return &BlacklistRepo{db: db}
}

func (r *BlacklistRepo) Create(ctx context.Context, b *model.Blacklist) (int64, error) {
	query := `
		INSERT INTO blacklist (ip, reason, expire_at, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`
	var reason, expireAt interface{}
	if b.Reason.Valid {
		reason = b.Reason.String
	}
	if b.ExpireAt.Valid {
		expireAt = b.ExpireAt.Time
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query, b.IP, reason, expireAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create blacklist: %w", err)
	}
	return id, nil
}

func (r *BlacklistRepo) GetByID(ctx context.Context, id int64) (*model.Blacklist, error) {
	query := `SELECT id, ip, reason, expire_at, created_at FROM blacklist WHERE id = $1`
	b := &model.Blacklist{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.IP, &b.Reason, &b.ExpireAt, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklist: %w", err)
	}
	return b, nil
}

func (r *BlacklistRepo) GetByIP(ctx context.Context, ip string) (*model.Blacklist, error) {
	query := `
		SELECT id, ip, reason, expire_at, created_at 
		FROM blacklist 
		WHERE ip = $1 AND (expire_at IS NULL OR expire_at > NOW())
	`
	b := &model.Blacklist{}
	err := r.db.QueryRowContext(ctx, query, ip).Scan(
		&b.ID, &b.IP, &b.Reason, &b.ExpireAt, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklist by IP: %w", err)
	}
	return b, nil
}

func (r *BlacklistRepo) IsBlacklisted(ctx context.Context, ip string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM blacklist WHERE ip = $1 AND (expire_at IS NULL OR expire_at > NOW()))`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, ip).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return exists, nil
}

func (r *BlacklistRepo) List(ctx context.Context, filter *model.BlacklistFilter) ([]*model.Blacklist, error) {
	conditions := []string{}
	args := []interface{}{}
	argIdx := 1

	if filter.IP != "" {
		conditions = append(conditions, fmt.Sprintf("ip LIKE $%d", argIdx))
		args = append(args, "%"+filter.IP+"%")
		argIdx++
	}
	if filter.ActiveOnly {
		conditions = append(conditions, "(expire_at IS NULL OR expire_at > NOW())")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	query := fmt.Sprintf(`
		SELECT id, ip, reason, expire_at, created_at
		FROM blacklist %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.Limit(), filter.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list blacklist: %w", err)
	}
	defer rows.Close()

	var list []*model.Blacklist
	for rows.Next() {
		b := &model.Blacklist{}
		err := rows.Scan(&b.ID, &b.IP, &b.Reason, &b.ExpireAt, &b.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blacklist: %w", err)
		}
		list = append(list, b)
	}
	return list, nil
}

func (r *BlacklistRepo) Update(ctx context.Context, id int64, reason string, expireAt *time.Time) error {
	var expireAtArg interface{}
	if expireAt != nil {
		expireAtArg = *expireAt
	}

	query := `UPDATE blacklist SET reason = $1, expire_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, reason, expireAtArg, id)
	if err != nil {
		return fmt.Errorf("failed to update blacklist: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("blacklist entry not found")
	}
	return nil
}

func (r *BlacklistRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM blacklist WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete blacklist: %w", err)
	}
	return nil
}

func (r *BlacklistRepo) DeleteByIP(ctx context.Context, ip string) error {
	query := `DELETE FROM blacklist WHERE ip = $1`
	_, err := r.db.ExecContext(ctx, query, ip)
	if err != nil {
		return fmt.Errorf("failed to delete blacklist by IP: %w", err)
	}
	return nil
}

func (r *BlacklistRepo) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM blacklist WHERE expire_at IS NOT NULL AND expire_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired blacklist: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

func (r *BlacklistRepo) Count(ctx context.Context, activeOnly bool) (int64, error) {
	query := `SELECT COUNT(*) FROM blacklist`
	if activeOnly {
		query += " WHERE expire_at IS NULL OR expire_at > NOW()"
	}
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count blacklist: %w", err)
	}
	return count, nil
}
