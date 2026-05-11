package infrastructure

// A PostgreSQL implementation of Store interfaces. In principle, the
// implementation could be with conceptually separate read and write databases
// or a combination of relational and document database features.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ronnieholm/resellerloyalty/internal/core"
)

// Currency

type currencyFlat struct {
	CID        uuid.UUID
	CCode      string
	CVersion   int32
	CCreatedAt time.Time
	CUpdatedAt *time.Time
	EID        *uuid.UUID
	ERate      *float64
	EFrom      *core.Date
	ECreatedAt *time.Time
	EUpdatedAt *time.Time
}

func (c currencyFlat) currency() *core.Currency {
	return &core.Currency{
		AggregateRoot: core.AggregateRoot{
			Version: c.CVersion,
			Entity: core.Entity{
				ID:        c.CID,
				CreatedAt: c.CCreatedAt,
				UpdatedAt: c.CUpdatedAt,
			},
		},
		Code: core.MustParseCurrencyCode(c.CCode),
	}
}

func (c currencyFlat) exchangeRate() *core.ExchangeRate {
	if c.EID != nil {
		return &core.ExchangeRate{
			Entity: core.Entity{
				ID:        *c.EID,
				CreatedAt: *c.ECreatedAt,
				UpdatedAt: c.EUpdatedAt,
			},
			Rate: core.MustParseRate(*c.ERate),
			From: core.MustParseExchangeRateFrom(*c.EFrom),
		}
	} else {
		return nil
	}
}

type PgCurrencyStore struct {
	Pool *pgxpool.Pool
}

func (cs PgCurrencyStore) ExistByID(ctx context.Context, id core.CurrencyID) (bool, error) {
	sql := "SELECT EXISTS (SELECT 1 FROM currency WHERE id = $1)"
	found := false
	err := cs.Pool.QueryRow(ctx, sql, id.V()).Scan(&found)
	if err != nil {
		return found, fmt.Errorf("exists by id: %s: %w", id.V(), err)
	}
	return found, nil
}

func (cs PgCurrencyStore) ExistByCode(ctx context.Context, code core.CurrencyCode) (bool, error) {
	sql := "SELECT EXISTS (SELECT 1 FROM currency WHERE code = $1)"
	found := false
	err := cs.Pool.QueryRow(ctx, sql, code.V()).Scan(&found)
	if err != nil {
		return found, fmt.Errorf("exists by code: %s: %w", code.V(), err)
	}
	return found, nil
}

func (cs PgCurrencyStore) mapCurrencies(flat []*currencyFlat) map[uuid.UUID]*core.Currency {
	currencies := map[uuid.UUID]*core.Currency{}
	for _, c := range flat {
		c2, ok := currencies[c.CID]
		if !ok {
			q := c.currency()
			currencies[c.CID] = q
			c2 = q
		}

		// For non-leafs, we have to maintain an identity map to avoid
		// duplicates. But Exchange rate ID being a leave makes it unique.
		e2 := c.exchangeRate()
		if e2 != nil {
			c2.ExchangeRates = append(c2.ExchangeRates, e2)
		}
	}
	return currencies
}

func (cs PgCurrencyStore) GetByCode(ctx context.Context, code core.CurrencyCode) (*core.Currency, error) {
	var sql = `
		SELECT c.id, c.code, c.version, c.created_at, c.updated_at,
  			   e.id, e.rate, e.from, e.created_at, e.updated_at
		FROM currency c
		LEFT JOIN exchange_rate e ON c.id = e.currency_id
		WHERE c.code = $1`
	rows, _ := cs.Pool.Query(ctx, sql, code.V())
	currencies, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[currencyFlat])
	if err != nil {
		return nil, fmt.Errorf("get by code: %s: %w", code.V(), err)
	}
	if len(currencies) == 0 {
		return nil, nil
	}
	c := cs.mapCurrencies(currencies)
	core.Assert(len(c) == 1, "data inconsistency")
	for _, v := range c {
		return v, nil
	}
	panic("unreachable")
}

// TierDiscount

type tierDiscountFlat struct {
	ID         uuid.UUID
	Authorized float64
	Advanced   float64
	Premier    float64
	From       core.Date
	Version    int32
	CreatedAt  time.Time
	UpdatedAt  *time.Time
}

func (td tierDiscountFlat) tierDiscount() *core.TierDiscount {
	return &core.TierDiscount{
		AggregateRoot: core.AggregateRoot{
			Version: td.Version,
			Entity: core.Entity{
				ID:        td.ID,
				CreatedAt: td.CreatedAt,
				UpdatedAt: td.UpdatedAt,
			},
		},
		Percentages: core.MustParseDiscountPercentages(
			td.Authorized,
			td.Advanced,
			td.Premier),
		From: core.MustParseTierDiscountFrom(td.From),
	}
}

type PgTierDiscountStore struct {
	Pool *pgxpool.Pool
}

