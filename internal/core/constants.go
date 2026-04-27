package core

import "time"

// If the domain had been modelled strictly with value objects, these constants
// would go inside a value object's validate function. But for most domains,
// adhering to such rigor is overkill. It results in significant boilerplate of
// little value. Instead such configuration is kept in a central location.

var (
	MinClock = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	MaxClock = time.Date(2034, 12, 31, 23, 59, 59, 0, time.UTC)

	MinExchangeRateFrom = NewDate(2024, 1, 1)
	MaxExchangeRateFrom = NewDate(2034, 12, 31)
)

const (
	MinExchangeRate              float64 = 1.
	MaxExchangeRate              float64 = 100.
	MinExchangeRateDecimalPlaces         = 0
	MaxExchangeRateDecimalPlaces         = 6
)
