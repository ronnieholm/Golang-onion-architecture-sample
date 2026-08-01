package infrastructure

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ronnieholm/resellerloyalty/internal/core"
)

// Dispatcher is in infrastructure rather than core or it would need to be
// passed concrete interface implementation for core to remain technology
// agnostic. Static dependencies could be passed directly, but non-static ones
// would have to be passed as funcs for Dispatcher to instantiate those on
// demand. Also, pgxpool.Pool, and possibly others in a large app, isn't
// interface based. Putting it behind an interface and adding funcs is overkill.
//
// Key is to share Dispatcher across hosts such as service or terminal app by
// having their infrastructure reference shared infrastructure.

// Handler represents the generic signature for handlers which allows for
// decorator handlers. It's assumed that every handler returns a response and
// error. Handlers without an actual response, such as non-get handlers, return
// a special Empty type.
type Handler[Req any, Res any] func(context.Context, Req) (Res, error)

// Empty signals callers that the handler didn't return a useful value on the
// success path.
type Empty struct{}

func WithTiming[Req any, Res any](next Handler[Req, Res]) Handler[Req, Res] {
	return func(ctx context.Context, r Req) (Res, error) {
		start := time.Now()

		// Using defer ensures code runs even if next panics.
		defer func() {
			log.Printf("WithTiming: %T %v", r, time.Since(start)) // TODO(rh): core.GetCurrencyQuery -> GetCurrencyQuery
		}()

		return next(ctx, r)
	}
}

func WithLogging[Req any, Res any](next Handler[Req, Res]) Handler[Req, Res] {
	return func(ctx context.Context, r Req) (res Res, err error) {
		t := fmt.Sprintf("%T", r)
		defer func() {
			// For a PII sensitive service, rather than logging every field,
			// identify by name or type a request that requires cleaning. Then
			// clone the request and remove the sensitive parts before logging.

			if rec := recover(); rec != nil {
				slog.ErrorContext(ctx, "handler panicked",
					slog.String("handler", t),
					slog.Any("panic", rec),
					// Beware that adding struct tags such as json:"..." to a
					// request is deliberately ignored by slog. For performance
					// reasons slog doesn't use the encoding/json package.
					// Passing a struct to slog.Any causes the logger to reflect
					// on field names.
					slog.Any("request", r),
				)
				panic(rec)
			}

			attrs := []any{
				slog.String("handler", t),
				slog.Any("request", r),
			}

			// TODO(rh): print type of error struct? Deep print as pgx struct contains useful actual error details outside String().
			if err != nil {
				slog.ErrorContext(ctx, "handler failed", append(attrs, slog.Any("error", err))...)
			} else {
				slog.InfoContext(ctx, "handler success", attrs...)
			}
		}()

		return next(ctx, r)
	}
}

// Decorate avoids repeating the common chain of decorators for every handler.
func Decorate[Req any, Res any](h func(context.Context, Req) (Res, error)) Handler[Req, Res] {
	handler := Handler[Req, Res](h)

	// Apply decorators in order, innermost to outermost.
	handler = WithLogging(handler)
	handler = WithTiming(handler)
	return handler
}

type DispatcherOption func(*dispatcherOptions)

// DispatcherOptions is the dependencies for which substitution is supported in
// tests. Only dependencies with an actual need are included.
type dispatcherOptions struct {
	clock core.Clock
}

func WithClock(clock core.Clock) DispatcherOption {
	return func(d *dispatcherOptions) {
		d.clock = clock
	}
}

type Dispatcher struct {
	PgxPool *pgxpool.Pool

	// Because core.Clock is an interface, it default is nil without it being a
	// pointer, i.e, no need to declare as *core.Clock.
	clock core.Clock

	// Because these handlers don't have request scoped dependencies, handlers
	// can be initialized once and reused. Any request scoped dependency, such
	// as context.Context, or transient dependency must be passed through the
	// Handle method.

	// Currency
	CreateCurrency     Handler[core.CreateCurrencyCommand, Empty]
	RemoveCurrency     Handler[core.RemoveCurrencyCommand, Empty]
	AddExchangeRate    Handler[core.AddExchangeRateCommand, Empty]
	UpdateExchangeRate Handler[core.UpdateExchangeRateCommand, Empty]
	RemoveExchangeRate Handler[core.RemoveExchangeRateCommand, Empty]
	GetCurrency        Handler[core.GetCurrencyQuery, *core.Currency]

	// TierDiscount
	CreateTierDiscount Handler[core.CreateTierDiscountCommand, Empty]
	UpdateTierDiscount Handler[core.UpdateTierDiscountCommand, Empty]
	RemoveTierDiscount Handler[core.RemoveTierDiscountCommand, Empty]
	GetTierDiscount    Handler[core.GetTierDiscountQuery, *core.TierDiscount]
}

