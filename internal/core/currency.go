package core

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Domain

type CurrencyStore interface {
	ExistByID(context.Context, CurrencyID) (bool, error)
	ExistByCode(context.Context, CurrencyCode) (bool, error)
	GetByCode(context.Context, CurrencyCode) (*Currency, error)
}

type CurrencyCreatedEvent struct {
	domainEventCommon
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
}

type ExchangeRateAddedEvent struct {
	domainEventCommon
	CurrencyID     uuid.UUID `json:"currency_id"`
	ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
	Rate           float64   `json:"rate"`
	From           Date      `json:"from"`
}

type ExchangeRateUpdatedEvent struct {
	domainEventCommon
	CurrencyID     uuid.UUID `json:"currency_id"`
	ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
	Rate           float64   `json:"rate"`
	From           Date      `json:"from"`
}

type ExchangeRateRemovedEvent struct {
	domainEventCommon
	CurrencyID     uuid.UUID `json:"currency_id"`
	ExchangeRateID uuid.UUID `json:"exchange_rate_id"`
}

type CurrencyRemovedEvent struct {
	domainEventCommon
	ID uuid.UUID `json:"id"`
}

const (
	CurrencyAddRequiresFutureFrom    = 1600
	CurrencyUpdateRequiresFutureFrom = 1601
	CurrencyRemoveRequiresFutureFrom = 1602
	CurrencyUpdateRequiresChange     = 1603
)

const (
	ExchangeRateMin              float64 = 1.
	ExchangeRateMax              float64 = 100.
	ExchangeRateDecimalPlacesMin         = 0
	ExchangeRateDecimalPlacesMax         = 6
)

// CurrencyID

type CurrencyID struct {
	v uuid.UUID
}

func (c CurrencyID) V() uuid.UUID   { return c.v }
func (c CurrencyID) String() string { return c.v.String() }

func ParseCurrencyId(v uuid.UUID) (CurrencyID, error) {
	if err := ValidateUUIDNotZero(v); err != nil {
		return CurrencyID{}, err
	}
	return CurrencyID{v}, nil
}

func MustParseCurrencyId(v uuid.UUID) CurrencyID {
	v1, err := ParseCurrencyId(v)
	if err != nil {
		panic(err)
	}
	return v1
}

// ExchangeRateID

type ExchangeRateID struct {
	v uuid.UUID
}

func (e ExchangeRateID) V() uuid.UUID   { return e.v }
func (e ExchangeRateID) String() string { return e.v.String() }

func ParseExchangeRateId(v uuid.UUID) (ExchangeRateID, error) {
	if err := ValidateUUIDNotZero(v); err != nil {
		return ExchangeRateID{}, err
	}
	return ExchangeRateID{v}, nil
}

func MustParseExchangeRateId(v uuid.UUID) ExchangeRateID {
	v1, err := ParseExchangeRateId(v)
	if err != nil {
		panic(err)
	}
	return v1
}

type Rate struct { // TODO(rh): call ExchangeRateRate?
	v float64
}

func (r Rate) V() float64 { return r.v }

func ParseRate(v float64) (Rate, error) { // TODO(rh): Idiomatic to call it ParseRate?
	errs := &FieldParseError{}
	if err := ValidateFloat64InclusiveRange(v, ExchangeRateMin, ExchangeRateMax); err != nil {
		errs.Add(err.Error())
	}
	if err := ValidateFloat64DecimalPlaces(v, ExchangeRateDecimalPlacesMin, ExchangeRateDecimalPlacesMax); err != nil {
		errs.Add(err.Error())
	}
	if err := errs.NilOrError(); err != nil {
		return Rate{}, err
	}
	return Rate{v}, nil
}

func MustParseRate(v float64) Rate {
	v1, err := ParseRate(v)
	if err != nil {
		panic(err)
	}
	return v1
}

var (
	ExchangeRateFromMin = NewDate(2024, 1, 1)
	ExchangeRateFromMax = NewDate(2034, 12, 31)
)

type ExchangeRateFrom struct {
	v Date
}

func (f ExchangeRateFrom) V() Date        { return f.v }
func (f ExchangeRateFrom) String() string { return f.v.String() }

func ParseExchangeRateFrom(v Date) (ExchangeRateFrom, error) {
	if err := ValidateDateInclusiveRange(v, ExchangeRateFromMin, ExchangeRateFromMax); err != nil {
		return ExchangeRateFrom{}, err
	}
	return ExchangeRateFrom{v}, nil
}

func MustParseExchangeRateFrom(v Date) ExchangeRateFrom {
	v1, err := ParseExchangeRateFrom(v)
	if err != nil {
		panic(err)
	}
	return v1
}

