package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Domain

type TierDiscountStore interface {
	ExistByID(context.Context, TierDiscountID) (bool, error)
	GetByID(context.Context, TierDiscountID) (*TierDiscount, error)
}

type TierDiscountCreatedEvent struct {
	domainEventCommon
	ID         uuid.UUID
	Authorized float64
	Advanced   float64
	Premier    float64
	From       Date
}

type TierDiscountUpdatedEvent struct {
	domainEventCommon
	ID         uuid.UUID
	Authorized float64
	Advanced   float64
	Premier    float64
	From       Date
}

type TierDiscountRemovedEvent struct {
	domainEventCommon
	ID uuid.UUID
}

const (
	TierDiscountCreateRequiresFututureFrom = 1600
	TierDiscountUpdateRequiresFutureFrom   = 1601
	TierDiscountRemoveRequiresFutureFrom   = 1602
	TierDiscountUpdateRequiresChange       = 1603
)

// TierDiscountID

type TierDiscountID struct {
	v uuid.UUID
}

func (t TierDiscountID) V() uuid.UUID   { return t.v }
func (t TierDiscountID) String() string { return t.v.String() }

func ParseTierDiscountID(v uuid.UUID) (TierDiscountID, error) {
	if err := ValidateUUIDNotZero(v); err != nil {
		return TierDiscountID{}, err
	}
	return TierDiscountID{v}, nil
}

func MustParseTierDiscountId(v uuid.UUID) TierDiscountID {
	v1, err := ParseTierDiscountID(v)
	if err != nil {
		panic(err)
	}
	return v1
}

// DiscountPercentages

const (
	DiscountPercentageMin float64 = 0.
	DiscountPercentageMax float64 = 100.
)

type DiscountPercentages struct {
	authorized float64
	advanced   float64
	premier    float64
}

func (c DiscountPercentages) Authorized() float64 { return c.authorized }
func (c DiscountPercentages) Advanced() float64   { return c.advanced }
func (c DiscountPercentages) Premier() float64    { return c.premier }

func ParseDiscountPercentages(authorized, advanced, premier float64) (DiscountPercentages, error) {
	errs := &FieldParseError{}
	if err := ValidateFloat64InclusiveRange(authorized, DiscountPercentageMin, DiscountPercentageMax); err != nil {
		errs.Add(err.Error()) // TODO(rh): field name would be missing from error.
	}
	if err := ValidateFloat64InclusiveRange(advanced, DiscountPercentageMin, DiscountPercentageMax); err != nil {
		errs.Add(err.Error())
	}
	if err := ValidateFloat64InclusiveRange(premier, DiscountPercentageMin, DiscountPercentageMax); err != nil {
		errs.Add(err.Error())
	}
	if authorized > advanced {
		message := fmt.Sprintf("Must be between 0 and %g inclusive, but was %g", advanced, authorized)
		errs.Add(message)
	}
	if advanced > premier {
		message := fmt.Sprintf("Must be between %g and %g inclusive, but was %g", authorized, premier, advanced)
		errs.Add(message)
	}

	if err := errs.NilOrError(); err != nil {
		return DiscountPercentages{}, err
	}

	return DiscountPercentages{
		authorized: authorized,
		advanced:   advanced,
		premier:    premier,
	}, nil
}

func MustParseDiscountPercentages(authorized, advanced, premier float64) DiscountPercentages {
	v1, err := ParseDiscountPercentages(authorized, advanced, premier)
	if err != nil {
		panic(err)
	}
	return v1
}

type TierDiscount struct {
	AggregateRoot
	Percentages DiscountPercentages
	From        From
}

func NewTierDiscount(id TierDiscountID, percentages DiscountPercentages, from From, createdAt time.Time) TierDiscount {
	td := TierDiscount{
		AggregateRoot: AggregateRoot{
			Entity: Entity{
				ID:        id.V(),
				CreatedAt: createdAt,
			},
		},
		From: from,
	}

	td.AddDomainEvent(TierDiscountCreatedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: createdAt,
		},
		ID:         id.V(),
		Authorized: percentages.Authorized(),
		Advanced:   percentages.Advanced(),
		Premier:    percentages.Premier(),
		From:       from.V(),
	})
	return td
}

