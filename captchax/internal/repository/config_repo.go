package repository

import (
	"context"
	"database/sql"
	"fmt"

	"captchax/internal/model"
)

type ConfigRepo struct {
	db *sql.DB
}

func NewConfigRepo(db *sql.DB) *ConfigRepo {
	return &ConfigRepo{db: db}
}

func (r *ConfigRepo) Get(ctx context.Context, key string) (*model.Config, error) {
	query := `SELECT id, key, value, description, updated_at FROM captcha_config WHERE key = $1`
	cfg := &model.Config{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&cfg.ID, &cfg.Key, &cfg.Value, &cfg.Description, &cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return cfg, nil
}

func (r *ConfigRepo) GetValue(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM captcha_config WHERE key = $1`
	var value string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config value: %w", err)
	}
	return value, nil
}

func (r *ConfigRepo) List(ctx context.Context) ([]*model.Config, error) {
	query := `SELECT id, key, value, description, updated_at FROM captcha_config ORDER BY key`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.Config
	for rows.Next() {
		cfg := &model.Config{}
		err := rows.Scan(&cfg.ID, &cfg.Key, &cfg.Value, &cfg.Description, &cfg.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (r *ConfigRepo) Set(ctx context.Context, key, value, description string) error {
	query := `
		INSERT INTO captcha_config (key, value, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = $2, description = $3, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, key, value, description)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

func (r *ConfigRepo) Update(ctx context.Context, key, value string) error {
	query := `UPDATE captcha_config SET value = $1, updated_at = NOW() WHERE key = $2`
	result, err := r.db.ExecContext(ctx, query, value, key)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("config key not found: %s", key)
	}
	return nil
}

func (r *ConfigRepo) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM captcha_config WHERE key = $1`
	_, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

func (r *ConfigRepo) GetSystemConfig(ctx context.Context) (*model.SystemConfig, error) {
	configs, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	sysCfg := model.DefaultSystemConfig()

	for _, cfg := range configs {
		switch cfg.Key {
		case "max_attempts_per_ip":
			fmt.Sscanf(cfg.Value, "%d", &sysCfg.MaxAttemptsPerIP)
		case "block_duration_minutes":
			fmt.Sscanf(cfg.Value, "%d", &sysCfg.BlockDurationMins)
		case "risk_threshold":
			fmt.Sscanf(cfg.Value, "%d", &sysCfg.RiskThreshold)
		case "session_timeout_seconds":
			fmt.Sscanf(cfg.Value, "%d", &sysCfg.SessionTimeoutSecs)
		case "enable_whitelist":
			sysCfg.EnableWhitelist = cfg.Value == "true"
		case "enable_blacklist":
			sysCfg.EnableBlacklist = cfg.Value == "true"
		}
	}

	return sysCfg, nil
}

func (r *ConfigRepo) SetSystemConfig(ctx context.Context, sysCfg *model.SystemConfig) error {
	configs := map[string]string{
		"max_attempts_per_ip":    fmt.Sprintf("%d", sysCfg.MaxAttemptsPerIP),
		"block_duration_minutes": fmt.Sprintf("%d", sysCfg.BlockDurationMins),
		"risk_threshold":         fmt.Sprintf("%d", sysCfg.RiskThreshold),
		"session_timeout_seconds": fmt.Sprintf("%d", sysCfg.SessionTimeoutSecs),
		"enable_whitelist":        boolToString(sysCfg.EnableWhitelist),
		"enable_blacklist":        boolToString(sysCfg.EnableBlacklist),
	}

	for key, value := range configs {
		if err := r.Update(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
