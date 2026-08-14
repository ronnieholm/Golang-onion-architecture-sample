package test

import (
	"maps"
	"math"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"pgregory.net/rapid"
)

// UUIDGen generates a version 4 compliant UUID.
func UUIDGen() *rapid.Generator[uuid.UUID] {
	return rapid.Custom(func(t *rapid.T) uuid.UUID {
		uuidBytes := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "uuid_bytes")
		var id uuid.UUID
		copy(id[:], uuidBytes)
		id[6] = (id[6] & 0x0f) | 0x40
		id[8] = (id[8] & 0x3f) | 0x80
		return id
	})
}

// MapKeyGen generates a key from a non-empty map. It's useful when a map
// represents an enum in which case the map's key is the enum's value and the
// map's value may be struct{}.
//
// Because key order is unstable across map instances, to guarantee the same
// outcome with the same seed, a comparator function is requires.
func MapKeyGen[K comparable, V any](m map[K]V, cmp func(K, K) int) *rapid.Generator[K] {
	if len(m) == 0 {
		panic("cannot draw from empty map")
	}
	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, cmp)
	return rapid.Custom(func(t *rapid.T) K {
		return rapid.SampledFrom(keys).Draw(t, "key")
	})
}

// TimeRangeGen generates a time between min and max, inclusive.
func TimeRangeGen(min, max time.Time) *rapid.Generator[time.Time] {
	minUnix := min.Unix()
	maxUnix := max.Unix()
	return rapid.Custom(func(t *rapid.T) time.Time {
		unix := rapid.Int64Range(minUnix, maxUnix).Draw(t, "unix")
		return time.Unix(unix, 0).In(time.UTC)
	})
}

// FakeClockGen generates a clock whose time is within the service's operating
// interval. At both ends the operating interval is narrower than the domain
// interval so domain may be smaller or larger than clock, e.g., to allow for
// generating an exchange rate from before or after current time.
func FakeClockGen() *rapid.Generator[core.Clock] {
	// Leave one year to each side of full clock range.
	min := core.MinClock.AddDate(1, 0, 0)
	max := core.MaxClock.AddDate(-1, 0, 0)
	if min.After(max) {
		panic("min must come before max")
	}
	return rapid.Custom(func(t *rapid.T) core.Clock {
		now := TimeRangeGen(min, max).Draw(t, "now")
		fc := &FakeClock{
			Now: now,
		}
		return fc
	})
}

// DateBetweenGen generates a date between min and max dates, inclusive.
func DateBetweenGen(min, max core.Date) *rapid.Generator[core.Date] {
	// Drawing a date between min and max by ranging over the Unix timestamp
	// interval would result in min date being drawn too often because the
	// non-date components of time would dominate, yet be chopped off.
	days := min.DaysBetween(max)
	return rapid.Custom(func(t *rapid.T) core.Date {
		offset := rapid.IntRange(0, days).Draw(t, "offset")
		return core.DateFromTime(min.Time.AddDate(0, 0, offset))
	})
}

func CurrencyCodeGen() *rapid.Generator[string] {
	return MapKeyGen(core.CurrencyCodes, strings.Compare)
}

func CreateCurrencyCommandGen() *rapid.Generator[core.CreateCurrencyCommand] {
	return rapid.Custom(func(t *rapid.T) core.CreateCurrencyCommand {
		return core.CreateCurrencyCommand{
			ID:   UUIDGen().Draw(t, "id"),
			Code: CurrencyCodeGen().Draw(t, "code"),
		}
	})
}

type CreateCurrencyValidFixture struct {
	Clock          core.Clock
	CreateCurrency core.CreateCurrencyCommand
	GetCurrecy     core.GetCurrencyQuery
}

func CreateCurrencyValidGen() *rapid.Generator[CreateCurrencyValidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyValidFixture {
		clock := FakeClockGen().Draw(t, "clock")
		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		get := core.GetCurrencyQuery{
			Code: create.Code,
		}
		return CreateCurrencyValidFixture{
			Clock:          clock,
			CreateCurrency: create,
			GetCurrecy:     get,
		}
	})
}

