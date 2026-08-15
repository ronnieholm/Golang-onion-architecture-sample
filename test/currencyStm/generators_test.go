package currencyStm

import (
	"strings"

	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/test/testutil"
	"pgregory.net/rapid"
)

func genCurrencyCode() *rapid.Generator[string] {
	return testutil.GenMapKey(core.CurrencyCodes, strings.Compare)
}

func genCreateCurrencyCommand() *rapid.Generator[core.CreateCurrencyCommand] {
	return rapid.Custom(func(t *rapid.T) core.CreateCurrencyCommand {
		return core.CreateCurrencyCommand{
			ID:   testutil.GenUUID().Draw(t, "id"),
			Code: genCurrencyCode().Draw(t, "code"),
		}
	})
}
