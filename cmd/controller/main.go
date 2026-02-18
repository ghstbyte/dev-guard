package main

import (
	"database/sql"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/controller"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/dayservice"
	"dev-guard_app/internal/tracker"
	"fmt"
	"log"
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

	track := tracker.Tracker{
		ProcessName: cfg.Tracker.TrackerProcess,
	}

	dayService := dayservice.NewDayService(repo, cfg, &track)

	ctrl := controller.NewController(cfg, repo, &track, dayService)

	if err := ctrl.Run(); err != nil {
		log.Fatal(err)
	}
}