type CreateCurrencyDuplicateIDInvalidFixture struct {
	Base           CreateCurrencyValidFixture
	CreateCurrency core.CreateCurrencyCommand
}

func CreateCurrencyDuplicateIDInvalidGen() *rapid.Generator[CreateCurrencyDuplicateIDInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateIDInvalidFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		create.ID = base.CreateCurrency.ID
		return CreateCurrencyDuplicateIDInvalidFixture{
			Base:           base,
			CreateCurrency: create,
		}
	})
}

func CreateCurrencyDuplicateCodeInvalidGen() *rapid.Generator[CreateCurrencyDuplicateIDInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateIDInvalidFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		create.Code = base.CreateCurrency.Code
		return CreateCurrencyDuplicateIDInvalidFixture{
			Base:           base,
			CreateCurrency: create,
		}
	})
}

func ExchangeRateRateGen() *rapid.Generator[float64] {
	return rapid.Custom(func(t *rapid.T) float64 {
		decimalPlaces := rapid.
			IntRange(core.ExchangeRateDecimalPlacesMin, core.ExchangeRateDecimalPlacesMax).
			Draw(t, "decimal_places")
		rate := rapid.Float64Range(core.ExchangeRateMin, core.ExchangeRateMax).Draw(t, "rate")
		ratio := math.Pow10(decimalPlaces)
		return math.Round(rate*ratio) / ratio
	})
}

func ExchangeRateFromGen() *rapid.Generator[core.Date] {
	return rapid.Custom(func(t *rapid.T) core.Date {
		return DateBetweenGen(core.ExchangeRateFromMin, core.ExchangeRateFromMax).Draw(t, "from")
	})
}

func ExchangeRateFromAfterGen(after core.Date) *rapid.Generator[core.Date] {
	if after.After(core.ExchangeRateFromMax) {
		panic("after must be after from max")
	}
	d := after.AddDate(0, 0, 1)
	return rapid.Custom(func(t *rapid.T) core.Date {
		return DateBetweenGen(d, core.ExchangeRateFromMax).Draw(t, "from")
	})
}

func AddExchangeRateCommandGen() *rapid.Generator[core.AddExchangeRateCommand] {
	return rapid.Custom(func(t *rapid.T) core.AddExchangeRateCommand {
		return core.AddExchangeRateCommand{
			ID:   UUIDGen().Draw(t, "id"),
			Code: CurrencyCodeGen().Draw(t, "code"),
			Rate: ExchangeRateRateGen().Draw(t, "rate"),
			From: ExchangeRateFromGen().Draw(t, "from"),
		}
	})
}

type AddExchangeRateFromDateBoundaryFixture = struct {
	Base            CreateCurrencyValidFixture
	AddExchangeRate core.AddExchangeRateCommand
	// The goal is to specify the invariant. Therefore include both valid and
	// invalid cases within the same fixture.
	ShouldPass bool
}

func AddExchangeRateFromDateBoundaryGen() *rapid.Generator[AddExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateFromDateBoundaryFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(-min, max).Draw(t, "offset")

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = today.AddDate(0, 0, offset)

		return AddExchangeRateFromDateBoundaryFixture{
			Base:            base,
			AddExchangeRate: add,
			ShouldPass:      offset > 0,
		}
	})
}

type AddExchangeRateDuplicateFromInvalidFixture = struct {
	Base             CreateCurrencyValidFixture
	AddExchangeRate1 core.AddExchangeRateCommand
	AddExchangeRate2 core.AddExchangeRateCommand
}

