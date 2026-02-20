package database

import (
	"database/sql"
	"dev-guard_app/internal/config"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func ConnectWithRetry(cfg *config.Config) (*sql.DB, error) {
	const maxRetries int = 10
	const baseDelay = 2 * time.Second

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		connStr := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
		)

		delay := baseDelay * time.Duration(1<<(attempt-1))
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			lastErr = err
			log.Printf("Попытка %d/%d: не удалось подключиться к базе данных: %v. Ждём...", attempt, maxRetries, lastErr)
			time.Sleep(delay)
			continue
		}

		if pingErr := db.Ping(); pingErr != nil {
			db.Close()
			lastErr = pingErr
			log.Printf("Попытка %d/%d: не удалось подключиться к базе данных: %v. Ждём...", attempt, maxRetries, lastErr)
			time.Sleep(delay)
			continue
		}

		log.Printf("Успешное подключение к базе данных (попытка %d)", attempt)
		return db, nil
	}
	return nil, fmt.Errorf("Не удалось подключиться к базе данных после %d попыток: %w", maxRetries, lastErr)
}
