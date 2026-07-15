package currency_test

import (
	"context"
	"testing"

	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (td *TierDiscountTests) SetupSuite() {
	td.ctx = context.Background()
	td.config = loadConfig()
	td.clock = &SwitchableClock{}
	td.dispatcher = infrastructure.NewDispatcher(td.ctx, *config, infrastructure.WithClock(td.clock))
}

func (td *TierDiscountTests) TearDownSuite() {
	td.dispatcher.Close()
}

func (td *TierDiscountTests) cleanUp() {
	resetDB(td.ctx, td.dispatcher.PgxPool)
}

func (td *TierDiscountTests) setupTierDiscount(t *rapid.T, fx CreateTierDiscountValidFixture) {
	td.clock.Current = fx.Clock
	for _, cmd := range fx.Noise {
		_, err := td.dispatcher.CreateTierDiscount(td.ctx, cmd)
		require.NoError(t, err, "Failed on ID %s", cmd.ID.String())
	}
}

func (td *TierDiscountTests) TestCreateTierDiscountValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		fx := CreateTierDiscountValidGen().Draw(t, "fx")
		td.setupTierDiscount(t, fx)

		_, err := td.dispatcher.CreateTierDiscount(td.ctx, fx.CreateTierDiscount)
		require.NoError(t, err)

		t_, err := td.dispatcher.GetTierDiscount(td.ctx, fx.GetTierDiscount)
		require.NoError(t, err)
		assert.Equal(t, fx.CreateTierDiscount.ID, t_.ID)
		assert.Equal(t, fx.CreateTierDiscount.Percentages.Authorized, t_.Percentages.Authorized())
		assert.Equal(t, fx.CreateTierDiscount.Percentages.Advanced, t_.Percentages.Advanced())
		assert.Equal(t, fx.CreateTierDiscount.Percentages.Premier, t_.Percentages.Premier())
		assert.Equal(t, fx.CreateTierDiscount.From, t_.From.V())
	})
}

func (td *TierDiscountTests) TestCreateTierDiscountDuplicateIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
		// Create CreateTierDiscountDuplicateIDInvalidGen()
		//   Generate noise
		//     Percentages and from should be different across noise
		//   Generate duplicate by drawing a new CreateXCommandGen() and setting ID from noise
	})
}

func (td *TierDiscountTests) TestCreateTierDiscountUniqueFromValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestCreateTierDiscountUniqueFromInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestCreateTierDiscountFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestUpdateTierDiscountValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestUpdateTierDiscountIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestUpdateTierDiscountUnchangedInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestUpdateTierDiscountFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestRemoveTierDiscountValid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestRemoveTierDiscountIDInvalid() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func (td *TierDiscountTests) TestRemoveTierDiscountFromPolicy() {
	rapid.Check(td.T(), func(t *rapid.T) {
		td.cleanUp()
	})
}

func TestTierDiscount(t *testing.T) {
	suite.Run(t, new(TierDiscountTests))
}
