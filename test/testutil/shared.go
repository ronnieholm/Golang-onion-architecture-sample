package testutil

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"pgregory.net/rapid"
)

// The testutil folder acts as a safe container for sharing code between test
// packages. As long as production code never imports testutil, none of its
// contents or third-party dependencies like rapid will leak into the production
// deployment.

var (
	once   sync.Once
	Config *infrastructure.Config
)

func LoadConfig() *infrastructure.Config {
	once.Do(func() {
		// Config file is read from parent directory rather than current
		// directory because the caller is a subfolder under test.
		c, err := infrastructure.LoadConfig("../testdata/service.json")
		if err != nil {
			panic(err)
		}
		Config = &c
	})
	return Config
}

// TODO(rh): any benefit in batching delete statements?
// DELETE statements must come in reverse dependency order.
var sql = []string{
	"DELETE FROM domain_event",
	"DELETE FROM exchange_rate",
	"DELETE FROM currency",
	"DELETE FROM tier_discount",
}

func ResetDB(ctx context.Context, pool *pgxpool.Pool) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		panic(err)
	}
	for _, s := range sql {
		_, err = tx.Exec(ctx, s)
		if err != nil {
			panic(err)
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
}

// Fakes.

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

// Generators.

// Prefer naming generators genX over xGen for better grouping in suggestions.

// GenUUID generates a version 4 compliant UUID.
func GenUUID() *rapid.Generator[uuid.UUID] {
	return rapid.Custom(func(t *rapid.T) uuid.UUID {
		uuidBytes := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "uuid_bytes")
		var id uuid.UUID
		copy(id[:], uuidBytes)
		id[6] = (id[6] & 0x0f) | 0x40
		id[8] = (id[8] & 0x3f) | 0x80
		return id
	})
}

// GenMapKey generates a key from a non-empty map. It's useful when a map
// represents an enum in which case the map's key is the enum's value and the
// map's value may be struct{}.
//
// Because key order is unstable across map instances, to guarantee the same
// outcome with the same seed, a comparator function is requires.
func GenMapKey[K comparable, V any](m map[K]V, cmp func(K, K) int) *rapid.Generator[K] {
	if len(m) == 0 {
		panic("cannot draw from empty map")
	}
	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, cmp)
	return rapid.Custom(func(t *rapid.T) K {
		return rapid.SampledFrom(keys).Draw(t, "key")
	})
}

// GenTimeRange generates a time between min and max, inclusive.
func GenTimeRange(min, max time.Time) *rapid.Generator[time.Time] {
	minUnix := min.Unix()
	maxUnix := max.Unix()
	return rapid.Custom(func(t *rapid.T) time.Time {
		unix := rapid.Int64Range(minUnix, maxUnix).Draw(t, "unix")
		return time.Unix(unix, 0).In(time.UTC)
	})
}

// GenFakeClock generates a clock whose time is within the service's operating
// interval. At both ends the operating interval is narrower than the domain
// interval so domain may be smaller or larger than clock, e.g., to allow for
// generating an exchange rate from before or after current time.
func GenFakeClock() *rapid.Generator[core.Clock] {
	// Leave one year to each side of full clock range.
	min := core.MinClock.AddDate(1, 0, 0)
	max := core.MaxClock.AddDate(-1, 0, 0)
	if min.After(max) {
		panic("min must come before max")
	}
	return rapid.Custom(func(t *rapid.T) core.Clock {
		now := GenTimeRange(min, max).Draw(t, "now")
		fc := &FakeClock{
			Now: now,
		}
		return fc
	})
}

// GenDateBetween generates a date between min and max dates, inclusive.
func GenDateBetween(min, max core.Date) *rapid.Generator[core.Date] {
	// Drawing a date between min and max by ranging over the Unix timestamp
	// interval would result in min date being drawn too often because the
	// non-date components of time would dominate, yet be chopped off.
	days := min.DaysBetween(max)
	return rapid.Custom(func(t *rapid.T) core.Date {
		offset := rapid.IntRange(0, days).Draw(t, "offset")
		return core.DateFromTime(min.Time.AddDate(0, 0, offset))
	})
}
