package infrastructure

import (
	"time"

	"github.com/ronnieholm/resellerloyalty/internal/core"
)

type RealTimeClock struct {
}

func (c *RealTimeClock) NowUTC() time.Time {
	return time.Now().UTC()
}

func (c *RealTimeClock) Today() core.Date {
	return core.DateFromTime(c.NowUTC())
}
