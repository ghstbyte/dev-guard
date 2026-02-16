package main

import (
	"context"
	"database/sql"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/dayservice"
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

	log.Println("=== Запуск трекера процесса ===")
	track := tracker.Tracker{
		ProcessName: cfg.Tracker.TrackerProcess,
	}

	dayService := dayservice.NewDayService(repo, cfg, &track)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := track.StartTracking(rootCtx); err != nil && err != context.Canceled {
			log.Printf("Tracker завершился с ошибкой: %v", err)
		}
	}()

	if err := dayService.LoadOrCreateCurrentDay(rootCtx); err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	prevDay, err := repo.GetDayByDate(rootCtx, yesterday)
	if err != nil {
		log.Printf("Ошибка получения вчерашнего дня: %v", err)
	} else if prevDay != nil && prevDay.Status == models.DayMissed && cfg.Enforcer.StrictMode.Enabled && dayService.GetCurrentDay().Status == models.DayPending {
		enf := enforcer.NewEnforcer(cfg.Enforcer.StrictMode.ForbiddenProcesses, true)
		enforcerCtx, cancelCtx := context.WithCancel(rootCtx)
		dayService.SetStrictCancel(cancelCtx)
		log.Println("[STRICT MODE] ACTIVATED")
		wg.Add(1)
		go func() {
			defer wg.Done()
			enf.Start(enforcerCtx)
		}()
	}

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
				if err := dayService.Update(rootCtx); err != nil {
					log.Printf("Ошибка обновления дня: %v", err)
				}
			}
		}

	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	log.Println("DevGuard is running. Press Ctrl+C to stop gracefully.")
	<-sigs

	log.Println("Signal received. Starting graceful shutdown...")
	cancel()
	wg.Wait()

	bgCtx := context.Background()
	if err := dayService.FinalClose(bgCtx); err != nil {
		log.Printf("Ошибка финального закрытия: %v", err)
	}
	log.Println("Graceful shutdown completed. Goodbye!")
}