func (r PgTierDiscountStore) ExistByID(ctx context.Context, id core.TierDiscountID) (bool, error) {
	sql := "SELECT EXISTS (SELECT 1 FROM tier_discount WHERE id = $1)"
	found := false
	err := r.Pool.QueryRow(ctx, sql, id.V()).Scan(&found)
	if err != nil {
		return found, fmt.Errorf("exists by id: %s: %w", id.V(), err)
	}
	return found, nil
}

func (r PgTierDiscountStore) mapTierDiscount(flat []*tierDiscountFlat) map[uuid.UUID]*core.TierDiscount {
	tierDiscounts := map[uuid.UUID]*core.TierDiscount{}
	for _, td := range flat {
		tierDiscounts[td.ID] = td.tierDiscount()
	}
	return tierDiscounts
}

func (r PgTierDiscountStore) GetByID(ctx context.Context, id core.TierDiscountID) (*core.TierDiscount, error) {
	var sql = `
		SELECT td.id, td.authorized, td.advanced, td.premier, td.from, td.version, td.created_at, td.updated_at
		FROM tier_discount td
		WHERE td.id = $1`
	rows, _ := r.Pool.Query(ctx, sql, id.V())
	tierDiscounts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[tierDiscountFlat])
	if err != nil {
		return nil, fmt.Errorf("get by id: %s: %w", id.V(), err)
	}
	if len(tierDiscounts) == 0 {
		return nil, nil
	}
	td := r.mapTierDiscount(tierDiscounts)
	core.Assert(len(td) == 1, "data inconsistency")
	for _, v := range td {
		return v, nil
	}
	panic("unreachable")
}

type PgStoreProjector struct {
	Pool *pgxpool.Pool
}

func (sp PgStoreProjector) withTx(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	// While pgx batching may be more efficient, don't use it as it makes
	// troubleshooting which query failed more difficult and also comes with
	//
	// If transactional cross-repository Save is ever needed, instead of each
	// repository's Save method creating a transaction, pass in a transaction.
	// Alternatively, create a single Save method spanning all repositories.
	tx, err := sp.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("unable to begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			panic(p)
		}
		// If err is non-nil from fn(tx) or from tx.Commit, ensure rollback to
		// clean up connection.
		if err != nil {
			rbErr := tx.Rollback(context.Background())
			if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("tx rollback failed: %w", rbErr))
			}
			return
		}
	}()

	err = fn(tx)
	if err != nil {
		// Defer will catch and rollback transaction.
		return err
	}

	// On failure, err is set, and defer will rollback tranaction to ensure the
	// connection is reset.
	if cErr := tx.Commit(ctx); cErr != nil {
		err = fmt.Errorf("tx commit failed: %w", cErr)
	}

	return err
}

var entityTableMap = map[reflect.Type]string{
	reflect.TypeFor[*core.Currency]():     "currency",
	reflect.TypeFor[*core.TierDiscount](): "tier_discount",
}

func (sp PgStoreProjector) enforceOptimisticLock(ctx context.Context, tx pgx.Tx, aggregate core.Aggregate) error {
	root := aggregate.GetAggregateRoot()

	t := reflect.TypeOf(aggregate)
	table, ok := entityTableMap[t]
	if !ok {
		panic(fmt.Sprintf("unhandled type: %T", aggregate))
	}

	q := fmt.Sprintf(`
		UPDATE %s
		SET version = version + 1
		WHERE id = $1 AND version = $2`, table)
	tag, err := tx.Exec(ctx, q, root.ID, root.Version)
	if err != nil {
		return fmt.Errorf("%s optimistic lock (id=%s) execution failed: %w", table, root.ID, err)
	}

	count := tag.RowsAffected()
	if count == 0 {
		return core.NewDataStaleError(sp.typeName(aggregate), root.ID)
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s optimistic lock (id=%s) unexpected row count: %d", table, root.ID, count)
	}
	return nil
}

// Apply enables projecting changes to one or more aggregates of same or
// different types to the database. In case of event publishing within a
// handler, different types of aggregates may be at play. Except for event
// publishing across aggregates, one shouldn't update one aggregate from
// another.
//
// Passing different types of aggregates, care must be taken to avoid a deadlock
// in the database during optimistic lock acquisition across roots. The easiest
// way to avoid such deadlock is to always pass the types in the same order
// across calls to Apply.
func (sp PgStoreProjector) Apply(ctx context.Context, aggregates ...core.Aggregate) error {
	// TODO(rh): In the future have apply stort types in reverse dependency order.
	return sp.withTx(ctx, func(tx pgx.Tx) error {
		for _, aggregate := range aggregates {
			root := aggregate.GetAggregateRoot()
			if len(root.DomainEvents) == 0 {
				continue
			}

			if root.Version > 0 {
				err := sp.enforceOptimisticLock(ctx, tx, aggregate)
				if err != nil {
					return err
				}
			}

			for _, event := range root.DomainEvents {
				if err := sp.persist(ctx, tx, root.ID, root.Version, event); err != nil {
					return err
				}
				if err := sp.project(ctx, tx, event); err != nil {
					return err
				}
			}

			// Beware that if the transaction fails, the domain events are gone.
			// The store isn't designed around retrying a failed transaction.
			// Instead changes to aggregates should be re-applied by re-running
			// the commands/queries, starting from up-to-date aggregate state.
			root.ClearDomainEvents()
		}
		return nil
	})
}

