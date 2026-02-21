package tracker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tklauser/ps"
)

type Tracker struct {
	ProcessName   string
	activeSeconds int64
	mu            sync.Mutex
	lastRunning   bool
}

func IsProcessRunning(processName string) (bool, error) {
	RunningProcesses, err := ps.Processes()
	if err != nil {
		return false, err
	}

	for _, p := range RunningProcesses {
		if p.Command() == processName {
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
				t.AddSeconds(10)
			}

			if running != t.lastRunning {
				if running {
					log.Println("Process active")
				} else {
					log.Println("Process not running")
				}
				t.lastRunning = running
			}
		}
	}
}

func (t *Tracker) AddSeconds(seconds int64) {
	t.mu.Lock()
	t.activeSeconds += seconds
	t.mu.Unlock()
}

func (t *Tracker) GetActiveMinutes() int {
	t.mu.Lock()
	secs := t.activeSeconds
	t.mu.Unlock()
	return int(secs / 60)
}

func (t *Tracker) SetActiveSeconds(secs int64) {
	t.mu.Lock()
	t.activeSeconds = secs
	t.mu.Unlock()
}
