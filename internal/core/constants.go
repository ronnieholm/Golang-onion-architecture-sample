package core

import "time"

var (
	MinClock = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	MaxClock = time.Date(2034, 12, 31, 23, 59, 59, 0, time.UTC)
)
