package models

import "time"

type DayStatus string

const (
	DayPending   DayStatus = "waiting"
	DayCompleted DayStatus = "completed"
	DayMissed    DayStatus = "missed"
	DayOff       DayStatus = "off"
)

type Day struct {
	Date          time.Time
	ActiveMinutes int
	Status        DayStatus `db:"status"`
	DebtMinutes   int
	Description   string
}

func (s DayStatus) String() string {
	switch s {
	case DayCompleted:
		return "День выполнен ✅ "
	case DayMissed:
		return "День пропущен ❌ "
	default:
		return string(s)
	}
}

// IsValid проверяет, что статус корректный (дополнительная защита).
func (s DayStatus) IsValid() bool {
	switch s {
	case DayCompleted, DayMissed, DayOff:
		return true
	}
	return false
}
