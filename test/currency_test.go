package currency_test

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

func (td *CurrencyTests) setupCurrency(t *rapid.T, fx CreateCurrencyValidFixture) {
	td.clock.Current = fx.Clock
	for _, cmd := range fx.Noise {
		_, err := td.dispatcher.CreateCurrency(td.ctx, cmd)
		require.NoError(t, err, "Failed on ID %s", cmd.ID.String())
	}
}

func (td *CurrencyTests) TestCreateCurrencyValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyValidGen().Draw(t, "fx")
		td.setupCurrency(t, fx)

		_, err := td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)
		require.NoError(t, err)

		c, err := td.dispatcher.GetCurrency(td.ctx, fx.GetCurrecy)
		require.NoError(t, err)
		assert.Equal(t, fx.CreateCurrency.ID, c.ID)
		assert.Equal(t, fx.CreateCurrency.Code, c.Code.V())
		// Assume system fields such as CreatedAt, UpdatedAt, Version are
		// correct and focus on the domain.
	})
}

func (td *CurrencyTests) TestCreateCurrencyDuplicateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyDuplicateInvalidGen().Draw(t, "fx")
		td.setupCurrency(t, fx.Base)
		_, err := td.dispatcher.CreateCurrency(td.ctx, fx.Base.CreateCurrency)
		require.NoError(t, err)

		_, err = td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.CreateCurrency.ID.String(), e.FieldValues["ID"])
	})
}

func (td *CurrencyTests) TestCreateCurrencyDuplicateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateCurrencyDuplicateCodeInvalidGen().Draw(t, "fx")
		td.setupCurrency(t, fx.Base)
		_, err := td.dispatcher.CreateCurrency(td.ctx, fx.Base.CreateCurrency)
		require.NoError(t, err)

		_, err = td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) setupExchangeRates(t *rapid.T, fx AddExchangeRateUniqueFromValidFixture) {
	td.clock.Current = fx.Clock
	_, err := td.dispatcher.CreateCurrency(td.ctx, fx.CreateCurrency)
	require.NoError(t, err)

	for _, cmd := range fx.AddExchangeRates {
		_, err := td.dispatcher.AddExchangeRate(td.ctx, cmd)
		require.NoError(t, err, "Failed on ID %s", cmd.ID.String())
	}
}

func (td *CurrencyTests) TestAddExchangeRateUniqueFromValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := AddExchangeRateUniqueFromValidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx)

		c, err := td.dispatcher.GetCurrency(td.ctx, fx.GetCurrency)
		require.NoError(t, err)
		require.Len(t, fx.Expected, len(c.ExchangeRates))

		for _, exchangeRate := range c.ExchangeRates {
			expectedRate, exists := fx.Expected[exchangeRate.ID]
			assert.True(t, exists, "Found unexpected from: %s", exchangeRate.From)
			assert.Equal(t, expectedRate.From, exchangeRate.From.V())
			assert.Equal(t, expectedRate.Rate, exchangeRate.Rate.V())
		}
	})
}

func (td *CurrencyTests) TestAddExchangeRateDuplicateFromInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := AddExchangeRateDuplicateFromInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRateDuplicate)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.AddExchangeRateDuplicate.From.String(), e.FieldValues["From"])
	})
}

func (td *CurrencyTests) TestAddExchangeRateFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := AddExchangeRateFromPolicyGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.AddExchangeRate(td.ctx, fx.AddExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyAddRequiresFutureFrom, e.Code)
		}
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateValidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)
		require.NoError(t, err)

		currency, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrency)
		require.NoError(t, err)

		for _, er := range currency.ExchangeRates {
			expected, exists := fx.Base.Expected[er.ID]
			assert.True(t, exists, "Found unexpected from: %s", er.From)
			if er.ID == fx.UpdateExchangeRate.ID {
				assert.Equal(t, fx.UpdateExchangeRate.From, er.From.V())
				assert.Equal(t, fx.UpdateExchangeRate.Rate, er.Rate.V())
			} else {
				assert.Equal(t, expected.From, er.From.V())
				assert.Equal(t, expected.Rate, er.Rate.V())
			}
		}
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateCodeInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.UpdateExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateIDInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, fx.UpdateExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateUnchangedInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateUnchangedInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		var e *core.DomainError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, core.CurrencyUpdateRequiresChange, e.Code)
	})
}

func (td *CurrencyTests) TestUpdateExchangeRateFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := UpdateExchangeRateFromPolicyGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.UpdateExchangeRate(td.ctx, fx.UpdateExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyUpdateRequiresFutureFrom, e.Code)
		}
	})
}

func (td *CurrencyTests) TestRemoveCurrencyValid() { // TODO(rh): combine TestRemoveCurrencyValid and TestRemoveCurrencyInvalid
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveCurrencyValidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.RemoveCurrency(td.ctx, fx.RemoveCurrencyCommand)

		require.NoError(t, err)
		_, err = td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrency)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestRemoveCurrencyInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveCurrencyInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.RemoveCurrency(td.ctx, fx.RemoveCurrencyCommand)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestRemoveCurrencyChildFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveCurrencyChildFromPolicyGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base.Base)
		td.clock.Current = fx.Base.RemoveClock

		_, err := td.dispatcher.RemoveCurrency(td.ctx, fx.RemoveCurrency)

		if fx.Base.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateValidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		require.NoError(t, err)
		c, err := td.dispatcher.GetCurrency(td.ctx, fx.Base.GetCurrency)
		require.NoError(t, err)
		require.Len(t, c.ExchangeRates, len(fx.WantExchangeRateIds))

		for _, e := range c.ExchangeRates {
			_, exists := fx.WantExchangeRateIds[e.ID]
			assert.True(t, exists, "Found unexpected id: %s", e.ID.String())
		}
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateCodeInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateCodeInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateIDInvalidGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)

		_, err := td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, fx.RemoveExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (td *CurrencyTests) TestRemoveExchangeRateFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := RemoveExchangeRateFromPolicyGen().Draw(t, "fx")
		td.setupExchangeRates(t, fx.Base)
		td.clock.Current = fx.RemoveClock

		_, err := td.dispatcher.RemoveExchangeRate(td.ctx, fx.RemoveExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func TestCurrency(t *testing.T) {
	suite.Run(t, new(CurrencyTests))
}
