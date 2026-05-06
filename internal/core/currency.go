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
	ExistByID(context.Context, uuid.UUID) (bool, error)
	ExistByCode(context.Context, string) (bool, error)
	GetByCode(context.Context, string) (*Currency, error)
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

type ExchangeRate struct {
	Entity
	Rate float64
	From Date
}

func NewExchangeRate(id uuid.UUID, rate float64, from Date, createdAt time.Time) ExchangeRate {
	return ExchangeRate{
		Entity: Entity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: nil,
		},
		Rate: rate,
		From: from,
	}
}

func (e *ExchangeRate) Update(rate float64, from Date, updatedAt time.Time) error {
	if e.Rate == rate && e.From.Equal(from) {
		return NewDomainError(
			CurrencyUpdateRequiresChange,
			fmt.Sprintf("update exchange rate requires a rate different from %g and/or a from different from %s", rate, from.String()))
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
	Code          string
	ExchangeRates []*ExchangeRate
}

func NewCurrency(id uuid.UUID, code string, createdAt time.Time) Currency {
	c := Currency{
		AggregateRoot: AggregateRoot{
			Entity: Entity{
				ID:        id,
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
		ID:   id,
		Code: code,
	})
	return c
}

func (e *Currency) Equal(other *Currency) bool {
	return EntityEqual(e, other)
}

func (c *Currency) AddExchangeRate(exchangeRate ExchangeRate, createdAt time.Time) error {
	today := DateFromTime(createdAt)
	if !exchangeRate.From.After(today) {
		return NewDomainError(
			CurrencyAddRequiresFutureFrom,
			fmt.Sprintf("add exchange rate requires from %s be after today %s", exchangeRate.From.String(), today.String()))
	}

	c.ExchangeRates = append(c.ExchangeRates, &exchangeRate)
	c.AddDomainEvent(ExchangeRateAddedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: createdAt,
		},
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
		Rate:           exchangeRate.Rate,
		From:           exchangeRate.From,
	})
	return nil
}

func (c *Currency) UpdateExchangeRate(exchangeRate ExchangeRate, rate float64, from Date, updatedAt time.Time) error {
	// TODO(rh): move check to ExchangeRate entity? Similar for other methods.
	today := DateFromTime(updatedAt)
	if !from.After(today) {
		return NewDomainError(
			CurrencyUpdateRequiresFutureFrom,
			fmt.Sprintf("update exchange rate requires from %s be after today %s", from.String(), today.String()))
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
		Rate:           exchangeRate.Rate,
		From:           exchangeRate.From,
	})
	return nil
}

func (c *Currency) RemoveExchangeRate(exchangeRate *ExchangeRate, updatedAt time.Time) error {
	today := DateFromTime(updatedAt)
	if !exchangeRate.From.After(today) {
		return NewDomainError(
			CurrencyRemoveRequiresFutureFrom,
			fmt.Sprintf("remove exchange rate requires from %s be after today %s", exchangeRate.From.String(), today.String()))
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
		return e.From.Compare(today) <= 0 // TODO(rh): use !e.From.Ater(today) which is simpler to read.
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

func (r CreateCurrencyCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", r.ID, err)
	ValidateStringCurrencyCode("Code", r.Code, err)
}

type CreateCurrencyHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h CreateCurrencyHandler) Handle(ctx context.Context, req CreateCurrencyCommand) error {
	found, err := h.Currencies.ExistByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("Currency", "ID", req.ID.String())
	}

	found, err = h.Currencies.ExistByCode(ctx, req.Code)
	if err != nil {
		return err // TODO(rh): add context (and other places)
	}
	if found {
		return NewConflictError("Currency", "Code", req.Code)
	}

	currency := NewCurrency(req.ID, req.Code, h.Clock.NowUTC())
	return h.Projector.Apply(ctx, &currency)
}

type RemoveCurrencyCommand struct {
	Code string
}

func (r RemoveCurrencyCommand) Validate(err *ValidationError) {
	ValidateStringCurrencyCode("Code", r.Code, err)
}

type RemoveCurrencyHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h RemoveCurrencyHandler) Handle(ctx context.Context, req RemoveCurrencyCommand) error {
	currency, err := h.Currencies.GetByCode(ctx, req.Code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", req.Code)
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

func (r AddExchangeRateCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", r.ID, err)
	ValidateStringCurrencyCode("Code", r.Code, err)
	ValidateFloat64InclusiveRange("Rate", r.Rate, MinExchangeRate, MaxExchangeRate, err)
	ValidateFloat64DecimalPlaces("Rate", r.Rate, MinExchangeRateDecimalPlaces, MaxExchangeRateDecimalPlaces, err)
	ValidateDateInclusiveRange("From", r.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

type AddExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h AddExchangeRateHandler) Handle(ctx context.Context, req AddExchangeRateCommand) error {
	currency, err := h.Currencies.GetByCode(ctx, req.Code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", req.Code)
	}

	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			return NewConflictError("ExchangeRate", "ID", req.ID.String())
		}
		if e.From.Equal(req.From) {
			return NewConflictError("ExchangeRate", "From", req.From.String())
		}
	}

	now := time.Now()
	exchangeRate := NewExchangeRate(req.ID, req.Rate, req.From, now)
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

func (r UpdateExchangeRateCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", r.ID, err)
	ValidateStringCurrencyCode("Code", r.Code, err)
	ValidateFloat64InclusiveRange("Rate", r.Rate, MinExchangeRate, MaxExchangeRate, err)
	ValidateFloat64DecimalPlaces("Rate", r.Rate, MinExchangeRateDecimalPlaces, MaxExchangeRateDecimalPlaces, err)
	ValidateDateInclusiveRange("From", r.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

type UpdateExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h UpdateExchangeRateHandler) Handle(ctx context.Context, req UpdateExchangeRateCommand) error {
	currency, err := h.Currencies.GetByCode(ctx, req.Code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", req.Code)
	}

	var (
		exchangeRateByID   *ExchangeRate
		exchangeRateByFrom *ExchangeRate
	)
	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			exchangeRateByID = e
		}
		if e.From.Equal(req.From) {
			exchangeRateByFrom = e
		}
	}
	if exchangeRateByID == nil {
		return NewNotFoundError("ExchangeRate", "ID", req.ID.String())
	}
	if exchangeRateByFrom != nil && exchangeRateByFrom.ID != req.ID {
		return NewConflictError("ExchangeRate", "From", req.From.String())
	}

	// TODO(rh): why *exchangeRateByID and not exchangeRateByID to minimize copying? Is exchangeRate large enough to avoid copying? Over 64 bytes.
	if err := currency.UpdateExchangeRate(*exchangeRateByID, req.Rate, req.From, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type RemoveExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
}

func (r RemoveExchangeRateCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", r.ID, err)
	ValidateStringCurrencyCode("Code", r.Code, err)
}

type RemoveExchangeRateHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h RemoveExchangeRateHandler) Handle(ctx context.Context, req RemoveExchangeRateCommand) error {
	currency, err := h.Currencies.GetByCode(ctx, req.Code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", req.Code)
	}

	var exchangeRate *ExchangeRate
	for _, e := range currency.ExchangeRates {
		if e.ID == req.ID {
			exchangeRate = e
		}
	}
	if exchangeRate == nil {
		return NewNotFoundError("ExchangeRate", "ID", req.ID.String())
	}

	if err := currency.RemoveExchangeRate(exchangeRate, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type GetCurrencyQuery struct {
	Code string
}

func (r GetCurrencyQuery) Validate(err *ValidationError) {
	ValidateStringCurrencyCode("Code", r.Code, err)
}

type GetCurrencyHandler struct {
	Currencies CurrencyStore
}

func (h GetCurrencyHandler) Handle(ctx context.Context, req GetCurrencyQuery) (*Currency, error) {
	currency, err := h.Currencies.GetByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if currency == nil {
		return nil, NewNotFoundError("Currency", "Code", req.Code)
	}
	return currency, nil
}