func (td *TierDiscount) Update(percentages DiscountPercentages, from From, updatedAt time.Time) error {
	today := DateFromTime(updatedAt)
	if !from.V().After(today) {
		return NewDomainError(
			TierDiscountUpdateRequiresFutureFrom,
			fmt.Sprintf("update tier discount requires from %s be after today %s", from, today.String()))
	}
	if td.Percentages == percentages && td.From == from {
		return NewDomainError(
			TierDiscountUpdateRequiresChange,
			"update tier discount requires a change to tier discount")
	}

	td.Percentages = percentages
	td.From = from
	td.UpdatedAt = &updatedAt

	td.AddDomainEvent(TierDiscountUpdatedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: updatedAt,
		},
		ID:         td.ID,
		Authorized: percentages.Authorized(),
		Advanced:   percentages.Advanced(),
		Premier:    percentages.Premier(),
		From:       from.V(),
	})
	return nil
}

func (td *TierDiscount) Remove(removeAt time.Time) error {
	today := DateFromTime(removeAt)
	if !td.From.V().After(today) {
		return NewDomainError(
			TierDiscountRemoveRequiresFutureFrom,
			fmt.Sprintf("remove tier discount requires from %s to be after today %s", td.From.V(), today))
	}

	td.AddDomainEvent(TierDiscountRemovedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: removeAt,
		},
		ID: td.ID,
	})
	return nil
}

// Application

type DiscountPercentagesInput struct {
	Authorized float64
	Advanced   float64
	Premier    float64
}

type CreateTierDiscountCommand struct {
	ID          uuid.UUID
	Percentages DiscountPercentagesInput
	From        Date
}

type CreateTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h CreateTierDiscountHandler) Handle(ctx context.Context, req CreateTierDiscountCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseTierDiscountID)
	percentages := Parse(errs, "Percentages", req.Percentages, func(dp DiscountPercentagesInput) (DiscountPercentages, error) {
		return ParseDiscountPercentages(dp.Advanced, dp.Advanced, dp.Premier)
	})
	from := Parse(errs, "From", req.From, ParseFrom)
	if errs.HasErrors() {
		return errs
	}

	found, err := h.TierDiscounts.ExistByID(ctx, id)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("TierDiscount", "ID", id.String())
	}

	tierDiscount := NewTierDiscount(id, percentages, from, h.Clock.NowUTC())
	return h.Projector.Apply(ctx, &tierDiscount)
}

type UpdateTierDiscountCommand struct {
	ID          uuid.UUID
	Percentages DiscountPercentagesInput
	From        Date
}

type UpdateTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h UpdateTierDiscountHandler) Handle(ctx context.Context, req UpdateTierDiscountCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseTierDiscountID)
	percentages := Parse(errs, "Percentages", req.Percentages, func(dp DiscountPercentagesInput) (DiscountPercentages, error) {
		return ParseDiscountPercentages(dp.Advanced, dp.Advanced, dp.Premier)
	})
	from := Parse(errs, "From", req.From, ParseFrom)
	if errs.HasErrors() {
		return errs
	}

	tierDiscount, err := h.TierDiscounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tierDiscount == nil {
		return NewNotFoundError("TierDiscount", "ID", id.String())
	}

	if err := tierDiscount.Update(percentages, from, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, tierDiscount)
}

type RemoveTierDiscountCommand struct {
	ID uuid.UUID
}

type RemoveTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h RemoveTierDiscountHandler) Handle(ctx context.Context, req RemoveTierDiscountCommand) error {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseTierDiscountID)
	if errs.HasErrors() {
		return errs
	}

	tierDiscount, err := h.TierDiscounts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tierDiscount == nil {
		return NewNotFoundError("TierDiscount", "ID", id.String())
	}

	if err := tierDiscount.Remove(h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, tierDiscount)
}

type GetTierDiscountQuery struct {
	ID uuid.UUID
}

type GetTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
}

func (h GetTierDiscountHandler) Handle(ctx context.Context, req GetTierDiscountQuery) (*TierDiscount, error) {
	errs := &RequestParseError{}
	id := Parse(errs, "ID", req.ID, ParseTierDiscountID)
	if errs.HasErrors() {
		return nil, errs
	}

	tierDiscount, err := h.TierDiscounts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tierDiscount == nil {
		return nil, NewNotFoundError("TierDiscount", "ID", id.String())
	}
	return tierDiscount, nil
}
