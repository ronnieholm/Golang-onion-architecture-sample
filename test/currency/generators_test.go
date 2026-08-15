package currency

import (
	"math"
	"strings"
	"uuid"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/test"
	"pgregory.net/rapid"
)

func genCurrencyCode() *rapid.Generator[string] {
	return test.GenMapKey(core.CurrencyCodes, strings.Compare)
}

func CreateCurrencyCommandGen() *rapid.Generator[core.CreateCurrencyCommand] {
	return rapid.Custom(func(t *rapid.T) core.CreateCurrencyCommand {
		return core.CreateCurrencyCommand{
			ID:   test.GenUUID().Draw(t, "id"),
			Code: genCurrencyCode().Draw(t, "code"),
		}
	})
}

type CreateCurrencyValidFixture struct {
	Clock          core.Clock
	CreateCurrency core.CreateCurrencyCommand
	GetCurrecy     core.GetCurrencyQuery
}

func genCreateCurrencyValid() *rapid.Generator[CreateCurrencyValidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyValidFixture {
		clock := test.GenFakeClock().Draw(t, "clock")
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

func genCreateCurrencyDuplicateIDInvalid() *rapid.Generator[CreateCurrencyDuplicateIDInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateIDInvalidFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		create.ID = base.CreateCurrency.ID
		return CreateCurrencyDuplicateIDInvalidFixture{
			Base:           base,
			CreateCurrency: create,
		}
	})
}

func genCreateCurrencyDuplicateCodeInvalid() *rapid.Generator[CreateCurrencyDuplicateIDInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateCurrencyDuplicateIDInvalidFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		create := CreateCurrencyCommandGen().Draw(t, "create_currency")
		create.Code = base.CreateCurrency.Code
		return CreateCurrencyDuplicateIDInvalidFixture{
			Base:           base,
			CreateCurrency: create,
		}
	})
}

func genExchangeRateRate() *rapid.Generator[float64] {
	return rapid.Custom(func(t *rapid.T) float64 {
		decimalPlaces := rapid.
			IntRange(core.ExchangeRateDecimalPlacesMin, core.ExchangeRateDecimalPlacesMax).
			Draw(t, "decimal_places")
		rate := rapid.Float64Range(core.ExchangeRateMin, core.ExchangeRateMax).Draw(t, "rate")
		ratio := math.Pow10(decimalPlaces)
		return math.Round(rate*ratio) / ratio
	})
}

func genExchangeRateFrom() *rapid.Generator[core.Date] {
	return rapid.Custom(func(t *rapid.T) core.Date {
		return test.GenDateBetween(core.ExchangeRateFromMin, core.ExchangeRateFromMax).Draw(t, "from")
	})
}

func genExchangeRateFromAfter(after core.Date) *rapid.Generator[core.Date] {
	if after.After(core.ExchangeRateFromMax) {
		panic("after must be after from max")
	}
	d := after.AddDate(0, 0, 1)
	return rapid.Custom(func(t *rapid.T) core.Date {
		return test.GenDateBetween(d, core.ExchangeRateFromMax).Draw(t, "from")
	})
}

func genAddExchangeRateCommand() *rapid.Generator[core.AddExchangeRateCommand] {
	return rapid.Custom(func(t *rapid.T) core.AddExchangeRateCommand {
		return core.AddExchangeRateCommand{
			ID:   test.GenUUID().Draw(t, "id"),
			Code: genCurrencyCode().Draw(t, "code"),
			Rate: genExchangeRateRate().Draw(t, "rate"),
			From: genExchangeRateFrom().Draw(t, "from"),
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

func genAddExchangeRateFromDateBoundary() *rapid.Generator[AddExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateFromDateBoundaryFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(-min, max).Draw(t, "offset")

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
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

func genAddExchangeRateDuplicateFromInvalid() *rapid.Generator[AddExchangeRateDuplicateFromInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) AddExchangeRateDuplicateFromInvalidFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(1, max).Draw(t, "offset")

		add1 := genAddExchangeRateCommand().Draw(t, "add_exchange_rate_1")
		add1.Code = base.CreateCurrency.Code
		add1.From = today.AddDate(0, 0, offset)

		add2 := genAddExchangeRateCommand().Draw(t, "add_exchange_rate_2")
		add2.Code = add1.Code
		add2.From = add1.From

		return AddExchangeRateDuplicateFromInvalidFixture{
			Base:             base,
			AddExchangeRate1: add1,
			AddExchangeRate2: add2,
		}
	})
}

func genUpdateExchangeRateCommand() *rapid.Generator[core.UpdateExchangeRateCommand] {
	return rapid.Custom(func(t *rapid.T) core.UpdateExchangeRateCommand {
		return core.UpdateExchangeRateCommand{
			ID:   test.GenUUID().Draw(t, "id"),
			Code: genCurrencyCode().Draw(t, "code"),
			Rate: genExchangeRateRate().Draw(t, "rate"),
			From: genExchangeRateFrom().Draw(t, "from"),
		}
	})
}

type UpdateExchangeRateFromDateBoundaryFixture = struct {
	Base               CreateCurrencyValidFixture
	AddExchangeRate    core.AddExchangeRateCommand
	UpdateExchangeRate core.UpdateExchangeRateCommand
	ShouldPass         bool
}

func genUpdateExchangeRateFromDateBoundary() *rapid.Generator[UpdateExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFromDateBoundaryFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		min := today.DaysBetween(core.ExchangeRateFromMin)
		max := today.DaysBetween(core.ExchangeRateFromMax)
		offset := rapid.IntRange(-min, max).Draw(t, "offset")

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		update := genUpdateExchangeRateCommand().Draw(t, "update_exchange_rate")
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

func genUpdateExchangeRateIDInvalid() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		update := genUpdateExchangeRateCommand().Draw(t, "update_exchange_rate")
		update.Code = base.CreateCurrency.Code
		update.From = add.From

		return UpdateExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
		}
	})
}