type ExchangeRate struct {
	Entity
	Rate Rate
	From ExchangeRateFrom
}

func NewExchangeRate(id ExchangeRateID, rate Rate, from ExchangeRateFrom, createdAt time.Time) ExchangeRate {
	return ExchangeRate{
		Entity: Entity{
			ID:        id.V(), // TODO(rh): fix
			CreatedAt: createdAt,
			UpdatedAt: nil,
		},
		Rate: rate,
		From: from,
	}
}

func (e *ExchangeRate) Update(rate Rate, from ExchangeRateFrom, updatedAt time.Time) error {
	if e.Rate == rate && e.From == from {
		return NewDomainError(
			CurrencyUpdateRequiresChange,
			fmt.Sprintf("update exchange rate requires a rate different from %g and/or a from different from %s", rate, from))
	}

	e.Rate = rate
	e.From = from
	e.UpdatedAt = &updatedAt
	return nil
}

func (e *ExchangeRate) Equal(other *ExchangeRate) bool {
	return EntityEqual(e, other)
}

type Currency struct {
	AggregateRoot
	Code          CurrencyCode
	ExchangeRates []*ExchangeRate
}

func NewCurrency(id CurrencyID, code CurrencyCode, createdAt time.Time) Currency {
	c := Currency{
		AggregateRoot: AggregateRoot{
			Entity: Entity{
				ID:        id.V(), // TOOD(rh): Fix by making entity generic.
				CreatedAt: createdAt,
			},
		},
		Code:          code,
		ExchangeRates: []*ExchangeRate{},
	}

	c.AddDomainEvent(CurrencyCreatedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: createdAt,
		},
		ID:   id.V(),
		Code: code.V(),
	})
	return c
}

func (e *Currency) Equal(other *Currency) bool {
	return EntityEqual(e, other)
}

func (c *Currency) AddExchangeRate(exchangeRate ExchangeRate, createdAt time.Time) error {
	today := DateFromTime(createdAt)
	if !exchangeRate.From.V().After(today) {
		return NewDomainError(
			CurrencyAddRequiresFutureFrom,
			fmt.Sprintf("add exchange rate requires from %s be after today %s", exchangeRate.From, today.String()))
	}

	c.ExchangeRates = append(c.ExchangeRates, &exchangeRate)
	c.AddDomainEvent(ExchangeRateAddedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: createdAt,
		},
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
		Rate:           exchangeRate.Rate.V(),
		From:           exchangeRate.From.V(),
	})
	return nil
}

func (c *Currency) UpdateExchangeRate(exchangeRate ExchangeRate, rate Rate, from ExchangeRateFrom, updatedAt time.Time) error {
	// TODO(rh): move check to ExchangeRate entity? Similar for other methods.
	today := DateFromTime(updatedAt)
	if !from.V().After(today) {
		return NewDomainError(
			CurrencyUpdateRequiresFutureFrom,
			fmt.Sprintf("update exchange rate requires from %s be after today %s", from, today.String()))
	}

	if err := exchangeRate.Update(rate, from, updatedAt); err != nil {
		return fmt.Errorf("currency update exchange rate: %w", err)
	}

	c.AddDomainEvent(ExchangeRateUpdatedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: updatedAt,
		},
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
		Rate:           exchangeRate.Rate.V(),
		From:           exchangeRate.From.V(),
	})
	return nil
}

func (c *Currency) RemoveExchangeRate(exchangeRate *ExchangeRate, updatedAt time.Time) error {
	today := DateFromTime(updatedAt)
	if !exchangeRate.From.V().After(today) {
		return NewDomainError(
			CurrencyRemoveRequiresFutureFrom,
			fmt.Sprintf("remove exchange rate requires from %s be after today %s", exchangeRate.From, today.String()))
	}

	idx := slices.IndexFunc(c.ExchangeRates, exchangeRate.Equal)
	Assert(idx != -1, "missing exchange rate %s", exchangeRate.ID)

	c.ExchangeRates = slices.Delete(c.ExchangeRates, idx, idx+1)
	c.AddDomainEvent(ExchangeRateRemovedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: updatedAt,
		},
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
	})
	return nil
}

