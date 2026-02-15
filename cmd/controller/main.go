package main

import (
	"bufio"
	"context"
	"database/sql"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/decision"
	"dev-guard_app/internal/enforcer"
	"dev-guard_app/internal/models"
	"dev-guard_app/internal/tracker"
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

	// Подключение к БД
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
	)
	log.Printf("using connection string: %s", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("cannot open DB: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("cannot ping DB: %v", err)
	}
	log.Println("successfully connected to database")

	// === Репозиторий и работа с днём (перемещено вверх) ===
	repo := database.NewRepository(db)

	ctx := context.Background()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	log.Printf("checking day for date: %s", today.Format("2006-01-02"))

	day, err := repo.GetDayByDate(ctx, today)
	if err != nil {
		log.Fatalf("failed to get day: %v", err)
	}

	if day == nil {
		day = &models.Day{
			Date:          today,
			ActiveMinutes: 0,
			Status:        models.DayPending,
			DebtMinutes:   0,
			Description:   "created automatically by tracker",
		}
		if err := repo.CreateDayIfNotExists(ctx, day); err != nil {
			log.Fatalf("failed to create day: %v", err)
		}
		log.Println("Создан новый день с 0 минут")
	} else {
		log.Printf("Найден существующий день: %d минут (статус %s)", day.ActiveMinutes, day.Status)
	}

	// === Создание трекера и загрузка минут из БД ===
	log.Println("=== Запуск трекера процесса ===")

	track := tracker.Tracker{
		ProcessName: cfg.Tracker.TrackerProcess,
	}
	track.ActiveSeconds = int64(day.ActiveMinutes * 60)
	log.Printf("Трекер загружен с %d минутами (%d секундами) из БД", day.ActiveMinutes, track.ActiveSeconds)

	// === Запуск трекинга  ===
	trackCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go track.StartTracking(trackCtx)

	yesterday := today.AddDate(0, 0, -1)
	prevDay, err := repo.GetDayByDate(ctx, yesterday)
	if prevDay != nil && prevDay.Status == models.DayMissed && cfg.Enforcer.StrictMode.Enabled {
		log.Println("STRICT MODE ACTIVATED")

		enf := enforcer.NewEnforcer(cfg.Enforcer.StrictMode.ForbiddenProcesses, true)
		enf.Start(trackCtx) // без go — горутина уже внутри Start
	}

	saveTicker := time.NewTicker(60 * time.Second)
	defer saveTicker.Stop()

	go func() {
		for {
			select {
			case <-trackCtx.Done():
				return
			case <-saveTicker.C:
				currentMinutes := track.GetActiveMinutes()
				day.ActiveMinutes = currentMinutes
				if err := repo.UpdateDay(ctx, *day); err != nil {
					log.Printf("Ошибка периодического сохранения: %v", err)
				} else {
					log.Printf("Периодическое сохранение: %d минут активности", currentMinutes)
				}
			}
		}
	}()

	// === Пауза на завершение ===
	fmt.Println("Press Enter to exit...")
	bufio.NewScanner(os.Stdin).Scan()

	cancel()

	finalMinutes := track.GetActiveMinutes()
	day.ActiveMinutes = finalMinutes

	closedDay := decision.CloseDay(*day, cfg.Tracker.DailyTargetMinutes)

	if err := repo.UpdateDay(ctx, closedDay); err != nil {
		log.Printf("Ошибка финального сохранения: %v", err)
	} else {
		log.Printf("День закрыт: статус %s, активных минут %d, долг %d минут",
			closedDay.Status, closedDay.ActiveMinutes, closedDay.DebtMinutes)
	}
	log.Println("Application finished")
}
