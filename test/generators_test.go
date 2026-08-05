package currency_test

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
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

// MapKeyGen generates a key from a non-empty map. It's handy when a map is used
// to represent an enum. In that case case the map's key is the enum's value
// while the map's value may be struct{}.
//
// Because key order is unstable across map instances, to guarantee the same
// outcome with the same seed requires a comparator function.
func MapKeyGen[K comparable, V any](m map[K]V, cmp func(K, K) int) *rapid.Generator[K] {
	if len(m) == 0 {
		panic("cannot draw from empty map")
	}

	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, cmp)
	return rapid.Custom(func(t *rapid.T) K {
		key := rapid.SampledFrom(keys).Draw(t, "key")
		return key
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

// FakeClockGen generates a clock whose current time is within the operating
// interval of the service. The operating interval is a narrower interval than
// that of domain values to allow for domain values to be smaller or larger than
// current time. For instance, space is guaranteed for an exchange rate whose
// from field is before or after current time.
func FakeClockGen() *rapid.Generator[core.Clock] {
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

// DateBetweenGen generates a Date between min and max, inclusive.
func DateBetweenGen(min, max core.Date) *rapid.Generator[core.Date] {
	// Drawing a date between min and max by ranging over the Unix timestamp
	// interval would result in min date being drawn too often because the
	// non-date components of time would be chopped off.
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
		codes := rapid.SliceOfNDistinct(
			CurrencyCodeGen(),
			/* min */ 1 /* max */, 2, // TOOD(rh): why not 0 to 1?
			func(s string) any { return s },
		).Draw(t, "codes")

		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		create.Code = codes[len(codes)-1]

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

type CreateCurrencyDuplicateInvalidFixture struct {
	Base           CreateCurrencyValidFixture
	CreateCurrency core.CreateCurrencyCommand
}

func CreateCurrencyDuplicateInvalidGen() *rapid.Generator[CreateCurrencyDuplicateInvalidFixture] {
	// Ensure IDs match, but other fields don't. It proves that a conflict error
	// is triggered specifically by the ID.
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateInvalidFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")

		// Mutate all other fields except ID.
		// TODO(rh): Maybe only randomly mutate?
		code := CurrencyCodeGen().
			Filter(func(c string) bool { return c != base.CreateCurrency.Code }).
			Draw(t, "code")
		create := core.CreateCurrencyCommand{
			ID:   base.CreateCurrency.ID,
			Code: code,
		}

		return CreateCurrencyDuplicateInvalidFixture{
			Base:           base,
			CreateCurrency: create,
		}
	})
}

func CreateCurrencyDuplicateCodeInvalidGen() *rapid.Generator[CreateCurrencyDuplicateInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateInvalidFixture {
		base := CreateCurrencyValidGen().Draw(t, "base")
		create := CreateCurrencyCommandGen().Draw(t, "create")

		// If any, mutate all other fields except ID and Code here.
		create.Code = base.CreateCurrency.Code

		return CreateCurrencyDuplicateInvalidFixture{
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
		panic("after must be after high from")
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

type AddExchangeRateUniqueFromValidFixture = struct {
	Clock            core.Clock
	CreateCurrency   core.CreateCurrencyCommand
	AddExchangeRates []core.AddExchangeRateCommand
	GetCurrency      core.GetCurrencyQuery
	Expected         map[uuid.UUID]core.AddExchangeRateCommand
}

func AddExchangeRateUniqueFromValidGen() *rapid.Generator[AddExchangeRateUniqueFromValidFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateUniqueFromValidFixture {
		clock := FakeClockGen().Draw(t, "clock")
		create := CreateCurrencyCommandGen().Draw(t, "currency")
		uniqueDates := rapid.SliceOfNDistinct(
			ExchangeRateFromAfterGen(clock.Today()),
			/* min */ 1 /* max */, 3,
			func(d core.Date) any { return d },
		).Draw(t, "unique_dates")

		// Separate arrange from assert part by separate collections. The slice
		// representation stays true to date generation order and simplifies
		// drawing an item from the slice in consuming generators. The map
		// representation avoids an O(n) lookup per item during assert.
		adds := make([]core.AddExchangeRateCommand, len(uniqueDates))
		expected := make(map[uuid.UUID]core.AddExchangeRateCommand, len(uniqueDates))
		for i, date := range uniqueDates {
			add := AddExchangeRateCommandGen().Draw(t, fmt.Sprintf("add_%d", i))
			add.Code = create.Code
			add.From = date
			adds[i] = add
			expected[add.ID] = add
		}

		get := core.GetCurrencyQuery{
			Code: create.Code,
		}

		return AddExchangeRateUniqueFromValidFixture{
			Clock:            clock,
			CreateCurrency:   create,
			AddExchangeRates: adds,
			GetCurrency:      get,
			Expected:         expected,
		}
	})
}

type AddExchangeRateFromPolicyFixture struct {
	Base            AddExchangeRateUniqueFromValidFixture
	AddExchangeRate core.AddExchangeRateCommand
	ShouldPass      bool
}

func AddExchangeRateFromPolicyGen() *rapid.Generator[AddExchangeRateFromPolicyFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateFromPolicyFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		today := base.Clock.Today()

		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)
		offsets := make(map[int]bool, len(base.Expected))
		for _, v := range base.Expected {
			offset := v.From.DaysBetween(today)
			offsets[offset] = true
		}

		offset := rapid.IntRange(-min, max).
			Filter(func(i int) bool { return !offsets[i] }).
			Draw(t, "offset")

		add := AddExchangeRateCommandGen().Draw(t, "add")
		add.Code = base.CreateCurrency.Code
		add.From = today.AddDate(0, 0, offset)

		return AddExchangeRateFromPolicyFixture{
			Base:            base,
			AddExchangeRate: add,
			ShouldPass:      offset > 0}
	})
}

type AddExchangeRateDuplicateFromInvalidFixture = struct {
	Base                     AddExchangeRateUniqueFromValidFixture
	AddExchangeRateDuplicate core.AddExchangeRateCommand
}

func AddExchangeRateDuplicateFromInvalidGen() *rapid.Generator[AddExchangeRateDuplicateFromInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateDuplicateFromInvalidFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add1 := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add1")

		rate := add1.Rate
		if !rapid.Bool().Draw(t, "same_rate") {
			rate = ExchangeRateRateGen().Draw(t, "new_rate")
		}

		add2 := core.AddExchangeRateCommand{
			ID:   UUIDGen().Draw(t, "id"),
			Code: base.CreateCurrency.Code,
			Rate: rate,
			From: add1.From,
		}

		return AddExchangeRateDuplicateFromInvalidFixture{
			Base:                     base,
			AddExchangeRateDuplicate: add2}
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

type UpdateExchangeRateFixture = struct {
	Base               AddExchangeRateUniqueFromValidFixture
	UpdateExchangeRate core.UpdateExchangeRateCommand
}

func UpdateExchangeRateValidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		today := base.Clock.Today()
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")
		update := core.UpdateExchangeRateCommand(add)

		fieldsUpdate := rapid.SliceOfNDistinct(
			rapid.IntRange(0, 1), 1, 2, func(i int) int { return i }).
			Draw(t, "fields_update")
		from := make(map[core.Date]bool, len(base.Expected))
		for _, v := range base.Expected {
			from[v.From] = true
		}

		for _, idx := range fieldsUpdate {
			switch idx {
			case 0:
				update.Rate = ExchangeRateRateGen().
					Filter(func(r float64) bool { return r != add.Rate }).
					Draw(t, "changeRate")
			case 1:
				update.From = ExchangeRateFromAfterGen(today).
					Filter(func(d core.Date) bool { return !from[d] }).
					Draw(t, "changeFrom")
			}
		}

		return UpdateExchangeRateFixture{
			Base:               base,
			UpdateExchangeRate: update,
		}
	})
}

func UpdateExchangeRateCodeInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		// Add noise to ensure SQL query or domain logic isn't accidentally
		// doing a LIMIT 1 or missing a WHERE clause on UPDATE.
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")

		otherCode := CurrencyCodeGen().
			Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
			Draw(t, "other_code")
		update := core.UpdateExchangeRateCommand{
			ID:   add.ID,
			Code: otherCode,
			Rate: add.Rate,
			From: add.From,
		}

		return UpdateExchangeRateFixture{
			Base:               base,
			UpdateExchangeRate: update,
		}
	})
}

func UpdateExchangeRateIDInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")

		otherID := UUIDGen().
			Filter(func(id uuid.UUID) bool { return id != add.ID }).
			Draw(t, "other_id")
		update := core.UpdateExchangeRateCommand{
			ID:   otherID,
			Code: add.Code,
			Rate: add.Rate,
			From: add.From,
		}

		return UpdateExchangeRateFixture{
			Base:               base,
			UpdateExchangeRate: update,
		}
	})
}

func UpdateExchangeRateUnchangedInvalidGen() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")
		update := core.UpdateExchangeRateCommand(add)

		return UpdateExchangeRateFixture{
			Base:               base,
			UpdateExchangeRate: update}
	})
}

type UpdateExchangeRateFromPolicyFixture struct {
	Base               AddExchangeRateUniqueFromValidFixture
	UpdateExchangeRate core.UpdateExchangeRateCommand
	// The goal is to specify the invariant. Therefore generate both valid and
	// invalid cases within the same fixture.
	ShouldPass bool
}

