package database

import (
	"context"
	"database/sql"
	"dev-guard_app/internal/models"
	"fmt"
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
			return nil, nil
		}
		return nil, err
	}

	return &day, nil
}

func (r *Repository) CreateDayIfNotExists(ctx context.Context, day *models.Day) error {

	existingDay, err := r.GetDayByDate(ctx, day.Date)
	if err != nil {
		return err
	}
	if existingDay != nil {
		return nil
	}

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

func (r *Repository) UpdateDay(ctx context.Context, day models.Day) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE days 
        SET 
            status = $1,
            debt_minutes = $2, 
            active_minutes = $3,
            description = $4
        WHERE date = $5
    `, day.Status, day.DebtMinutes, day.ActiveMinutes, day.Description, day.Date)

	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return fmt.Errorf("no day updated for date %v", day.Date)
	}

	return nil
}
