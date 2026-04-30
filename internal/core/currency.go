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
	Save(context.Context, *Currency) error
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
			DomainEvents: []DomainEvent{},
			Entity: Entity{
				ID:        id,
				CreatedAt: createdAt,
				UpdatedAt: nil,
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
		return e.From.Compare(today) <= 0
	})
	if !canRemove {
		return NewDomainError(
			CurrencyRemoveRequiresFutureFrom,
			fmt.Sprintf("remove currency requires all exchange rates to have from after today %s", today.String()))
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
	Clock      Clock
}

func (h CreateCurrencyHandler) validate(req CreateCurrencyCommand, err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
	ValidateStringCurrencyCode("Code", req.Code, err)
}

func (h CreateCurrencyHandler) Handle(ctx context.Context, req CreateCurrencyCommand) error {
	if err := Validate(req, h.validate); err != nil {
		return err
	}

	found, err := h.Currencies.ExistByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("Currency", "ID", req.ID.String())
	}

	found, err = h.Currencies.ExistByCode(ctx, req.Code)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("Currency", "Code", req.Code)
	}

	currency := NewCurrency(req.ID, req.Code, h.Clock.NowUTC())
	return h.Currencies.Save(ctx, &currency)
}

type RemoveCurrencyCommand struct {
	Code string
}

type RemoveCurrencyHandler struct {
	Currencies CurrencyStore
	Clock      Clock
}

func (h RemoveCurrencyHandler) validate(req RemoveCurrencyCommand, err *ValidationError) {
	ValidateStringCurrencyCode("Code", req.Code, err)
}

func (h RemoveCurrencyHandler) Handle(ctx context.Context, req RemoveCurrencyCommand) error {
	if err := Validate(req, h.validate); err != nil {
		return err
	}

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
	return h.Currencies.Save(ctx, currency)
}

type AddExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
	Rate float64
	From Date
}

type AddExchangeRateHandler struct {
	Currencies CurrencyStore
	Clock      Clock
}

func (h AddExchangeRateHandler) validate(c AddExchangeRateCommand, err *ValidationError) {
	ValidateUUIDNotZero("ID", c.ID, err)
	ValidateStringCurrencyCode("Code", c.Code, err)
	ValidateFloat64InclusiveRange("Rate", c.Rate, MinExchangeRate, MaxExchangeRate, err)
	ValidateFloat64DecimalPlaces("Rate", c.Rate, MinExchangeRateDecimalPlaces, MaxExchangeRateDecimalPlaces, err)
	ValidateDateInclusiveRange("From", c.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

func (h AddExchangeRateHandler) Handle(ctx context.Context, req AddExchangeRateCommand) error {
	if err := Validate(req, h.validate); err != nil {
		return err
	}

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
	return h.Currencies.Save(ctx, currency)
}

type UpdateExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
	Rate float64
	From Date
}

type UpdateExchangeRateHandler struct {
	Currencies CurrencyStore
	Clock      Clock
}

func (h UpdateExchangeRateHandler) validate(req UpdateExchangeRateCommand, err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
	ValidateStringCurrencyCode("Code", req.Code, err)
	ValidateFloat64InclusiveRange("Rate", req.Rate, MinExchangeRate, MaxExchangeRate, err)
	ValidateFloat64DecimalPlaces("Rate", req.Rate, MinExchangeRateDecimalPlaces, MaxExchangeRateDecimalPlaces, err)
	ValidateDateInclusiveRange("From", req.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

func (h UpdateExchangeRateHandler) Handle(ctx context.Context, req UpdateExchangeRateCommand) error {
	if err := Validate(req, h.validate); err != nil {
		return err
	}

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

	if err := currency.UpdateExchangeRate(*exchangeRateByID, req.Rate, req.From, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Currencies.Save(ctx, currency)
}

type RemoveExchangeRateCommand struct {
	ID   uuid.UUID
	Code string
}

type RemoveExchangeRateHandler struct {
	Currencies CurrencyStore
	Clock      Clock
}

func (h RemoveExchangeRateHandler) validate(req RemoveExchangeRateCommand, err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
	ValidateStringCurrencyCode("Code", req.Code, err)
}

func (h RemoveExchangeRateHandler) Handle(ctx context.Context, req RemoveExchangeRateCommand) error {
	if err := Validate(req, h.validate); err != nil {
		return err
	}

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
	return h.Currencies.Save(ctx, currency)
}

type GetCurrencyByCodeQuery struct {
	Code string
}

type GetCurrencyByCodeHandler struct {
	Currencies CurrencyStore
}

func (h GetCurrencyByCodeHandler) validate(qry GetCurrencyByCodeQuery, err *ValidationError) {
	ValidateStringCurrencyCode("Code", qry.Code, err)
}

func (h GetCurrencyByCodeHandler) Handle(ctx context.Context, qry GetCurrencyByCodeQuery) (*Currency, error) {
	if err := Validate(qry, h.validate); err != nil {
		return nil, err
	}

	currency, err := h.Currencies.GetByCode(ctx, qry.Code)
	if err != nil {
		return nil, err
	}
	if currency == nil {
		return nil, NewNotFoundError("Currency", "Code", qry.Code)
	}
	return currency, nil
}
