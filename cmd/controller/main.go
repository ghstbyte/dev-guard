package main

import (
	"bufio"
	"dev-guard_app/internal/config"
	"fmt"
	"log"
	"os"
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

	fmt.Printf("Target: %d min\n", cfg.Tracker.DailyTargetMinutes)
	fmt.Printf("Tracked: %s\n", cfg.Tracker.TrackerProcess)
	fmt.Printf("Forbiden: %v\n", cfg.Enforcer.StrictMode.ForbiddenProcesses)

	fmt.Println("Press Enter to exit...")
	bufio.NewScanner(os.Stdin).Scan()
	log.Println("Application finished")
}
