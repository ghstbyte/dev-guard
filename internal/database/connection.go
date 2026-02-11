package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewConnection() (*sql.DB, error) {

	connStr := "host=localhost port=5432 user=postgres password=8190 dbname=devguard sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("Не удалось открыть подключение: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping не прошёл: %w", err)
	}

	return db, nil

}