func genUpdateExchangeRateCodeInvalid() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		update := genUpdateExchangeRateCommand().Draw(t, "update_exchange_rate")
		update.ID = add.ID
		update.Code = genCurrencyCode().
			Filter(func(s string) bool { return s != base.CreateCurrency.Code }).
			Draw(t, "other_code")

		return UpdateExchangeRateFixture{
			Base:               base,
			AddExchangeRate:    add,
			UpdateExchangeRate: update,
		}
	})
}

func genUpdateExchangeRateUnchangedInvalid() *rapid.Generator[UpdateExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) UpdateExchangeRateFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

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

func genRemoveCurrencyFromDateBoundary() *rapid.Generator[RemoveCurrencyFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyFromDateBoundaryFixture {
		base := genCreateCurrencyValid().Draw(t, "base")

		var add *core.AddExchangeRateCommand
		var removeClock *core.Clock
		var shouldPass = true
		hasExchangeRate := rapid.Bool().Draw(t, "has_exchange_rate")
		if hasExchangeRate {
			today := base.Clock.Today()

			cmd := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
			cmd.Code = base.CreateCurrency.Code
			cmd.From = genExchangeRateFromAfter(today).Draw(t, "from")
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
				Map(test.GenDateBetween(min, max), func(d core.Date) core.Clock { return &test.FakeClock{Now: d.Time} }).
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

func genRemoveCurrencyCommand() *rapid.Generator[core.RemoveCurrencyCommand] {
	return rapid.Custom(func(t *rapid.T) core.RemoveCurrencyCommand {
		return core.RemoveCurrencyCommand{
			Code: genCurrencyCode().Draw(t, "code"),
		}
	})
}

type RemoveCurrencyCodeInvalidFixture struct {
	RemoveCurrencyCommand core.RemoveCurrencyCommand
}

func genRemoveCurrencyCodeInvalid() *rapid.Generator[RemoveCurrencyCodeInvalidFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveCurrencyCodeInvalidFixture {
		remove := genRemoveCurrencyCommand().Draw(t, "remove_currency")
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

func genRemoveExchangeRateFromDateBoundary() *rapid.Generator[RemoveExchangeRateFromDateBoundaryFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFromDateBoundaryFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		shouldPass := rapid.Bool().Draw(t, "should_pass")
		min := core.ExchangeRateFromMin
		max := core.ExchangeRateFromMax
		if shouldPass {
			max = base.Clock.Today()
		} else {
			min = add.From
		}
		removeClock := rapid.
			Map(test.GenDateBetween(min, max), func(d core.Date) core.Clock { return &test.FakeClock{Now: d.Time} }).
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

func genRemoveExchangeRateCodeInvalid() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		otherCode := genCurrencyCode().
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

func genRemoveExchangeRateIDInvalid() *rapid.Generator[RemoveExchangeRateFixture] {
	return rapid.Custom(func(t *rapid.T) RemoveExchangeRateFixture {
		base := genCreateCurrencyValid().Draw(t, "base")
		today := base.Clock.Today()

		add := genAddExchangeRateCommand().Draw(t, "add_exchange_rate")
		add.Code = base.CreateCurrency.Code
		add.From = genExchangeRateFromAfter(today).Draw(t, "from")

		otherID := test.GenUUID().
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
