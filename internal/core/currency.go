package core

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"
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

type Rate struct {
	v float64
}

func (r Rate) V() float64 { return r.v }

func ParseRate(v float64) (Rate, error) {
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
		ID:        id.V(),
		CreatedAt: createdAt,
		UpdatedAt: nil,
		Rate:      rate,
		From:      from,
	}
}

func (e *ExchangeRate) Update(rate Rate, from ExchangeRateFrom, updatedAt time.Time) error {
	today := DateFromTime(updatedAt)
	if !from.V().After(today) {
		return NewDomainError(
			CurrencyUpdateRequiresFutureFrom,
			fmt.Sprintf("update exchange rate requires from %s be after today %s", from, today.String()))
	}

	if e.Rate == rate && e.From == from {
		return NewDomainError(
			CurrencyUpdateRequiresChange,
			fmt.Sprintf("update exchange rate requires a rate different from %g and/or a from different from %s", rate.V(), from))
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
		ID:            id.V(),
		CreatedAt:     createdAt,
		Code:          code,
		ExchangeRates: []*ExchangeRate{},
	}

	c.AddDomainEvent(CurrencyCreatedEvent{
		OccurredAt: createdAt,
		ID:         id.V(),
		Code:       code.V(),
	})
	return c
}

func (e *Currency) Equal(other *Currency) bool {
	return EntityEqual(e, other)
}

func (c *Currency) AddExchangeRate(exchangeRate ExchangeRate, createdAt time.Time) error {
	for _, e := range c.ExchangeRates {
		if e.ID == exchangeRate.ID {
			return NewConflictError("ExchangeRate", "ID", exchangeRate.ID.String())
		}
		if e.From == exchangeRate.From {
			return NewConflictError("ExchangeRate", "From", exchangeRate.From.String())
		}
	}

	today := DateFromTime(createdAt)
	if !exchangeRate.From.V().After(today) {
		return NewDomainError(
			CurrencyAddRequiresFutureFrom,
			fmt.Sprintf("add exchange rate requires from %s be after today %s", exchangeRate.From, today.String()))
	}

	c.ExchangeRates = append(c.ExchangeRates, &exchangeRate)
	c.AddDomainEvent(ExchangeRateAddedEvent{
		OccurredAt:     createdAt,
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
		Rate:           exchangeRate.Rate.V(),
		From:           exchangeRate.From.V(),
	})
	return nil
}

func (c *Currency) UpdateExchangeRate(exchangeRateId ExchangeRateID, rate Rate, from ExchangeRateFrom, updatedAt time.Time) error {
	var (
		exchangeRateByID   *ExchangeRate
		exchangeRateByFrom *ExchangeRate
	)
	for _, e := range c.ExchangeRates {
		if e.ID == exchangeRateId.V() {
			exchangeRateByID = e
		}
		if e.From.V() == from.V() {
			exchangeRateByFrom = e
		}
	}
	if exchangeRateByID == nil {
		return NewNotFoundError("ExchangeRate", "ID", exchangeRateId.String())
	}
	if exchangeRateByFrom != nil && exchangeRateByFrom.ID != exchangeRateId.V() {
		return NewConflictError("ExchangeRate", "From", from.String())
	}

	if err := exchangeRateByID.Update(rate, from, updatedAt); err != nil {
		return fmt.Errorf("currency update exchange rate: %w", err)
	}

	c.AddDomainEvent(ExchangeRateUpdatedEvent{
		OccurredAt:     updatedAt,
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRateByID.ID,
		Rate:           exchangeRateByID.Rate.V(),
		From:           exchangeRateByID.From.V(),
	})
	return nil
}

func (c *Currency) RemoveExchangeRate(exchangeRateID ExchangeRateID, updatedAt time.Time) error {
	var exchangeRate *ExchangeRate
	for _, e := range c.ExchangeRates {
		if e.ID == exchangeRateID.V() {
			exchangeRate = e
		}
	}
	if exchangeRate == nil {
		return NewNotFoundError("ExchangeRate", "ID", exchangeRateID.String())
	}

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
		OccurredAt:     updatedAt,
		CurrencyID:     c.ID,
		ExchangeRateID: exchangeRate.ID,
	})
	return nil
}