func (c *Currency) RemoveCurrency(removeAt time.Time) error {
	today := DateFromTime(removeAt)
	canRemove := !slices.ContainsFunc(c.ExchangeRates, func(e *ExchangeRate) bool {
		return e.From.V().Compare(today) <= 0 // TODO(rh): use !e.From.Ater(today) which is simpler to read.
	})
	if !canRemove {
		return NewDomainError(
			CurrencyRemoveRequiresFutureFrom,
			fmt.Sprintf("remove currency requires all exchange rates to have from after today %s", today))
	}

	for i := len(c.ExchangeRates) - 1; i >= 0; i-- {
		if err := c.RemoveExchangeRate(c.ExchangeRates[i], removeAt); err != nil {
			return fmt.Errorf("remove currency: %w", err)
		}
	}

	c.AddDomainEvent(CurrencyRemovedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: removeAt,
		},
		ID: c.ID,
	})
	return nil
}

// Application

type CreateCurrencyCommand struct {
	ID   uuid.UUID
	Code string
}

type CreateCurrencyHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h CreateCurrencyHandler) Handle(ctx context.Context, req CreateCurrencyCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseCurrencyId)
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	if errs.HasErrors() {
		return errs
	}

	found, err := h.Currencies.ExistByID(ctx, id)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("Currency", "ID", id.String())
	}

	found, err = h.Currencies.ExistByCode(ctx, code)
	if err != nil {
		return err // TODO(rh): add context (and other places)
	}
	if found {
		return NewConflictError("Currency", "Code", code.V())
	}

	currency := NewCurrency(id, code, h.Clock.NowUTC())
	return h.Projector.Apply(ctx, &currency)
}

type RemoveCurrencyCommand struct {
	Code string
}

func (r RemoveCurrencyCommand) Validate(err *RequestParseError) {
}

type RemoveCurrencyHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h RemoveCurrencyHandler) Handle(ctx context.Context, req RemoveCurrencyCommand) error {
	errs := &RequestParseError{}
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	if errs.HasErrors() {
		return errs
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	if err := currency.RemoveCurrency(h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type AddExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
	Rate float64
	From Date
}

type AddExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h AddExchangeRateHandler) Handle(ctx context.Context, req AddExchangeRateCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseExchangeRateId)
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	rate := Parse(errs, "Rate", req.Rate, ParseRate)
	from := Parse(errs, "From", req.From, ParseExchangeRateFrom)
	if errs.HasErrors() {
		return errs
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			return NewConflictError("ExchangeRate", "ID", id.String())
		}
		if e.From.V() == req.From {
			return NewConflictError("ExchangeRate", "From", from.String())
		}
	}

	now := time.Now()
	exchangeRate := NewExchangeRate(id, rate, from, now)
	err = currency.AddExchangeRate(exchangeRate, h.Clock.NowUTC())
	if err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type UpdateExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
	Rate float64
	From Date
}

type UpdateExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h UpdateExchangeRateHandler) Handle(ctx context.Context, req UpdateExchangeRateCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseExchangeRateId)
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	rate := Parse(errs, "Rate", req.Rate, ParseRate)
	from := Parse(errs, "From", req.From, ParseExchangeRateFrom)
	if errs.HasErrors() {
		return errs
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	var (
		exchangeRateByID   *ExchangeRate
		exchangeRateByFrom *ExchangeRate
	)
	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			exchangeRateByID = e
		}
		if e.From.V().Equal(req.From) {
			exchangeRateByFrom = e
		}
	}
	if exchangeRateByID == nil {
		return NewNotFoundError("ExchangeRate", "ID", id.String())
	}
	if exchangeRateByFrom != nil && exchangeRateByFrom.ID != req.ID {
		return NewConflictError("ExchangeRate", "From", from.String())
	}

	// TODO(rh): why *exchangeRateByID and not exchangeRateByID to minimize copying? Is exchangeRate large enough to avoid copying? Over 64 bytes.
	if err := currency.UpdateExchangeRate(*exchangeRateByID, rate, from, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type RemoveExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
}

type RemoveExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h RemoveExchangeRateHandler) Handle(ctx context.Context, req RemoveExchangeRateCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseExchangeRateId)
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	if errs.HasErrors() {
		return errs
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	var exchangeRate *ExchangeRate
	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			exchangeRate = e
		}
	}
	if exchangeRate == nil {
		return NewNotFoundError("ExchangeRate", "ID", id.String())
	}

	if err := currency.RemoveExchangeRate(exchangeRate, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type GetCurrencyQuery struct {
	Code string
}

type GetCurrencyHandler struct {
	Currencies CurrencyStore
}

func (h GetCurrencyHandler) Handle(ctx context.Context, req GetCurrencyQuery) (*Currency, error) {
	errs := &RequestParseError{}
	code := Parse(errs, "Code", req.Code, ParseCurrencyCode)
	if errs.HasErrors() {
		return nil, errs
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if currency == nil {
		return nil, NewNotFoundError("Currency", "Code", code.V())
	}
	return currency, nil
}
