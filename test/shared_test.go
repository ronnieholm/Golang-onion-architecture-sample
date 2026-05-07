package currency_test

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
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
	"DELETE FROM tier_discount",
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
