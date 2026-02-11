package database

import (
	"context"
	"database/sql"
	"dev-guard_app/internal/models"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDayByDate(ctx context.Context, date time.Time) (*models.Day, error) {
	query := `
		SELECT date, active_minutes, status, debt_minutes, description
		FROM days
		WHERE date = $1
	`

	var day models.Day
	err := r.db.QueryRowContext(ctx, query, date).Scan(
		&day.Date,
		&day.ActiveMinutes,
		&day.Status,
		&day.DebtMinutes,
		&day.Description,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // день не найден
		}
		return nil, err
	}

	return &day, nil
}

func (r *Repository) CreateDayIfNotExists(ctx context.Context, day *models.Day) error {
	// сначала пытаемся получить день
	existingDay, err := r.GetDayByDate(ctx, day.Date)
	if err != nil {
		return err
	}
	if existingDay != nil {
		return nil // день уже есть, ничего не делаем
	}

	// создаем новый день
	query := `
		INSERT INTO days (date, active_minutes, status, debt_minutes, description)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.ExecContext(ctx, query,
		day.Date,
		day.ActiveMinutes,
		day.Status,
		day.DebtMinutes,
		day.Description,
	)
	return err
}