func AddExchangeRateDuplicateFromInvalidGen() *rapid.Generator[AddExchangeRateDuplicateFromInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateDuplicateFromInvalidFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(1, max).Draw(t, "offset")

		add1 := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate_1")
		add1.Code = base.CreateCurrency.Code
		add1.From = today.AddDate(0, 0, offset)

		add2 := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate_2")
		add2.Code = add1.Code
		add2.From = add1.From

		return AddExchangeRateDuplicateFromInvalidFixture{
			Base:             base,
			AddExchangeRate1: add1,
			AddExchangeRate2: add2,
		}
	})
}

func UpdateExchangeRateCommandGen() *rapid.Generator[core.UpdateExchangeRateCommand] {
	return rapid.Custom(func(t *rapid.T) core.UpdateExchangeRateCommand {
		return core.UpdateExchangeRateCommand{
			ID:   UUIDGen().Draw(t, "id"),
			Code: CurrencyCodeGen().Draw(t, "code"),
			Rate: ExchangeRateRateGen().Draw(t, "rate"),
			From: ExchangeRateFromGen().Draw(t, "from"),
		}
	})
}

type UpdateExchangeRateFromDateBoundaryFixture = struct {
	Base               CreateCurrencyValidFixture
	AddExchangeRate    core.AddExchangeRateCommand
	UpdateExchangeRate core.UpdateExchangeRateCommand
	ShouldPass         bool
}

func UpdateExchangeRateFromDateBoundaryGen() *rapid.Generator[UpdateExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFromDateBoundaryFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(-min, max).Draw(t, "offset")

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		update := UpdateExchangeRateCommandGen().Draw(t, "update_exchange_rate")
		update.ID = add.ID
		update.Code = add.Code
		update.From = today.AddDate(0, 0, offset)

		return UpdateExchangeRateFromDateBoundaryFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
			ShouldPass:         offset > 0,
		}
	})
}

type UpdateExchangeRateFixture = struct {
	Base               CreateCurrencyValidFixture
	AddExchangeRate    core.AddExchangeRateCommand
	UpdateExchangeRate core.UpdateExchangeRateCommand
}

func UpdateExchangeRateIDInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		update := UpdateExchangeRateCommandGen().Draw(t, "update_exchange_rate")
		update.Code = base.CreateCurrency.Code
		update.From = add.From

		return UpdateExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
		}
	})
}

func UpdateExchangeRateCodeInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		update := UpdateExchangeRateCommandGen().Draw(t, "update_exchange_rate")
		update.ID = add.ID
		update.Code = CurrencyCodeGen().
			Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
			Draw(t, "other_code")

		return UpdateExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
		}
	})
}

func UpdateExchangeRateUnchangedInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		update := core.UpdateExchangeRateCommand{
			ID:   add.ID,
			Code: base.CreateCurrency.Code,
			Rate: add.Rate,
			From: add.From,
		}

		return UpdateExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
		}
	})
}

type RemoveCurrencyFromDateBoundaryFixture struct {
	Base            CreateCurrencyValidFixture
	AddExchangeRate *core.AddExchangeRateCommand
	RemoveClock     *core.Clock
	RemoveCurrency  core.RemoveCurrencyCommand
	ShouldPass      bool
}

func RemoveCurrencyFromDateBoundaryGen() *rapid.Generator[RemoveCurrencyFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyFromDateBoundaryFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")

		var add *core.AddExchangeRateCommand
		var removeClock *core.Clock
		var shouldPass = true
		hasExchangeRate := rapid.Bool().Draw(t, "has_exchange_rate")
		if hasExchangeRate {
			today := base.Clock.Today()

			cmd := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
			cmd.Code = base.CreateCurrency.Code
			cmd.From = ExchangeRateFromAfterGen(today).Draw(t, "from")
			add = &cmd

			min := core.ExchangeRateFromMin
			max := core.ExchangeRateFromMax
			shouldPass = rapid.Bool().Draw(t, "should_pass")
			if shouldPass {
				max = base.Clock.Today()
			} else {
				min = add.From
			}
			clock := rapid.
				Map(DateBetweenGen(min, max), func(d core.Date) core.Clock { return &FakeClock{Now: d.Time} }).
				Draw(t, "remove_clock")
			removeClock = &clock
		}

		remove := core.RemoveCurrencyCommand{
			Code: base.CreateCurrency.Code,
		}

		return RemoveCurrencyFromDateBoundaryFixture{
			Base:            base,
			AddExchangeRate: add,
			RemoveClock:     removeClock,
			RemoveCurrency:  remove,
			ShouldPass:      shouldPass,
		}
	})
}

