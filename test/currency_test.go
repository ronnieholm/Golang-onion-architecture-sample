package currency_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"pgregory.net/rapid"
)

var (
	once   sync.Once
	config *infrastructure.Config
)

func loadConfig() *infrastructure.Config {
	once.Do(func() {
		c, err := infrastructure.LoadConfig("./testdata/service.json")
		if err != nil {
			panic(err)
		}
		config = &c
	})
	return config
}

// DELETE statements must come in reverse dependency order.
var sql = []string{
	"DELETE FROM domain_event",
	"DELETE FROM exchange_rate",
	"DELETE FROM currency",
}

func resetDB(ctx context.Context, pool *pgxpool.Pool) {
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

type CurrencyTests struct {
	suite.Suite
	ctx        context.Context
	config     *infrastructure.Config
	clock      *SwitchableClock
	dispatcher infrastructure.Dispatcher
}

func (ct *CurrencyTests) SetupSuite() {
	ct.ctx = context.Background()
	ct.config = loadConfig()
	ct.clock = &SwitchableClock{}
	ct.dispatcher = infrastructure.NewDispatcher(ct.ctx, *config, infrastructure.WithClock(ct.clock))
}

func (ct *CurrencyTests) TearDownSuite() {
	ct.dispatcher.Close()
}

func (ct *CurrencyTests) cleanUp() {
	resetDB(ct.ctx, ct.dispatcher.PgxPool)
}

func (ct *CurrencyTests) setupCurrency(t *rapid.T, fx CreateCurrencyValidFixture) {
	ct.clock.Current = fx.Clock
	for _, cmd := range fx.Noise {
		_, err := ct.dispatcher.CreateCurrency(ct.ctx, cmd)
		require.NoError(t, err, "Failed on ID %s", cmd.ID.String())
	}
}

func (ct *CurrencyTests) TestCreateCurrencyValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := CreateCurrencyValidGen().Draw(t, "fx")
		ct.setupCurrency(t, fx)

		_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)
		require.NoError(t, err)

		c, err := ct.dispatcher.GetCurrencyByCode(ct.ctx, fx.GetCurrecyByCode)
		require.NoError(t, err)
		assert.Equal(t, fx.CreateCurrency.ID, c.ID)
		assert.Equal(t, fx.CreateCurrency.Code, c.Code)
		// Assume system fields such as CreatedAt, UpdatedAt, Version are
		// correct and focus on the domain.
	})
}

func (ct *CurrencyTests) TestCreateCurrencyDuplicateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := CreateCurrencyDuplicateIDInvalidGen().Draw(t, "fx")
		ct.setupCurrency(t, fx.Base)
		_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.Base.CreateCurrency)
		require.NoError(t, err)

		_, err = ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.CreateCurrency.ID.String(), e.FieldValues["ID"])
	})
}

func (ct *CurrencyTests) TestCreateCurrencyDuplicateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := CreateCurrencyDuplicateCodeInvalidGen().Draw(t, "fx")
		ct.setupCurrency(t, fx.Base)
		_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.Base.CreateCurrency)
		require.NoError(t, err)

		_, err = ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.CreateCurrency.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) setupExchangeRates(t *rapid.T, fx AddExchangeRateUniqueFromValidFixture) {
	ct.clock.Current = fx.Clock
	_, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)
	require.NoError(t, err)

	for _, cmd := range fx.AddExchangeRates {
		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, cmd)
		require.NoError(t, err, "Failed on ID %s", cmd.ID.String())
	}
}

func (ct *CurrencyTests) TestAddExchangeRateUniqueFromValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := AddExchangeRateUniqueFromValidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx)

		c, err := ct.dispatcher.GetCurrencyByCode(ct.ctx, fx.GetCurrencyByCode)
		require.NoError(t, err)
		require.Len(t, fx.Expected, len(c.ExchangeRates))

		for _, exchangeRate := range c.ExchangeRates {
			expectedRate, exists := fx.Expected[exchangeRate.ID]
			assert.True(t, exists, "Found unexpected from: %s", exchangeRate.From.String())
			assert.Equal(t, expectedRate.From, exchangeRate.From)
			assert.Equal(t, expectedRate.Rate, exchangeRate.Rate)
		}
	})
}

