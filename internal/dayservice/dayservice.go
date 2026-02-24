package dayservice

import (
	"context"
	"dev-guard_app/internal/config"
	"dev-guard_app/internal/database"
	"dev-guard_app/internal/decision"
	"dev-guard_app/internal/enforcer"
	"dev-guard_app/internal/models"
	"dev-guard_app/internal/tracker"
	"errors"
	"log"
	"sync"
	"time"
)

type DayService struct {
	repo              *database.Repository
	cfg               *config.Config
	currentDay        *models.Day
	strictCancel      context.CancelFunc
	tracker           *tracker.Tracker
	lastLoggedMinutes int
	isStrict          bool
	mu                sync.Mutex
}

func NewDayService(repo *database.Repository, cfg *config.Config, track *tracker.Tracker) *DayService {
	return &DayService{
		repo:              repo,
		cfg:               cfg,
		tracker:           track,
		currentDay:        nil,
		strictCancel:      nil,
		lastLoggedMinutes: -1,
		isStrict:          false,
	}
}

func (s *DayService) LoadOrCreateCurrentDay(ctx context.Context) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	log.Printf("checking day for date: %s", today.Format("2006-01-02"))

	day, err := s.repo.GetDayByDate(ctx, today)
	if err != nil {
		log.Printf("failed to get day: %v", err)
		return err
	}

	if day == nil {
		day = &models.Day{
			Date:          today,
			ActiveMinutes: 0,
			Status:        models.DayPending,
			DebtMinutes:   0,
			Description:   "created automatically by tracker",
		}
		if err := s.repo.CreateDayIfNotExists(ctx, day); err != nil {
			log.Printf("failed to create day: %v", err)
			return err
		}
		log.Println("Создан новый день с 0 минут")
	} else {
		log.Printf("Найден существующий день: %d минут (статус %s)", day.ActiveMinutes, day.Status)
	}

	s.currentDay = day

	s.tracker.SetActiveSeconds(int64(s.currentDay.ActiveMinutes * 60))
	log.Printf("Трекер загружен с %d минутами из БД", s.currentDay.ActiveMinutes)

	return nil
}

func (s *DayService) GetCurrentDay() *models.Day {
	return s.currentDay
}

func (s *DayService) SetStrictCancel(cancel context.CancelFunc) {
	s.strictCancel = cancel
}

func (s *DayService) Update(ctx context.Context) error {
	if s.currentDay == nil {
		return errors.New("current day not loaded")
	}

	currentMinutes := s.tracker.GetActiveMinutes()
	previousMinutes := s.currentDay.ActiveMinutes
	s.currentDay.ActiveMinutes = currentMinutes

	if s.currentDay.Status == models.DayPending {
		if s.currentDay.ActiveMinutes >= s.cfg.Tracker.DailyTargetMinutes {
			closedDay := decision.CloseDay(*s.currentDay, s.cfg.Tracker.DailyTargetMinutes)
			*s.currentDay = closedDay
			log.Printf("День выполнен: %d минут (>= нормы %d), статус теперь %s", s.currentDay.ActiveMinutes, s.cfg.Tracker.DailyTargetMinutes, s.currentDay.Status)

			if s.strictCancel != nil {
				s.strictCancel()
				s.strictCancel = nil
				log.Println("[STRICT MODE] Disabled")
			}
		}
	}
	if s.currentDay.Status == models.DayMissed {
		newDebt := s.cfg.Tracker.DailyTargetMinutes - s.currentDay.ActiveMinutes
		if newDebt < 0 {
			newDebt = 0
		}
		if newDebt != s.currentDay.DebtMinutes {
			s.currentDay.DebtMinutes = newDebt
			log.Printf("[ДОЛГ УМЕНЬШЕН] Было %d, стало %d (активных минут %d)", s.currentDay.DebtMinutes+(s.currentDay.ActiveMinutes-previousMinutes), s.currentDay.DebtMinutes, s.currentDay.ActiveMinutes)
		}
		if s.currentDay.ActiveMinutes >= s.cfg.Tracker.DailyTargetMinutes {
			s.currentDay.Status = models.DayCompleted
			s.currentDay.DebtMinutes = 0
			log.Printf("[ДЕНЬ РЕАБИЛИТИРОВАН] Статус %s, долг обнулён: %d минут (>= нормы %d)", s.currentDay.Status, s.currentDay.ActiveMinutes, s.cfg.Tracker.DailyTargetMinutes)

			if s.strictCancel != nil {
				s.strictCancel()
				s.strictCancel = nil
				log.Println("[STRICT MODE] Disabled — день реабилитирован")
			}
		}
	}
	if err := s.repo.UpdateDay(ctx, *s.currentDay); err != nil {
		log.Printf("Ошибка периодического сохранения: %v", err)
		return err
	} else {
		if currentMinutes != s.lastLoggedMinutes {
			log.Printf("Периодическое сохранение: %d минут активности", currentMinutes)
			s.lastLoggedMinutes = currentMinutes
		}
	}

	return nil
}

func (s *DayService) FinalClose(ctx context.Context) error {
	if s.currentDay == nil {
		return errors.New("current day not loaded")
	}

	finalMinutes := s.tracker.GetActiveMinutes()
	s.currentDay.ActiveMinutes = finalMinutes

	closedDay := decision.CloseDay(*s.currentDay, s.cfg.Tracker.DailyTargetMinutes)
	*s.currentDay = closedDay

	if err := s.repo.UpdateDay(ctx, closedDay); err != nil {
		log.Printf("Ошибка финального сохранения: %v", err)
		return err
	} else {
		log.Printf("День закрыт при завершении: статус %s, активных минут %d, долг %d минут",
			closedDay.Status, closedDay.ActiveMinutes, closedDay.DebtMinutes)
	}

	return nil
}

func (s *DayService) ActivateStrictModeIfNeeded(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStrict {
		log.Println("Strict-mode уже запущен.")
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	prevDay, err := s.repo.GetDayByDate(ctx, yesterday)
	if err != nil {
		log.Printf("Ошибка при получении предыдущего дня %v", err)
		return
	}

	if prevDay != nil && prevDay.Status == models.DayMissed && s.cfg.Enforcer.StrictMode.Enabled && s.currentDay.Status == models.DayPending {
		enf := enforcer.NewEnforcer(s.cfg.Enforcer.StrictMode.ForbiddenProcesses, true)
		enforcerCtx, cancel := context.WithCancel(ctx)
		s.SetStrictCancel(cancel)
		s.isStrict = true
		log.Println("[STRICT MODE] ACTIVATED")

		go func() {
			enf.Start(enforcerCtx)
		}()
	}
}
