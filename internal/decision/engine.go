package decision

import (
	"dev-guard_app/internal/models"
)

func IsDayCompleted(ActiveMinutes, dailyNorm int) models.DayStatus {
	if ActiveMinutes >= dailyNorm {
		return models.DayCompleted
	}
	return models.DayMissed
}

func CloseDay(day models.Day, dailyNorm int) models.Day {
	if day.Status == models.DayOff {
		return day
	}

	status := IsDayCompleted(day.ActiveMinutes, dailyNorm)
	day.Status = status

	if status == models.DayMissed {
		day.DebtMinutes = dailyNorm - day.ActiveMinutes
	} else {
		day.DebtMinutes = 0
	}
	return day
}
