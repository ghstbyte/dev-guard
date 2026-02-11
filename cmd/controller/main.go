package main

import (
	"bufio"
	"context"
	"database/sql"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/models"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	log.Println("Application started")

	cfg, err := config.Load("C:/Users/graveyard/Desktop/dev-guard_app/configs/config.yaml")
	if err != nil {
		log.Printf("Config ERROR: %v", err)
		fmt.Println("Press Enter to exit...")
		bufio.NewScanner(os.Stdin).Scan()
		return
	}

	ctx := context.Background()

	// 1. Загрузка конфига (подключение к БД)
	connStr := "host=localhost port=5432 user=postgres password=8190 dbname=devguard sslmode=disable"
	log.Printf("using connection string: %s", connStr)

	// 2. Подключение к БД
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("cannot open DB: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("cannot ping DB: %v", err)
	}
	log.Printf("successfully connected to database")

	// 3. Создаём репозиторий
	repo := database.NewRepository(db)

	// 4. Берём "сегодня" как date (без времени), по текущему локальному дню
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	log.Printf("checking day for date: %s", today.Format("2006-01-02"))

	// 5. Пробуем получить запись за сегодня
	day, err := repo.GetDayByDate(ctx, today)
	if err != nil {
		log.Fatalf("failed to get day: %v", err)
	}

	if day == nil {
		// 6. Дня нет → создаём новую запись
		newDay := &models.Day{
			Date:          today,
			ActiveMinutes: 0,
			Status:        "ok",
			DebtMinutes:   0,
			Description:   "created by main.go (first access today)",
		}

		log.Printf("no record found for %s", today.Format("2006-01-02"))
		log.Printf("creating new day: %+v", newDay)

		if err := repo.CreateDayIfNotExists(ctx, newDay); err != nil {
			log.Fatalf("failed to create day: %v", err)
		}

		log.Printf("new day successfully created: %+v", newDay)
	} else {
		log.Printf("found existing day record: %+v", day)
	}

	fmt.Println("Config loaded:")
	fmt.Printf("Daily target: %d minutes\n", cfg.Tracker.DailyTargetMinutes)
	fmt.Printf("Strict mode: %v\n", cfg.Enforcer.StrictMode.Enabled)
	fmt.Printf("Tracked process: %s\n", cfg.Tracker.TrackerProcess)
	fmt.Printf("Forbiden processes: %v\n", cfg.Enforcer.StrictMode.ForbiddenProcesses)

	fmt.Println("Press Enter to exit...")
	bufio.NewScanner(os.Stdin).Scan()
	log.Println("Application finished")
}
