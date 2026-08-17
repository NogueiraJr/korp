package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"korp/estoque/internal/models"
)

type ActivityRepository struct {
	Pool *pgxpool.Pool
}

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{Pool: pool}
}

// Log appends an audit event. Failures are swallowed on purpose so logging
// never breaks a business operation.
func (r *ActivityRepository) Log(ctx context.Context, event, entity string, details interface{}) error {
	var raw []byte
	var err error
	if details != nil {
		raw, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}
	_, err = r.Pool.Exec(ctx,
		`INSERT INTO activity_logs (event, entity, details) VALUES ($1, $2, $3)`,
		event, entity, raw)
	return err
}

func (r *ActivityRepository) List(ctx context.Context, limit int) ([]models.ActivityLog, error) {
	rows, err := r.Pool.Query(ctx,
		`SELECT id, event, entity, details, created_at
		 FROM activity_logs ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []models.ActivityLog{}
	for rows.Next() {
		var log models.ActivityLog
		if err := rows.Scan(&log.ID, &log.Event, &log.Entity, &log.Details, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}