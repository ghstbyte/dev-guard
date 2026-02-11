package models

import "time"

type Day struct {
	Date          time.Time
	ActiveMinutes int
	Status        string
	DebtMinutes   int
	Description   string
}
