package test

import (
	"context"
	"testing"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
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
	clock      *SwitchableClock
	dispatcher infrastructure.Dispatcher
}

func (td *CurrencyTests) SetupSuite() {
	td.ctx = context.Background()
	td.config = loadConfig()
	td.clock = &SwitchableClock{}
	td.dispatcher = infrastructure.NewDispatcher(td.ctx, *config, infrastructure.WithClock(td.clock))
}

func (td *CurrencyTests) TearDownSuite() {
	td.dispatcher.Close()
}

func (td *CurrencyTests) cleanUp() {
	resetDB(td.ctx, td.dispatcher.PgxPool)
}

func (td *CurrencyTests) setup(t *rapid.T, fx CreateCurrencyValidFixture) {
	td.clock.Current = fx.Clock
	_, err := td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)
	require.NoError(t, err)
}

func (td *CurrencyTests) TestCreateCurrencyValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyValidGen().Draw(t, "fx")

		td.setup(t, fx)

		c, err := td.dispatcher.GetCurrency(td.ctx, fx.GetCurrecy)
		require.NoError(t, err)
		assert.Equal(t, fx.CreateCurrency.ID, c.ID)
		assert.Equal(t, fx.CreateCurrency.Code, c.Code)
	})
}

func (td *CurrencyTests) TestCreateCurrencyDuplicateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyDuplicateIDInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)

		_, err := td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.ID.String(), e.FieldValues["ID"])
	})
}

func (td *CurrencyTests) TestCreateCurrencyDuplicateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyDuplicateCodeInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)

		_, err := td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestAddExchangeRateFromDateBoundary() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := AddExchangeRateFromDateBoundaryGen().Draw(t, "fx")
		td.setup(t, fx.Base)

		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrecy)
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

func (td *CurrencyTests) TestAddExchangeRateDuplicateFromInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := AddExchangeRateDuplicateFromInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate1)
		require.NoError(t, err)

		_, err = td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate2)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.AddExchangeRate2.From.String(), e.FieldValues["From"])
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateFromDateBoundary() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateFromDateBoundaryGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrecy)
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

func (td *CurrencyTests) TestUpdateExchangeRateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateIDInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.UpdateExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateCodeInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.UpdateExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateUnchangedInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateUnchangedInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.DomainError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, core.CurrencyUpdateRequiresChange, e.Code)
	})
}

func (td *CurrencyTests) TestRemoveCurrencyFromDateBoundary() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveCurrencyFromDateBoundaryGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		if fx.AddExchangeRate != nil {
			_, err := td.dispatcher.AddExchangeRate(td.ctx, *fx.AddExchangeRate)
			require.NoError(t, err)
			td.clock.Current = *fx.RemoveClock
		}

		_, err := td.dispatcher.RemoveCurrency(td.ctx, fx.RemoveCurrency)

		if fx.ShouldPass {
			require.NoError(t, err)
			_, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrecy)
			require.Error(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (td *CurrencyTests) TestRemoveCurrencyCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveCurrencyCodeInvalidGen().Draw(t, "fx")

		_, err := td.dispatcher.RemoveCurrency(td.ctx, fx.RemoveCurrencyCommand)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateFromDateBoundary() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateFromDateBoundaryGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)
		td.clock.Current = fx.RemoveClock

		_, err = td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
			c, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrecy)
			require.NoError(t, err)
			require.Len(t, c.ExchangeRates, 0)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateCodeInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		require.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.RemoveExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateIDInvalidGen().Draw(t, "fx")
		td.setup(t, fx.Base)
		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)
		require.NoError(t, err)

		_, err = td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

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
