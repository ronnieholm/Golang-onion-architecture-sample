package infrastructure

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

type currencyFlat struct {
	CID        uuid.UUID
	CCode      string
	CVersion   int
	CCreatedAt time.Time
	CUpdatedAt *time.Time
	EID        *uuid.UUID
	ERate      *float64
	EFrom      *time.Time // TODO(rh): can we use Date directly given its Scanner implementation?
	ECreatedAt *time.Time
	EUpdatedAt *time.Time
}

func (c currencyFlat) currency() *core.Currency {
	return &core.Currency{
		AggregateRoot: core.AggregateRoot{
			Version:      c.CVersion,
			DomainEvents: nil,
			Entity: core.Entity{
				ID:        c.CID,
				CreatedAt: c.CCreatedAt,
				UpdatedAt: c.CUpdatedAt,
			},
		},
		Code: c.CCode,
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
			Rate: *c.ERate,
			From: core.DateFromTime(*c.EFrom),
		}
	} else {
		return nil
	}
}

type PgCurrencyStore struct {
	Pool *pgxpool.Pool
}

func (r PgCurrencyStore) ExistByID(ctx context.Context, id uuid.UUID) (bool, error) {
	sql := "SELECT EXISTS (SELECT 1 FROM currency WHERE id = $1)"
	found := false
	err := r.Pool.QueryRow(ctx, sql, id).Scan(&found)
	if err != nil {
		return found, fmt.Errorf("exists by id failed for id=%s: %w", id, err)
	}
	return found, nil
}

func (r PgCurrencyStore) ExistByCode(ctx context.Context, code string) (bool, error) {
	sql := "SELECT EXISTS (SELECT 1 FROM currency WHERE code = $1)"
	found := false
	err := r.Pool.QueryRow(ctx, sql, code).Scan(&found)
	if err != nil {
		return found, fmt.Errorf("exists by code failed for code=%s: %w", code, err)
	}
	return found, nil
}

func (r PgCurrencyStore) mapCurrencies(flat []*currencyFlat) map[uuid.UUID]*core.Currency {
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

func (r PgCurrencyStore) GetByCode(ctx context.Context, code string) (*core.Currency, error) {
	var sql = `
		SELECT c.id, c.code, c.version, c.created_at, c.updated_at,
  			   e.id, e.rate, e.from, e.created_at, e.updated_at
		FROM currency c
		LEFT JOIN exchange_rate e ON c.id = e.currency_id
		WHERE c.code = $1`
	rows, _ := r.Pool.Query(ctx, sql, code)
	currencies, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[currencyFlat])
	if err != nil {
		return nil, fmt.Errorf("get by code failed for code=%s: %w", code, err)
	}
	if len(currencies) == 0 {
		return nil, nil
	}
	c := r.mapCurrencies(currencies)
	if len(c) != 1 {
		panic("unreachable")
	}
	for _, x := range c {
		return x, nil
	}
	panic("unreachable")
}

