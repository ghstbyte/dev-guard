package controller

import (
	"context"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/dayservice"
	"dev-guard_app/internal/tracker"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Controller struct {
	cfg        *config.Config
	repo       *database.Repository
	track      *tracker.Tracker
	dayService *dayservice.DayService
	wg         *sync.WaitGroup
	rootCtx    context.Context
	cancel     context.CancelFunc
}

func NewController(cfg *config.Config, repo *database.Repository, track *tracker.Tracker, dayService *dayservice.DayService) *Controller {
	rootCtx, cancel := context.WithCancel(context.Background())
	return &Controller{
		cfg:        cfg,
		repo:       repo,
		track:      track,
		dayService: dayService,
		wg:         &sync.WaitGroup{},
		rootCtx:    rootCtx,
		cancel:     cancel,
	}
}

func (c *Controller) Run() error {
	log.Println("=== Запуск трекера процесса ===")
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.track.StartTracking(c.rootCtx); err != nil && err != context.Canceled {
			log.Printf("Tracker завершился с ошибкой: %v", err)
		}
	}()

	if err := c.dayService.LoadOrCreateCurrentDay(c.rootCtx); err != nil {
		return err
	}

	c.dayService.ActivateStrictModeIfNeeded(c.rootCtx)

	saveTicker := time.NewTicker(60 * time.Second)
	defer saveTicker.Stop()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.rootCtx.Done():
				return
			case <-saveTicker.C:
				if err := c.dayService.Update(c.rootCtx); err != nil {
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
	c.cancel()
	c.wg.Wait()

	bgCtx := context.Background()
	if err := c.dayService.FinalClose(bgCtx); err != nil {
		log.Printf("Ошибка финального закрытия: %v", err)
	}

	log.Println("Graceful shutdown completed. Goodbye!")
	return nil
}
