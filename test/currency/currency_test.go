package currency_test

import (
	"context"
	"testing"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"github.com/ronnieholm/resellerloyalty/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"pgregory.net/rapid"
)

// Assume system fields CreatedAt, UpdatedAt, Version are correct and focus
// testing on the domain.

type CurrencyTests struct {
	suite.Suite
	ctx        context.Context
	config     *infrastructure.Config
	clock      *testutil.SwitchableClock
	dispatcher infrastructure.Dispatcher
}

func (ct *CurrencyTests) SetupSuite() {
	ct.ctx = context.Background()
	ct.config = testutil.LoadConfig()
	ct.clock = &testutil.SwitchableClock{}
	ct.dispatcher = infrastructure.NewDispatcher(ct.ctx, *testutil.Config, infrastructure.WithClock(ct.clock))
}

func (ct *CurrencyTests) TearDownSuite() {
	ct.dispatcher.Close()
}

func (ct *CurrencyTests) cleanUp() {
	testutil.ResetDB(ct.ctx, ct.dispatcher.PgxPool)
}

func (ct *CurrencyTests) setup(t *rapid.T, fx CreateCurrencyValidFixture) {
	ct.clock.Current = fx.Clock
	_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)
	require.NoError(t, err)
}

func (ct *CurrencyTests) TestCreateCurrencyValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genCreateCurrencyValid().Draw(t, "fx")

		ct.setup(t, fx)

		c, err := ct.dispatcher.GetCurrency(ct.ctx, fx.GetCurrecy)
		require.NoError(t, err)
		assert.Equal(t, fx.CreateCurrency.ID, c.ID)
		assert.Equal(t, fx.CreateCurrency.Code, c.Code)
	})
}

func (ct *CurrencyTests) TestCreateCurrencyDuplicateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genCreateCurrencyDuplicateIDInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)

		_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.ID.String(), e.FieldValues["ID"])
	})
}

func (ct *CurrencyTests) TestCreateCurrencyDuplicateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genCreateCurrencyDuplicateCodeInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)

		_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestAddExchangeRateFromDateBoundary() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genAddExchangeRateFromDateBoundary().Draw(t, "fx")
		ct.setup(t, fx.Base)

		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := ct.dispatcher.GetCurrency(ct.ctx, fx.Base.GetCurrecy)
			require.NoError(t, err)
			require.Len(t, c.ExchangeRates, 1)
			e := c.ExchangeRates[0]
			assert.Equal(t, fx.AddExchangeRate.ID, e.ID)
			assert.Equal(t, fx.AddExchangeRate.Code, fx.Base.CreateCurrency.Code)
			assert.Equal(t, fx.AddExchangeRate.From, e.From)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyAddRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestAddExchangeRateDuplicateFromInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genAddExchangeRateDuplicateFromInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate1)
		require.NoError(t, err)

		_, err = ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate2)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.AddExchangeRate2.From.String(), e.FieldValues["From"])
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateFromDateBoundary() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genUpdateExchangeRateFromDateBoundary().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := ct.dispatcher.GetCurrency(ct.ctx, fx.Base.GetCurrecy)
			require.NoError(t, err)
			e := c.ExchangeRates[0]
			assert.Equal(t, fx.UpdateExchangeRate.ID, e.ID)
			assert.Equal(t, fx.UpdateExchangeRate.Code, fx.Base.CreateCurrency.Code)
			assert.Equal(t, fx.UpdateExchangeRate.From, e.From)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyUpdateRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genUpdateExchangeRateIDInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.UpdateExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genUpdateExchangeRateCodeInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.UpdateExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateUnchangedInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genUpdateExchangeRateUnchangedInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.DomainError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, core.CurrencyUpdateRequiresChange, e.Code)
	})
}

func (ct *CurrencyTests) TestRemoveCurrencyFromDateBoundary() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genRemoveCurrencyFromDateBoundary().Draw(t, "fx")
		ct.setup(t, fx.Base)
		if fx.AddExchangeRate != nil {
			_, err := ct.dispatcher.AddExchangeRate(ct.ctx, *fx.AddExchangeRate)
			require.NoError(t, err)
			ct.clock.Current = *fx.RemoveClock
		}

		_, err := ct.dispatcher.RemoveCurrency(ct.ctx, fx.RemoveCurrency)

		if fx.ShouldPass {
			require.NoError(t, err)
			_, err := ct.dispatcher.GetCurrency(ct.ctx, fx.Base.GetCurrecy)
			require.Error(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestRemoveCurrencyCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genRemoveCurrencyCodeInvalid().Draw(t, "fx")

		_, err := ct.dispatcher.RemoveCurrency(ct.ctx, fx.RemoveCurrencyCommand)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateFromDateBoundary() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genRemoveExchangeRateFromDateBoundary().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)
		ct.clock.Current = fx.RemoveClock

		_, err = ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := ct.dispatcher.GetCurrency(ct.ctx, fx.Base.GetCurrecy)
			require.NoError(t, err)
			require.Len(t, c.ExchangeRates, 0)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genRemoveExchangeRateCodeInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.RemoveExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := genRemoveExchangeRateIDInvalid().Draw(t, "fx")
		ct.setup(t, fx.Base)
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.RemoveExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func TestCurrency(t *testing.T) {
	suite.Run(t, new(CurrencyTests))
}