func UpdateExchangeRateFromPolicyGen() *rapid.Generator[UpdateExchangeRateFromPolicyFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFromPolicyFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		today := base.Clock.Today()
		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)

		offsets := make(map[int]bool, len(base.Expected))
		for _, v := range base.Expected {
			offset := v.From.DaysBetween(today)
			offsets[offset] = true
		}
		offset := rapid.IntRange(-min, max).
			Filter(func(i int) bool { return !offsets[i] }).
			Draw(t, "offset")

		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")
		update := core.UpdateExchangeRateCommand{
			ID:   add.ID,
			Code: add.Code,
			Rate: ExchangeRateRateGen().Draw(t, "rate"),
			From: today.AddDate(0, 0, offset),
		}

		return UpdateExchangeRateFromPolicyFixture{
			Base:               base,
			UpdateExchangeRate: update,
			ShouldPass:         offset > 0}
	})
}

type RemoveCurrencyFixture struct {
	Base                  AddExchangeRateUniqueFromValidFixture
	RemoveCurrencyCommand core.RemoveCurrencyCommand
}

func RemoveCurrencyValidGen() *rapid.Generator[RemoveCurrencyFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		remove := core.RemoveCurrencyCommand{
			Code: base.CreateCurrency.Code,
		}
		return RemoveCurrencyFixture{
			Base:                  base,
			RemoveCurrencyCommand: remove,
		}
	})
}

func RemoveCurrencyInvalidGen() *rapid.Generator[RemoveCurrencyFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		remove := core.RemoveCurrencyCommand{
			Code: CurrencyCodeGen().
				Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
				Draw(t, "code"),
		}
		return RemoveCurrencyFixture{
			Base:                  base,
			RemoveCurrencyCommand: remove,
		}
	})
}

type RemoveCurrencyChildFromPolicyPolicyFixture struct {
	Base           RemoveExchangeRateFromPolicyFixture
	RemoveCurrency core.RemoveCurrencyCommand
}

