package models

import "time"

type DayStatus string

const (
	DayPending   DayStatus = "ожидание"
	DayCompleted DayStatus = "выполнен"
	DayMissed    DayStatus = "пропущен"
	DayOff       DayStatus = "выходной"
)

type Day struct {
	Date          time.Time
	ActiveMinutes int
	Status        DayStatus
	DebtMinutes   int
	Description   string
}

// String реализует интерфейс fmt.Stringer для удобного вывода.
func (s DayStatus) String() string {
	return string(s)
}

// IsValid проверяет, что статус корректный (дополнительная защита).
func (s DayStatus) IsValid() bool {
	switch s {
	case DayCompleted, DayMissed, DayOff:
		return true
	}
	return false
}