func (ct *CurrencyTests) TestAddExchangeRateDuplicateFromInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := AddExchangeRateDuplicateFromInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRateDuplicate)

		var e *core.ConflictError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, 1, len(e.FieldValues))
		assert.Equal(t, fx.AddExchangeRateDuplicate.From.String(), e.FieldValues["From"])
	})
}

func (ct *CurrencyTests) TestAddExchangeRateFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := AddExchangeRateFromPolicyGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.AddExchangeRate(ct.ctx, fx.AddExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyAddRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := UpdateExchangeRateValidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)
		require.NoError(t, err)

		currency, err := ct.dispatcher.GetCurrencyByCode(ct.ctx, fx.Base.GetCurrencyByCode)
		require.NoError(t, err)

		for _, er := range currency.ExchangeRates {
			expected, exists := fx.Base.Expected[er.ID]
			assert.True(t, exists, "Found unexpected from: %s", er.From.String())
			if er.ID == fx.UpdateExchangeRate.ID {
				assert.Equal(t, fx.UpdateExchangeRate.From, er.From)
				assert.Equal(t, fx.UpdateExchangeRate.Rate, er.Rate)
			} else {
				assert.Equal(t, expected.From, er.From)
				assert.Equal(t, expected.Rate, er.Rate)
			}
		}
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := UpdateExchangeRateCodeInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.UpdateExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := UpdateExchangeRateIDInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, fx.UpdateExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateUnchangedInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := UpdateExchangeRateUnchangedInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		var e *core.DomainError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, core.CurrencyUpdateRequiresChangedRate, e.Code)
	})
}

func (ct *CurrencyTests) TestUpdateExchangeRateFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := UpdateExchangeRateFromPolicyGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.UpdateExchangeRate(ct.ctx, fx.UpdateExchangeRate)

		if fx.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyUpdateRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestRemoveCurrencyValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveCurrencyValidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.RemoveCurrency(ct.ctx, fx.RemoveCurrencyCommand)

		require.NoError(t, err)
		_, err = ct.dispatcher.GetCurrencyByCode(ct.ctx, fx.Base.GetCurrencyByCode)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestRemoveCurrencyInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveCurrencyInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.RemoveCurrency(ct.ctx, fx.RemoveCurrencyCommand)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveCurrencyCommand.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestRemoveCurrencyChildFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveCurrencyChildFromPolicyGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base.Base)
		ct.clock.Current = fx.Base.RemoveClock

		_, err := ct.dispatcher.RemoveCurrency(ct.ctx, fx.RemoveCurrency)

		if fx.Base.ShouldPass {
			require.NoError(t, err)
		} else {
			var e *core.DomainError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, core.CurrencyRemoveRequiresFutureFrom, e.Code)
		}
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveExchangeRateValidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		require.NoError(t, err)
		c, err := ct.dispatcher.GetCurrencyByCode(ct.ctx, fx.Base.GetCurrencyByCode)
		require.NoError(t, err)
		require.Len(t, c.ExchangeRates, len(fx.WantExchangeRateIds))

		for _, e := range c.ExchangeRates {
			_, exists := fx.WantExchangeRateIds[e.ID]
			assert.True(t, exists, "Found unexpected id: %s", e.ID.String())
		}
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateCodeInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveExchangeRateCodeInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "Currency", e.Entity)
		assert.Equal(t, fx.RemoveExchangeRate.Code, e.FieldValues["Code"])
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveExchangeRateIDInvalidGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)

		_, err := ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

		var e *core.NotFoundError
		require.ErrorAs(t, err, &e)
		assert.Equal(t, "ExchangeRate", e.Entity)
		assert.Equal(t, fx.RemoveExchangeRate.ID.String(), e.FieldValues["ID"])
	})
}

func (ct *CurrencyTests) TestRemoveExchangeRateFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		fx := RemoveExchangeRateFromPolicyGen().Draw(t, "fx")
		ct.setupExchangeRates(t, fx.Base)
		ct.clock.Current = fx.RemoveClock

		_, err := ct.dispatcher.RemoveExchangeRate(ct.ctx, fx.RemoveExchangeRate)

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