func (sp PgStoreProjector) typeName(ty any) string {
	// Remove "core." prefix from type name.
	t := reflect.TypeOf(ty)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func (sp PgStoreProjector) persist(ctx context.Context, tx pgx.Tx, aggregateID uuid.UUID, version int32, event core.DomainEvent) error {
	eventType := sp.typeName(event)
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event %s): %w", eventType, err)
	}

	q := `INSERT INTO domain_event (aggregate_id, type, payload, version, occurred_at) values ($1, $2, $3, $4, $5)`
	tag, err := tx.Exec(ctx, q, aggregateID, eventType, b, version+1, event.At())
	if err != nil {
		return fmt.Errorf("persist %s execution failed: %w", eventType, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("persist %s unexpected row count: %d", eventType, tag.RowsAffected())
	}
	return nil
}

// TODO(rh): If you decide that deleting a row that is already gone shouldn't be
// an error (idempotency), you can create a second helper checkIgnoreMissing
// that doesn't care if RowsAffected is 0, or add a boolean flag to the existing
// helper. Otherwise, the version above is the cleanest way to handle strictly
// enforced 1-row changes.
func (sp PgStoreProjector) checkExec(err error, tag pgconn.CommandTag, event core.DomainEvent, id uuid.UUID) error {
	eventType := sp.typeName(event)
	if err != nil {
		return fmt.Errorf("project %s (id=%s) execution failed: %w", eventType, id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("project %s (id=%s) unexpected row count: %d", eventType, id, tag.RowsAffected())
	}
	return nil
}

func (sp PgStoreProjector) project(ctx context.Context, tx pgx.Tx, event core.DomainEvent) error {
	switch e := event.(type) {
	// Currency
	case core.CurrencyCreatedEvent:
		q := `INSERT INTO currency (id, code, version, created_at) VALUES ($1, $2, $3, $4)`
		tag, err := tx.Exec(ctx, q, e.ID, e.Code, 1, e.OccurredAt)
		return sp.checkExec(err, tag, e, e.ID)
	case core.CurrencyRemovedEvent:
		tag, err := tx.Exec(ctx, "DELETE FROM currency WHERE id = $1", e.ID)
		return sp.checkExec(err, tag, e, e.ID)
	case core.ExchangeRateAddedEvent:
		q := `INSERT INTO exchange_rate (id, currency_id, rate, "from", created_at) values ($1, $2, $3, $4, $5)`
		tag, err := tx.Exec(ctx, q, e.ExchangeRateID, e.CurrencyID, e.Rate, e.From, e.OccurredAt)
		return sp.checkExec(err, tag, e, e.ExchangeRateID)
	case core.ExchangeRateUpdatedEvent:
		q := `
            UPDATE exchange_rate 
            SET rate = $1, "from" = $2, updated_at = $3 
            WHERE id = $4 AND currency_id = $5`
		tag, err := tx.Exec(ctx, q, e.Rate, e.From, e.OccurredAt, e.ExchangeRateID, e.CurrencyID)
		return sp.checkExec(err, tag, e, e.ExchangeRateID)
	case core.ExchangeRateRemovedEvent:
		q := `DELETE FROM exchange_rate WHERE id = $1 AND currency_id = $2`
		tag, err := tx.Exec(ctx, q, e.ExchangeRateID, e.CurrencyID)
		return sp.checkExec(err, tag, e, e.ExchangeRateID)

	// TierDiscount
	case core.TierDiscountCreatedEvent:
		q := `INSERT INTO tier_discount (id, authorized, advanced, premier, "from", version, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		tag, err := tx.Exec(ctx, q, e.ID, e.Authorized, e.Advanced, e.Premier, e.From, 1, e.OccurredAt)
		return sp.checkExec(err, tag, e, e.ID)
	case core.TierDiscountUpdatedEvent:
		q := `
            UPDATE tier_discount 
            SET authorized = $1, advanced = $2, premier = $3, "from" = $4, updated_at = $5 
            WHERE id = $6`
		tag, err := tx.Exec(ctx, q, e.Authorized, e.Advanced, e.Premier, e.From, e.ID, e.OccurredAt)
		return sp.checkExec(err, tag, e, e.ID)
	case core.TierDiscountRemovedEvent:
		tag, err := tx.Exec(ctx, "DELETE FROM tier_discount WHERE id = $1", e.ID)
		return sp.checkExec(err, tag, e, e.ID)
	default:
		panic(fmt.Sprintf("unhandled type: %T", e))
	}
}