func RemoveCurrencyChildFromPolicyGen() *rapid.Generator[RemoveCurrencyChildFromPolicyPolicyFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyChildFromPolicyPolicyFixture {
		base := RemoveExchangeRateFromPolicyGen().Draw(t, "base")
		remove := core.RemoveCurrencyCommand{
			Code: base.RemoveExchangeRate.Code,
		}

		return RemoveCurrencyChildFromPolicyPolicyFixture{
			Base:           base,
			RemoveCurrency: remove,
		}
	})
}

type RemoveExchangeRateValidFixture struct {
	Base                AddExchangeRateUniqueFromValidFixture
	RemoveExchangeRate  core.RemoveExchangeRateCommand
	WantExchangeRateIds map[uuid.UUID]struct{}
}

func RemoveExchangeRateValidGen() *rapid.Generator[RemoveExchangeRateValidFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateValidFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")

		remove := core.RemoveExchangeRateCommand{
			ID:   add.ID,
			Code: base.CreateCurrency.Code,
		}

		wantIds := make(map[uuid.UUID]struct{}, len(base.Expected)-1)
		for k := range base.Expected {
			if k != add.ID {
				wantIds[k] = struct{}{}
			}
		}

		return RemoveExchangeRateValidFixture{
			Base:                base,
			RemoveExchangeRate:  remove,
			WantExchangeRateIds: wantIds,
		}
	})
}

type RemoveExchangeRateFixture = struct {
	Base               AddExchangeRateUniqueFromValidFixture
	RemoveExchangeRate core.RemoveExchangeRateCommand
}

func RemoveExchangeRateCodeInvalidGen() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")

		otherCode := CurrencyCodeGen().
			Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
			Draw(t, "other_code")
		remove := core.RemoveExchangeRateCommand{
			ID:   add.ID,
			Code: otherCode,
		}

		return RemoveExchangeRateFixture{
			Base:               base,
			RemoveExchangeRate: remove,
		}
	})
}

func RemoveExchangeRateIDInvalidGen() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")
		add := rapid.SampledFrom(base.AddExchangeRates).Draw(t, "add")

		otherID := UUIDGen().
			Filter(func(id uuid.UUID) bool { return id != add.ID }).
			Draw(t, "other_id")
		remove := core.RemoveExchangeRateCommand{
			ID:   otherID,
			Code: add.Code,
		}

		return RemoveExchangeRateFixture{
			Base:               base,
			RemoveExchangeRate: remove,
		}
	})
}

type RemoveExchangeRateFromPolicyFixture struct {
	Base               AddExchangeRateUniqueFromValidFixture
	RemoveClock        core.Clock
	RemoveExchangeRate core.RemoveExchangeRateCommand
	ShouldPass         bool
}

func RemoveExchangeRateFromPolicyGen() *rapid.Generator[RemoveExchangeRateFromPolicyFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFromPolicyFixture {
		base := AddExchangeRateUniqueFromValidGen().Draw(t, "base")

		sorted := slices.Clone(base.AddExchangeRates)
		slices.SortFunc(sorted, func(a, b core.AddExchangeRateCommand) int {
			return a.From.Compare(b.From)
		})

		idx := rapid.IntRange(0, len(sorted)-1).Draw(t, "idx")
		exchangeRate := sorted[idx]

		min := core.ExchangeRateFromMin
		max := core.ExchangeRateFromMax
		shouldPass := rapid.Bool().Draw(t, "should_pass")
		if shouldPass {
			max = base.Clock.Today()
		} else {
			min = exchangeRate.From
		}

		removeClock := rapid.
			Map(DateBetweenGen(min, max), func(d core.Date) core.Clock { return &FakeClock{Now: d.Time} }).
			Draw(t, "remove_clock")
		remove := core.RemoveExchangeRateCommand{
			ID:   exchangeRate.ID,
			Code: base.CreateCurrency.Code,
		}

		return RemoveExchangeRateFromPolicyFixture{
			Base:               base,
			RemoveClock:        removeClock,
			RemoveExchangeRate: remove,
			ShouldPass:         shouldPass,
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
