package currency_test

import (
	"fmt"
	"time"

	"github.com/ronnieholm/resellerloyalty/internal/core"
)

type SwitchableClock struct {
	Current core.Clock
}

func (p *SwitchableClock) NowUTC() time.Time { return p.Current.NowUTC() }
func (p *SwitchableClock) Today() core.Date  { return p.Current.Today() }

type FakeClock struct {
	Now time.Time
}

func (fc *FakeClock) NowUTC() time.Time { return fc.Now }
func (fc *FakeClock) Today() core.Date  { return core.DateFromTime(fc.Now) }

// Called by Rapid to print a failed test result.
func (fc *FakeClock) GoString() string {
	if fc == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", fc.Now.Format(time.RFC3339))
}
