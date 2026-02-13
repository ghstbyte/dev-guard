package tracker

import (
	"context"
	"log"
	"time"

	"github.com/tklauser/ps"
)

type Tracker struct {
	ProcessName   string
	ActiveSeconds int64
}

func IsProcessRunning(processName string) (bool, error) {
	RunningProcesses, err := ps.Processes()
	if err != nil {
		return false, err
	}

	for _, p := range RunningProcesses {
		if p.Command() == processName {
			log.Println("Найдено совпадение:", processName)
			return true, nil
		}
	}
	return false, nil
}

func (t *Tracker) StartTracking(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			running, err := IsProcessRunning(t.ProcessName)
			if err != nil {
				log.Printf("error checking process %s: %v", t.ProcessName, err)
				continue
			}
			if running {
				interval := 10 * time.Second
				t.ActiveSeconds += int64(interval.Seconds())
				log.Println("Process active")
			} else {
				log.Println("Process not running")
			}
		}
	}
}

func (t *Tracker) GetActiveMinutes() int {
	return int(t.ActiveSeconds / 60)
}