func (c *Currency) RemoveCurrency(removeAt time.Time) error {
	today := DateFromTime(removeAt)
	canRemove := !slices.ContainsFunc(c.ExchangeRates, func(e *ExchangeRate) bool {
		return !e.From.V().After(today)
	})
	if !canRemove {
		return NewDomainError(
			CurrencyRemoveRequiresFutureFrom,
			fmt.Sprintf("remove currency requires all exchange rates to have from after today %s", today))
	}

	for i := len(c.ExchangeRates) - 1; i >= 0; i-- {
		if err := c.RemoveExchangeRate(MustParseExchangeRateId(c.ExchangeRates[i].ID), removeAt); err != nil {
			return fmt.Errorf("remove currency: %w", err)
		}
	}

	c.AddDomainEvent(CurrencyRemovedEvent{
		OccurredAt: removeAt,
		ID:         c.ID,
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
	parser := &RequestParseCollector{}
	id := parser.Parse("ID", req.ID, ParseCurrencyId)
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	if parser.HasErrors() {
		return parser
	}

	exist, err := h.Currencies.ExistByID(ctx, id)
	if err != nil {
		return err
	}
	if exist {
		return NewConflictError("Currency", "ID", id.String())
	}

	exist, err = h.Currencies.ExistByCode(ctx, code)
	if err != nil {
		return err // TODO(rh): add context (and other places)
	}
	if exist {
		return NewConflictError("Currency", "Code", code.V())
	}

	currency := NewCurrency(id, code, h.Clock.NowUTC())
	return h.Projector.Apply(ctx, &currency)
}

type RemoveCurrencyCommand struct {
	Code string
}

func (r RemoveCurrencyCommand) Validate(err *RequestParseCollector) {
}

type RemoveCurrencyHandler struct {
	Currencies CurrencyStore
	Projector  StoreProjector
	Clock      Clock
}

func (h RemoveCurrencyHandler) Handle(ctx context.Context, req RemoveCurrencyCommand) error {
	parser := &RequestParseCollector{}
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	if parser.HasErrors() {
		return parser
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
	parser := &RequestParseCollector{}
	id := parser.Parse("ID", req.ID, ParseExchangeRateId)
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	rate := parser.Parse("Rate", req.Rate, ParseRate)
	from := parser.Parse("From", req.From, ParseExchangeRateFrom)
	if parser.HasErrors() {
		return parser
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	now := h.Clock.NowUTC()
	exchangeRate := NewExchangeRate(id, rate, from, now)
	err = currency.AddExchangeRate(exchangeRate, now)
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
	parser := &RequestParseCollector{}
	id := parser.Parse("ID", req.ID, ParseExchangeRateId)
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	rate := parser.Parse("Rate", req.Rate, ParseRate)
	from := parser.Parse("From", req.From, ParseExchangeRateFrom)
	if parser.HasErrors() {
		return parser
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	if err := currency.UpdateExchangeRate(id, rate, from, h.Clock.NowUTC()); err != nil {
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
	parser := &RequestParseCollector{}
	id := parser.Parse("ID", req.ID, ParseExchangeRateId)
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	if parser.HasErrors() {
		return parser
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if currency == nil {
		return NewNotFoundError("Currency", "Code", code.V())
	}

	if err := currency.RemoveExchangeRate(id, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, currency)
}

type GetCurrencyQuery struct {
	Code string
}

type ExchangeRateResponse struct {
	ID        uuid.UUID
	Rate      float64
	From      Date
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type CurrencyResponse struct {
	ID            uuid.UUID
	Code          string
	ExchangeRates []*ExchangeRateResponse
	CreatedAt     time.Time
	UpdatedAt     *time.Time
}

type GetCurrencyHandler struct {
	Currencies CurrencyStore
}

func (h GetCurrencyHandler) Handle(ctx context.Context, req GetCurrencyQuery) (*CurrencyResponse, error) {
	parser := &RequestParseCollector{}
	code := parser.Parse("Code", req.Code, ParseCurrencyCode)
	if parser.HasErrors() {
		return nil, parser
	}

	currency, err := h.Currencies.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if currency == nil {
		return nil, NewNotFoundError("Currency", "Code", code.V())
	}

	exchangeRates := make([]*ExchangeRateResponse, len(currency.ExchangeRates))
	for i, e := range currency.ExchangeRates {
		exchangeRates[i] = &ExchangeRateResponse{
			ID:        e.ID,
			Rate:      e.Rate.V(),
			From:      e.From.V(),
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		}
	}

	return &CurrencyResponse{
		ID:            currency.ID,
		Code:          currency.Code.V(),
		ExchangeRates: exchangeRates,
		CreatedAt:     currency.CreatedAt,
		UpdatedAt:     currency.UpdatedAt,
	}, nil
}
