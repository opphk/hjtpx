package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/opphk/captcha-system/internal/model"
)

type ChallengeRepository struct {
	*BaseRepository
}

func NewChallengeRepository(db *sqlx.DB) *ChallengeRepository {
	return &ChallengeRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *ChallengeRepository) Create(ctx context.Context, challenge *model.Challenge) error {
	challenge.CreatedAt = time.Now()
	challenge.UpdatedAt = time.Now()
	query := `INSERT INTO challenges (challenge_id, type, difficulty, data, solution, expires_at, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		challenge.ChallengeID,
		challenge.Type,
		challenge.Difficulty,
		challenge.Data,
		challenge.Solution,
		challenge.ExpiresAt,
		challenge.CreatedAt,
		challenge.UpdatedAt,
	)
	return err
}

func (r *ChallengeRepository) GetByChallengeID(ctx context.Context, challengeID string) (*model.Challenge, error) {
	var challenge model.Challenge
	query := `SELECT id, challenge_id, type, difficulty, data, solution, expires_at, created_at, updated_at
			  FROM challenges WHERE challenge_id = $1`
	err := r.db.GetContext(ctx, &challenge, query, challengeID)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (r *ChallengeRepository) Delete(ctx context.Context, challengeID string) error {
	query := `DELETE FROM challenges WHERE challenge_id = $1`
	_, err := r.db.ExecContext(ctx, query, challengeID)
	return err
}

func (r *ChallengeRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM challenges WHERE expires_at < $1`
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