func RemoveCurrencyCommandGen() *rapid.Generator[core.RemoveCurrencyCommand] {
	return rapid.Custom(func(t *rapid.T) core.RemoveCurrencyCommand {
		return core.RemoveCurrencyCommand{
			Code: CurrencyCodeGen().Draw(t, "code"),
		}
	})
}

type RemoveCurrencyCodeInvalidFixture struct {
	RemoveCurrencyCommand core.RemoveCurrencyCommand
}

func RemoveCurrencyCodeInvalidGen() *rapid.Generator[RemoveCurrencyCodeInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyCodeInvalidFixture {
		remove := RemoveCurrencyCommandGen().Draw(t, "remove_currency")
		return RemoveCurrencyCodeInvalidFixture{
			RemoveCurrencyCommand: remove,
		}
	})
}

type RemoveExchangeRateFromDateBoundaryFixture struct {
	Base               CreateCurrencyValidFixture
	AddExchangeRate    core.AddExchangeRateCommand
	RemoveClock        core.Clock
	RemoveExchangeRate core.RemoveExchangeRateCommand
	ShouldPass         bool
}

func RemoveExchangeRateFromDateBoundaryGen() *rapid.Generator[RemoveExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFromDateBoundaryFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		shouldPass := rapid.Bool().Draw(t, "should_pass")
		min := core.ExchangeRateFromMin
		max := core.ExchangeRateFromMax
		if shouldPass {
			max = base.Clock.Today()
		} else {
			min = add.From
		}
		removeClock := rapid.
			Map(DateBetweenGen(min, max), func(d core.Date) core.Clock { return &FakeClock{Now: d.Time} }).
			Draw(t, "remove_clock")
		remove := core.RemoveExchangeRateCommand{
			ID:   add.ID,
			Code: base.CreateCurrency.Code,
		}

		return RemoveExchangeRateFromDateBoundaryFixture{
			Base:               base,
			AddExchangeRate:    add,
			RemoveClock:        removeClock,
			RemoveExchangeRate: remove,
			ShouldPass:         shouldPass,
		}
	})
}

type RemoveExchangeRateFixture = struct {
	Base               CreateCurrencyValidFixture
	AddExchangeRate    core.AddExchangeRateCommand
	RemoveExchangeRate core.RemoveExchangeRateCommand
}

func RemoveExchangeRateCodeInvalidGen() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		otherCode := CurrencyCodeGen().
			Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
			Draw(t, "other_code")
		remove := core.RemoveExchangeRateCommand{
			ID:   add.ID,
			Code: otherCode,
		}

		return RemoveExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			RemoveExchangeRate: remove,
		}
	})
}

func RemoveExchangeRateIDInvalidGen() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		today := base.Clock.Today()

		add := AddExchangeRateCommandGen().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = ExchangeRateFromAfterGen(today).Draw(t, "from")

		otherID := UUIDGen().
			Filter(func(id uuid.UUID) bool { return id != add.ID }).
			Draw(t, "other_id")
		remove := core.RemoveExchangeRateCommand{
			ID:   otherID,
			Code: add.Code,
		}

		return RemoveExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			RemoveExchangeRate: remove,
		}
	})
}

// TierDiscount

