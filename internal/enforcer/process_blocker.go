package enforcer

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/tklauser/ps"
)

type Enforcer struct {
	forbidden []string
	enabled   bool
}

func NewEnforcer(forbidden []string, enabled bool) *Enforcer {
	return &Enforcer{
		forbidden: forbidden,
		enabled:   enabled,
	}
}

func (e *Enforcer) Start(ctx context.Context) {
	if !e.enabled {
		log.Println("Enforcer disabled")
		return
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Enforcer stopped by context")
			return
		case <-ticker.C:
			allProcesses, err := ps.Processes()
			if err != nil {
				log.Printf("error getting processes: %v", err)
				continue
			}

			forbiddenList := e.forbidden
			if len(forbiddenList) == 0 {
				continue
			}

			for _, p := range allProcesses {
				processName := strings.ToLower(p.Command())
				for _, forbidden := range forbiddenList {
					forbiddenLower := strings.ToLower(forbidden)
					if strings.Contains(processName, forbiddenLower) {
						timestamp := time.Now().Format("2006-01-02 15:04:05")
						log.Printf("[STRICT MODE] Violation: process=%s pid=%d time=%s", p.Command(), p.PID(), timestamp)
					}
				}
			}
		}

	}
}
