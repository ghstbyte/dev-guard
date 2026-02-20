package main

import (
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/controller"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/dayservice"
	"dev-guard_app/internal/tracker"
	"log"
)

func main() {
	log.Println("Application started")

	cfg, err := config.Load("C:/Users/graveyard/Desktop/dev-guard_app/configs/config.yaml")
	if err != nil {
		log.Fatalf("Не удалось загрузить конфиг: %v", err)
	}

	db, err := database.ConnectWithRetry(cfg)
	if err != nil {
		log.Fatalf("Критическая ошибка: %v", err)
	}

	repo := database.NewRepository(db)

	track := tracker.Tracker{
		ProcessName: cfg.Tracker.TrackerProcess,
	}

	dayService := dayservice.NewDayService(repo, cfg, &track)

	ctrl := controller.NewController(cfg, repo, &track, dayService)

	if err := ctrl.Run(); err != nil {
		log.Fatal(err)
	}
}
