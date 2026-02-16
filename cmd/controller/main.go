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
	"os/signal"
	"sync"
	"syscall"
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

	log.Println("=== Запуск трекера процесса ===")

	track := tracker.Tracker{
		ProcessName: cfg.Tracker.TrackerProcess,
	}
	track.ActiveSeconds = int64(day.ActiveMinutes * 60)
	log.Printf("Трекер загружен с %d минутами (%d секундами) из БД", day.ActiveMinutes, track.ActiveSeconds)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var enforcerCancel context.CancelFunc

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := track.StartTracking(rootCtx); err != nil && err != context.Canceled {
			log.Printf("Tracker завершился с ошибкой: %v", err)
		}
	}()

	saveTicker := time.NewTicker(60 * time.Second)
	defer saveTicker.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-saveTicker.C:
				currentMinutes := track.GetActiveMinutes()
				previousMinutes := day.ActiveMinutes
				day.ActiveMinutes = currentMinutes
				if day.Status == models.DayPending {
					if day.ActiveMinutes >= cfg.Tracker.DailyTargetMinutes {
						closedDay := decision.CloseDay(*day, cfg.Tracker.DailyTargetMinutes)
						*day = closedDay
						log.Printf("День выполнен: %d минут (>= нормы %d), статус теперь %s", day.ActiveMinutes, cfg.Tracker.DailyTargetMinutes, day.Status)

						if enforcerCancel != nil {
							enforcerCancel()
							enforcerCancel = nil
							log.Println("[STRICT MODE] Disabled")
						}
					}
				}

				if day.Status == models.DayMissed {
					newDebt := cfg.Tracker.DailyTargetMinutes - day.ActiveMinutes
					if newDebt < 0 {
						newDebt = 0
					}
					if newDebt != day.DebtMinutes {
						day.DebtMinutes = newDebt
						log.Printf("[ДОЛГ УМЕНЬШЕН] Было %d, стало %d (активных минут %d)", day.DebtMinutes+(day.ActiveMinutes-previousMinutes), day.DebtMinutes, day.ActiveMinutes)
					}
					if day.ActiveMinutes >= cfg.Tracker.DailyTargetMinutes {
						day.Status = models.DayCompleted
						day.DebtMinutes = 0
						log.Printf("[ДЕНЬ РЕАБИЛИТИРОВАН] Статус %s, долг обнулён: %d минут (>= нормы %d)", day.Status, day.ActiveMinutes, cfg.Tracker.DailyTargetMinutes)

						if enforcerCancel != nil {
							enforcerCancel()
							enforcerCancel = nil
							log.Println("[STRICT MODE] Disabled — день реабилитирован")
						}
					}
				}

				if err := repo.UpdateDay(rootCtx, *day); err != nil {
					log.Printf("Ошибка периодического сохранения: %v", err)
				} else {
					log.Printf("Периодическое сохранение: %d минут активности", currentMinutes)
				}
			}
		}
	}()

	yesterday := today.AddDate(0, 0, -1)
	prevDay, err := repo.GetDayByDate(rootCtx, yesterday)
	if err != nil {
		log.Printf("Ошибка получения вчерашнего дня: %v", err)
	} else if prevDay != nil && prevDay.Status == models.DayMissed && cfg.Enforcer.StrictMode.Enabled && day.Status == models.DayPending {
		enf := enforcer.NewEnforcer(cfg.Enforcer.StrictMode.ForbiddenProcesses, true)
		enforcerCtx, cancel := context.WithCancel(rootCtx)
		enforcerCancel = cancel
		log.Println("[STRICT MODE] ACTIVATED")

		wg.Add(1)
		go func() {
			defer wg.Done()
			enf.Start(enforcerCtx)
		}()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	log.Println("DevGuard is running. Press Ctrl+C to stop gracefully.")

	<-sigs
	log.Println("Signal received. Starting graceful shutdown...")

	cancel()

	wg.Wait()

	finalMinutes := track.GetActiveMinutes()
	day.ActiveMinutes = finalMinutes
	closedDay := decision.CloseDay(*day, cfg.Tracker.DailyTargetMinutes)

	bgCtx := context.Background()
	if err := repo.UpdateDay(bgCtx, closedDay); err != nil {
		log.Printf("Ошибка финального сохранения: %v", err)
	} else {
		log.Printf("День закрыт при завершении: статус %s, активных минут %d, долг %d минут",
			closedDay.Status, closedDay.ActiveMinutes, closedDay.DebtMinutes)
	}

	log.Println("Graceful shutdown completed. Goodbye!")
}
