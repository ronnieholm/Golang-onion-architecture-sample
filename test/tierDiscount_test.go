package currency_test

import (
	"context"
	"testing"

	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"github.com/stretchr/testify/suite"
	"pgregory.net/rapid"
)

type TierDiscountTests struct {
	suite.Suite
	ctx        context.Context
	config     *infrastructure.Config
	clock      *SwitchableClock
	dispatcher infrastructure.Dispatcher
}

func (tdt *TierDiscountTests) SetupSuite() {
	tdt.ctx = context.Background()
	tdt.config = loadConfig()
	tdt.clock = &SwitchableClock{}
	tdt.dispatcher = infrastructure.NewDispatcher(tdt.ctx, *config, infrastructure.WithClock(tdt.clock))
}

func (tdt *TierDiscountTests) TearDownSuite() {
	tdt.dispatcher.Close()
}

func (tdt *TierDiscountTests) cleanUp() {
	resetDB(tdt.ctx, tdt.dispatcher.PgxPool)
}

func (ct *TierDiscountTests) TestCreateTierDiscountValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
		// fx := CreateCurrencyValidGen().Draw(t, "fx")
		// ct.setupCurrency(t, fx)

		// _, err := ct.dispatcher.CreateCurrency(ct.ctx, fx.CreateCurrency)
		// require.NoError(t, err)

		// c, err := ct.dispatcher.GetCurrency(ct.ctx, fx.GetCurrecyByCode)
		// require.NoError(t, err)
		// assert.Equal(t, fx.CreateCurrency.ID, c.ID)
		// assert.Equal(t, fx.CreateCurrency.Code, c.Code.V())
		// Assume system fields such as CreatedAt, UpdatedAt, Version are
		// correct and focus on the domain.
	})
}

func (ct *TierDiscountTests) TestCreateTierDiscountDuplicateIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestCreateTierDiscountUniqueFromValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestCreateTierDiscountUniqueFromInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestCreateTierDiscountFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestUpdateTierDiscountValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestUpdateTierDiscountIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestUpdateTierDiscountUnchangedInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestUpdateTierDiscountFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestRemoveTierDiscountValid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestRemoveTierDiscountIDInvalid() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func (ct *TierDiscountTests) TestRemoveTierDiscountFromPolicy() {
	rapid.Check(ct.T(), func(t *rapid.T) {
		ct.cleanUp()
	})
}

func TestTierDiscount(t *testing.T) {
	suite.Run(t, new(TierDiscountTests))
}