func (r PgCurrencyStore) withTx(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	// While pgx batching may be more efficient, don't use it as it makes
	// troubleshooting which query failed more difficult and also comes with
	//
	// If transactional cross-repository Save is ever needed, instead of each
	// repository's Save method creating a transaction, pass in a transaction.
	// Alternatively, create a single Save method spanning all repositories.
	tx, err := r.Pool.Begin(ctx)
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

// TODO(rh): generalize to any aggregate.
func (r PgCurrencyStore) enforceOptimisticLock(ctx context.Context, tx pgx.Tx, entity *core.Currency) error {
	q := `
		UPDATE currency
		SET version = version + 1
		WHERE id = $1 AND version = $2`
	tag, err := tx.Exec(ctx, q, entity.ID, entity.Version)
	if err != nil {
		return fmt.Errorf("currency optimistic lock (id=%s) execution failed: %w", entity.ID, err)
	}

	count := tag.RowsAffected()
	if count == 0 {
		return core.NewDataStaleError("currency", entity.ID)
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("currency optimistic lock (id=%s) unexpected row count: %d", entity.ID, count)
	}
	return nil
}

// TODO(rh): splitting into writer and reader, save should be variadic on aggregates. Then a handler you update multiple (in the case of an event publishing) and pass aggregate in dependency order and changes would be saved in a single tx.
func (r PgCurrencyStore) Save(ctx context.Context, entity *core.Currency) error {
	if len(entity.DomainEvents) == 0 {
		return nil
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		if entity.Version > 0 {
			err := r.enforceOptimisticLock(ctx, tx, entity)
			if err != nil {
				return err
			}
		}

		// TODO(rh): copies event. Use pointer slice inside aggregate or index for loop?
		for _, event := range entity.DomainEvents {
			if err := r.logDomainEvent(ctx, tx, entity.ID, entity.Version, event); err != nil {
				return err
			}
			if err := r.projectDomainEvent(ctx, tx, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r PgCurrencyStore) logDomainEvent(ctx context.Context, tx pgx.Tx, aggregateID uuid.UUID, version int, event core.DomainEvent) error {
	// Remove "core." prefix from event type name.
	t := reflect.TypeOf(event)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	eventType := t.Name()

	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event %s): %w", t, err)
	}

	q := `INSERT INTO domain_event (aggregate_id, type, payload, version, occurred_at) values ($1, $2, $3, $4, $5)`
	tag, err := tx.Exec(ctx, q, aggregateID, eventType, b, version+1, event.At())
	if err != nil {
		return fmt.Errorf("add domain event %s execution failed: %w", t, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("add domain event %s unexpected row count: %d", t, tag.RowsAffected())
	}
	return nil
}

// TODO(rh): If you decide that deleting a row that is already gone shouldn't be
// an error (idempotency), you can create a second helper checkIgnoreMissing
// that doesn't care if RowsAffected is 0, or add a boolean flag to the existing
// helper. Otherwise, the version above is the cleanest way to handle strictly
// enforced 1-row changes.
func (r PgCurrencyStore) checkExec(err error, tag pgconn.CommandTag, action string, id uuid.UUID) error {
	if err != nil {
		return fmt.Errorf("%s (id=%s) execution failed: %w", action, id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%s (id=%s) unexpected row count: %d", action, id, tag.RowsAffected())
	}
	return nil
}

func (r PgCurrencyStore) projectDomainEvent(ctx context.Context, tx pgx.Tx, event core.DomainEvent) error {
	switch e := event.(type) {
	case core.CurrencyCreatedEvent:
		q := "INSERT INTO currency (id, code, version, created_at) VALUES ($1, $2, $3, $4)"
		tag, err := tx.Exec(ctx, q, e.ID, e.Code, 1, e.OccurredAt)
		return r.checkExec(err, tag, "create currency", e.ID)
	case core.CurrencyRemovedEvent:
		tag, err := tx.Exec(ctx, "DELETE FROM currency WHERE id = $1", e.ID)
		return r.checkExec(err, tag, "remove currency", e.ID)
	case core.ExchangeRateAddedEvent:
		q := `INSERT INTO exchange_rate (id, currency_id, rate, "from", created_at) values ($1, $2, $3, $4, $5)`
		tag, err := tx.Exec(ctx, q, e.ExchangeRateID, e.CurrencyID, e.Rate, e.From, e.OccurredAt)
		return r.checkExec(err, tag, "insert exchange rate", e.ExchangeRateID)
	case core.ExchangeRateUpdatedEvent:
		q2 := `
            UPDATE exchange_rate 
            SET rate = $1, "from" = $2, updated_at = $3 
            WHERE id = $4 AND currency_id = $5`
		tag, err := tx.Exec(ctx, q2, e.Rate, e.From, e.OccurredAt, e.ExchangeRateID, e.CurrencyID)
		return r.checkExec(err, tag, "update exchange rate", e.ExchangeRateID)
	case core.ExchangeRateRemovedEvent:
		q := "DELETE FROM exchange_rate WHERE id = $1 AND currency_id = $2"
		tag, err := tx.Exec(ctx, q, e.ExchangeRateID, e.CurrencyID)
		return r.checkExec(err, tag, "remove exchange rate", e.ExchangeRateID)
	default:
		return fmt.Errorf("unhandled type: %T", e)
	}
}