func NewDispatcher(ctx context.Context, config Config, opts ...DispatcherOption) Dispatcher {
	// Beware that pgxpool's maxConns is the greater of 4 and runtime.NumCPU().
	// On a powerful machine, maxConns of 32 may cause the pool's internal limit
	// be larger than the database server's limit, i.e., pgxpool MaxConns may be
	// set higher than the PostgreSQL server's max_connections.
	//
	// In such case, rather than pgxpool queueing request, an error is
	// attempting to run a query:
	//
	// server error: FATAL: sorry, too many clients already (SQLSTATE 53300)

	c, _ := pgxpool.ParseConfig(config.DBUrl)
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Application-wide connection string options should generally be part
		// of the DB_URL setting. But they can be set through
		//
		//  _, err := conn.Exec(ctx, "SET synchronous_commit TO OFF")
		//  return err
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}

	o := &dispatcherOptions{
		clock: &RealTimeClock{},
	}
	for _, opt := range opts {
		opt(o)
	}

	currencyStore := &PgCurrencyStore{
		Pool: pool,
	}
	tierDiscountStore := &PgTierDiscountStore{
		Pool: pool,
	}
	projector := &PgStoreProjector{
		Pool: pool,
	}

	// The benefit of setting up dependencies before any calls are dispatched is
	// that allocations are kept to a minimum across the lifetime of the
	// application.
	//
	// It can make substituting dependencies in tests more challenging. With a
	// dependency injection container, tests would surgically update the
	// container before a dependency is requested. Instead here a test would
	// create a separate instance of the dispatcher with fakes passed during
	// construction.
	//
	// It follows that any static dependency used to construct a handler must be
	// thread-safe. Non-static dependencies should be passed through the Handle
	// method, and depending on their nature may also have to be thread-safe.

	// Currency
	createCurrency := core.CreateCurrencyHandler{
		Currencies: currencyStore,
		Projector:  projector,
		Clock:      o.clock,
	}
	removeCurrency := core.RemoveCurrencyHandler{
		Currencies: currencyStore,
		Projector:  projector,
		Clock:      o.clock,
	}
	addExchangeRate := core.AddExchangeRateHandler{
		Currencies: currencyStore,
		Projector:  projector,
		Clock:      o.clock,
	}
	updateExchangeRate := core.UpdateExchangeRateHandler{
		Currencies: currencyStore,
		Projector:  projector,
		Clock:      o.clock,
	}
	removeExchangeRate := core.RemoveExchangeRateHandler{
		Currencies: currencyStore,
		Projector:  projector,
		Clock:      o.clock,
	}
	getCurrency := core.GetCurrencyHandler{
		Currencies: currencyStore,
	}

	// TierDiscount
	createTierDiscount := core.CreateTierDiscountHandler{
		TierDiscounts: tierDiscountStore,
		Projector:     projector,
		Clock:         o.clock,
	}
	updateTierDiscount := core.UpdateTierDiscountHandler{
		TierDiscounts: tierDiscountStore,
		Projector:     projector,
		Clock:         o.clock,
	}
	removeTierDiscount := core.RemoveTierDiscountHandler{
		TierDiscounts: tierDiscountStore,
		Projector:     projector,
		Clock:         o.clock,
	}
	getTierDiscount := core.GetTierDiscountHandler{
		TierDiscounts: tierDiscountStore,
	}

	return Dispatcher{
		PgxPool: pool,
		clock:   o.clock,
		// For handlers that don't have a success return value, the choice is
		// between (1) changing the handler in core to return (Empty, error) and
		// changing every return path inside the handler to Empty, err or (2)
		// patch the signature as below. To avoid polluting core, (2) is chosen.

		// Currency
		CreateCurrency: Decorate(func(ctx context.Context, req core.CreateCurrencyCommand) (Empty, error) {
			return Empty{}, createCurrency.Handle(ctx, req)
		}),
		RemoveCurrency: Decorate(func(ctx context.Context, req core.RemoveCurrencyCommand) (Empty, error) {
			return Empty{}, removeCurrency.Handle(ctx, req)
		}),
		AddExchangeRate: Decorate(func(ctx context.Context, req core.AddExchangeRateCommand) (Empty, error) {
			return Empty{}, addExchangeRate.Handle(ctx, req)
		}),
		UpdateExchangeRate: Decorate(func(ctx context.Context, req core.UpdateExchangeRateCommand) (Empty, error) {
			return Empty{}, updateExchangeRate.Handle(ctx, req)
		}),
		RemoveExchangeRate: Decorate(func(ctx context.Context, req core.RemoveExchangeRateCommand) (Empty, error) {
			return Empty{}, removeExchangeRate.Handle(ctx, req)
		}),
		GetCurrency: Decorate(getCurrency.Handle),

		// TierDiscount
		CreateTierDiscount: Decorate(func(ctx context.Context, req core.CreateTierDiscountCommand) (Empty, error) {
			return Empty{}, createTierDiscount.Handle(ctx, req)
		}),
		RemoveTierDiscount: Decorate(func(ctx context.Context, req core.RemoveTierDiscountCommand) (Empty, error) {
			return Empty{}, removeTierDiscount.Handle(ctx, req)
		}),
		UpdateTierDiscount: Decorate(func(ctx context.Context, req core.UpdateTierDiscountCommand) (Empty, error) {
			return Empty{}, updateTierDiscount.Handle(ctx, req)
		}),
		GetTierDiscount: Decorate(getTierDiscount.Handle),
	}
}

func (d *Dispatcher) Close() {
	d.PgxPool.Close()
}
