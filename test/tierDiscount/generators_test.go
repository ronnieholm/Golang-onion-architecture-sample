package tierDiscount_test

import (
	"math"
	"slices"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/test/testutil"
	"pgregory.net/rapid"
)

func genDiscountPercentages() *rapid.Generator[core.DiscountPercentagesInput] {
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

func genTierDiscountFrom() *rapid.Generator[core.Date] {
	return rapid.Custom(func(t *rapid.T) core.Date {
		return testutil.GenDateBetween(core.TierDiscountFromMin, core.TierDiscountFromMin).Draw(t, "from")
	})
}

func genTierDiscountAfter(after core.Date) *rapid.Generator[core.Date] {
	if after.After(core.TierDiscountFromMax) {
		panic("after must be after high from")
	}
	d := after.AddDate(0, 0, 1)

	return rapid.Custom(func(t *rapid.T) core.Date {
		return testutil.GenDateBetween(d, core.TierDiscountFromMax).Draw(t, "from")
	})
}

func genCreateTierDiscountCommand() *rapid.Generator[core.CreateTierDiscountCommand] {
	return rapid.Custom(func(t *rapid.T) core.CreateTierDiscountCommand {
		return core.CreateTierDiscountCommand{
			ID:          testutil.GenUUID().Draw(t, "id"),
			Percentages: genDiscountPercentages().Draw(t, "percentages"),
			From:        genTierDiscountFrom().Draw(t, "from"),
		}
	})
}

type CreateTierDiscountValidFixture struct {
	Clock              core.Clock
	CreateTierDiscount core.CreateTierDiscountCommand
	GetTierDiscount    core.GetTierDiscountQuery
}

func genCreateTierDiscountValid() *rapid.Generator[CreateTierDiscountValidFixture] {
	return rapid.Custom(func(t *rapid.T) CreateTierDiscountValidFixture {
		clock := testutil.GenFakeClock().Draw(t, "clock")
		uniqueDates := rapid.SliceOfNDistinct(
			genTierDiscountAfter(clock.Today()),
			/* min */ 1 /* max */, 2,
			func(d core.Date) any { return d },
		).Draw(t, "unique_dates")

		create := genCreateTierDiscountCommand().Draw(t, "create")
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

func genCreateTierDiscountDuplicateIDInvalid() *rapid.Generator[CreateTierDiscountDuplicateInvalidFixture] {
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
