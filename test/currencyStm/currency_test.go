package currencyStm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
	"github.com/ronnieholm/resellerloyalty/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

type CurrencyStateMachine struct {
	ctx        context.Context
	config     *infrastructure.Config
	clock      *testutil.SwitchableClock // TODO(rh): Each action should advance the clock.
	dispatcher infrastructure.Dispatcher

	// statistics
	// maps for attempted an actual execution of each actions

	currencies map[string]core.CurrencyResponse
	freeCodes  map[string]struct{}
}

func NewCurrencyStateMachine(t *rapid.T) *CurrencyStateMachine {
	m := &CurrencyStateMachine{
		ctx:        context.Background(),
		config:     testutil.LoadConfig(),
		clock:      &testutil.SwitchableClock{},
		currencies: make(map[string]core.CurrencyResponse),
		freeCodes:  make(map[string]struct{}),
	}

	m.dispatcher = infrastructure.NewDispatcher(m.ctx, *testutil.Config, infrastructure.WithClock(m.clock))
	testutil.ResetDB(m.ctx, m.dispatcher.PgxPool)
	m.clock.Current = testutil.GenFakeClock().Draw(t, "clock")

	for cc := range core.CurrencyCodes {
		m.freeCodes[cc] = struct{}{}
	}

	return m
}

func (m *CurrencyStateMachine) TearDown() {
	m.dispatcher.Close()
}

func (m *CurrencyStateMachine) CreateCurrency(t *rapid.T) {
	if len(m.freeCodes) == 0 {
		t.Skip("no free currency code")
	}

	create := genCreateCurrencyCommand().Draw(t, "create_currency")
	create.Code = testutil.GenMapKey(m.freeCodes, strings.Compare).Draw(t, "free_code")
	get := core.GetCurrencyQuery{
		Code: create.Code,
	}
	delete(m.freeCodes, create.Code)

	m.dispatcher.CreateCurrency(m.ctx, create)
	res, err := m.dispatcher.GetCurrency(m.ctx, get)
	require.NoError(t, err)
	m.currencies[create.Code] = *res
}

func (m *CurrencyStateMachine) RemoveCurrency(t *rapid.T) {
	if len(m.currencies) == 0 {
		t.Skip("no currency")
	}

	code := testutil.GenMapKey(m.currencies, strings.Compare).Draw(t, "code")
	m.freeCodes[code] = struct{}{}
	delete(m.currencies, code)
	remove := core.RemoveCurrencyCommand{Code: code}
	get := core.GetCurrencyQuery{
		Code: code,
	}

	m.dispatcher.RemoveCurrency(m.ctx, remove)
	_, err := m.dispatcher.GetCurrency(m.ctx, get)

	var e *core.NotFoundError
	require.ErrorAs(t, err, &e)
	assert.Equal(t, "Currency", e.Entity)
	require.Equal(t, 1, len(e.FieldValues))
	assert.Equal(t, code, e.FieldValues["Code"])
}

func (m *CurrencyStateMachine) Check(t *rapid.T) {
	// Contains invariant checking code.
	// fmt.Println("check")

	// Query domain events from #x to avoid querying all events and replaying for every check (need repository for domain events?)
}

func (m *CurrencyStateMachine) PrintStats() {

}

func TestCurrencyStateMachine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := NewCurrencyStateMachine(t)
		defer m.TearDown()

		t.Repeat(rapid.StateMachineActions(m))
		fmt.Print("--------------------------------------\n")
	})
}