func DiscountPercentagesGen() *rapid.Generator[core.DiscountPercentagesInput] {
	return rapid.Custom(func(t *rapid.T) core.DiscountPercentagesInput {
		dp := rapid.SliceOfN(
			rapid.IntRange(core.TierDiscountPercentageDecimalPlacesMin, core.TierDiscountPercentageDecimalPlacesMax),
			3, 3).Draw(t, "decimal_places")
		ps := rapid.SliceOfNDistinct(
			rapid.Float64Range(core.TierDiscountPercentageMin, core.TierDiscountPercentageMax),
			3, 3,
			func(f float64) any { return f },
		).Draw(t, "percentages")

		p := [3]float64{}
		for i := range p {
			ratio := math.Pow10(dp[i])
			p[i] = math.Round(ps[i]*ratio) / ratio
		}
		slices.Sort(p[:])
		return core.DiscountPercentagesInput{
			Authorized: p[0],
			Advanced:   p[1],
			Premier:    p[2],
		}
	})
}

func TierDiscountFromGen() *rapid.Generator[core.Date] {
	return rapid.Custom(func(t *rapid.T) core.Date {
		return DateBetweenGen(core.TierDiscountFromMin, core.TierDiscountFromMin).Draw(t, "from")
	})
}

func TierDiscountAfterGen(after core.Date) *rapid.Generator[core.Date] {
	if after.After(core.TierDiscountFromMax) {
		panic("after must be after high from")
	}
	d := after.AddDate(0, 0, 1)

	return rapid.Custom(func(t *rapid.T) core.Date {
		return DateBetweenGen(d, core.TierDiscountFromMax).Draw(t, "from")
	})
}

func CreateTierDiscountCommandGen() *rapid.Generator[core.CreateTierDiscountCommand] {
	return rapid.Custom(func(t *rapid.T) core.CreateTierDiscountCommand {
		return core.CreateTierDiscountCommand{
			ID:          UUIDGen().Draw(t, "id"),
			Percentages: DiscountPercentagesGen().Draw(t, "percentages"),
			From:        TierDiscountFromGen().Draw(t, "from"),
		}
	})
}

type CreateTierDiscountValidFixture struct {
	Clock              core.Clock
	CreateTierDiscount core.CreateTierDiscountCommand
	GetTierDiscount    core.GetTierDiscountQuery
}

func CreateTierDiscountValidGen() *rapid.Generator[CreateTierDiscountValidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateTierDiscountValidFixture {
		clock := FakeClockGen().Draw(t, "clock")
		uniqueDates := rapid.SliceOfNDistinct(
			TierDiscountAfterGen(clock.Today()),
			/* min */ 1 /* max */, 2,
			func(d core.Date) any { return d },
		).Draw(t, "unique_dates")

		create := CreateTierDiscountCommandGen().Draw(t, "create")
		create.From = uniqueDates[len(uniqueDates)-1]

		get := core.GetTierDiscountQuery{
			ID: create.ID,
		}

		return CreateTierDiscountValidFixture{
			Clock:              clock,
			CreateTierDiscount: create,
			GetTierDiscount:    get,
		}
	})
}

type CreateTierDiscountDuplicateInvalidFixture struct {
	Base               CreateTierDiscountValidFixture
	CreateTierDiscount core.CreateTierDiscountCommand
}

func CreateTierDiscountDuplicateIDInvalidGen() *rapid.Generator[CreateTierDiscountDuplicateInvalidFixture] {
	// return rapid.Custom(func(t *rapid.T) CreateTierDiscountDuplicateInvalidFixture {
	// 	base := CreateTierDiscountValidGen().Draw(t, "base")

	// 	// Mutate all other fields except ID.

	// 	// Percentages
	// 	// From (unique)

	// 	code := CurrencyCodeGen().
	// 		Filter(func(c string) bool { return c != base.CreateCurrency.Code }).
	// 		Draw(t, "code")
	// 	create := core.CreateTierDiscountCommand{
	// 		ID:   base.Create.ID,
	// 		Code: code,
	// 	}

	// 	return CreateTierDiscountDuplicateInvalidFixture{
	// 		Base:               base,
	// 		CreateTierDiscount: create,
	// 	}
	// })
	return nil
}

// UpdateTierDiscountCommand
// RemoveTierDiscountCommand
// GetTierDiscountQuery
