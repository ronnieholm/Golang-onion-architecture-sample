package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Domain

type TierDiscountStore interface {
	ExistByID(context.Context, uuid.UUID) (bool, error)
	GetByID(context.Context, uuid.UUID) (*TierDiscount, error)
}

type TierDiscountCreatedEvent struct {
	domainEventCommon
	ID                   uuid.UUID
	AuthorizedPercentage float64
	AdvancedPercentage   float64
	PremierPercentage    float64
	From                 Date
}

type TierDiscountUpdatedEvent struct {
	domainEventCommon
	ID                   uuid.UUID
	AuthorizedPercentage float64
	AdvancedPercentage   float64
	PremierPercentage    float64
	From                 Date
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

type DiscountPercentages struct {
	Authorized float64
	Advanced   float64
	Premier    float64
}

type TierDiscount struct {
	AggregateRoot
	Percentages DiscountPercentages
	From        Date
}

func NewTierDiscount(id uuid.UUID, percentages DiscountPercentages, from Date, createdAt time.Time) TierDiscount {
	td := TierDiscount{
		AggregateRoot: AggregateRoot{
			Entity: Entity{
				ID:        id,
				CreatedAt: createdAt,
			},
		},
		From: from,
	}

	td.AddDomainEvent(TierDiscountCreatedEvent{
		domainEventCommon: domainEventCommon{
			OccurredAt: createdAt,
		},
		ID:                   id,
		AuthorizedPercentage: percentages.Authorized,
		AdvancedPercentage:   percentages.Advanced,
		PremierPercentage:    percentages.Premier,
		From:                 from,
	})
	return td
}

func (td *TierDiscount) Update(percentages DiscountPercentages, from Date, updatedAt time.Time) error {
	today := DateFromTime(updatedAt)
	if !from.After(today) {
		return NewDomainError(
			TierDiscountUpdateRequiresFutureFrom,
			fmt.Sprintf("update tier discount requires from %s be after today %s", from.String(), today.String()))
	}
	if td.Percentages == percentages && td.From.Equal(from) {
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
		ID:                   td.ID,
		AuthorizedPercentage: percentages.Authorized,
		AdvancedPercentage:   percentages.Advanced,
		PremierPercentage:    percentages.Premier,
		From:                 from,
	})
	return nil
}

func (td *TierDiscount) Remove(removeAt time.Time) error {
	today := DateFromTime(removeAt)
	if !td.From.After(today) {
		return NewDomainError(
			TierDiscountRemoveRequiresFutureFrom,
			fmt.Sprintf("remove tier discount requires from %s to be after today %s", td.From, today))
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

type CreateTierDiscountCommand struct {
	ID                   uuid.UUID
	AuthorizedPercentage float64
	AdvancedPercentage   float64
	PremierPercentage    float64
	From                 Date
}

func (req CreateTierDiscountCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
	ValidateDiscountPercentages("AuthorizedPercentage", "AdvancedPercentage", "PremierPercentage", req.AuthorizedPercentage, req.AdvancedPercentage, req.PremierPercentage, err)
	ValidateDateInclusiveRange("From", req.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

type CreateTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h CreateTierDiscountHandler) Handle(ctx context.Context, req CreateTierDiscountCommand) error {
	found, err := h.TierDiscounts.ExistByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if found {
		return NewConflictError("TierDiscount", "ID", req.ID.String())
	}

	dp := DiscountPercentages{
		Authorized: req.AuthorizedPercentage,
		Advanced:   req.AdvancedPercentage,
		Premier:    req.PremierPercentage,
	}
	tierDiscount := NewTierDiscount(req.ID, dp, req.From, h.Clock.NowUTC())
	return h.Projector.Apply(ctx, &tierDiscount)
}

type UpdateTierDiscountCommand struct {
	ID                   uuid.UUID
	AuthorizedPercentage float64
	AdvancedPercentage   float64
	PremierPercentage    float64
	From                 Date
}

func (req UpdateTierDiscountCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
	ValidateDiscountPercentages("AuthorizedPercentage", "AdvancedPercentage", "PremierPercentage", req.AuthorizedPercentage, req.AdvancedPercentage, req.PremierPercentage, err)
	ValidateDateInclusiveRange("From", req.From, MinExchangeRateFrom, MaxExchangeRateFrom, err)
}

type UpdateTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h UpdateTierDiscountHandler) Handle(ctx context.Context, req UpdateTierDiscountCommand) error {
	tierDiscount, err := h.TierDiscounts.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if tierDiscount == nil {
		return NewNotFoundError("TierDiscount", "ID", req.ID.String())
	}

	dp := DiscountPercentages{
		Authorized: req.AuthorizedPercentage,
		Advanced:   req.AdvancedPercentage,
		Premier:    req.PremierPercentage,
	}
	if err := tierDiscount.Update(dp, req.From, h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, tierDiscount)
}

type RemoveTierDiscountCommand struct {
	ID uuid.UUID
}

func (req RemoveTierDiscountCommand) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
}

type RemoveTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
	Projector     StoreProjector
	Clock         Clock
}

func (h RemoveTierDiscountHandler) Handle(ctx context.Context, req RemoveTierDiscountCommand) error {
	tierDiscount, err := h.TierDiscounts.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if tierDiscount == nil {
		return NewNotFoundError("TierDiscount", "ID", req.ID.String())
	}

	if err := tierDiscount.Remove(h.Clock.NowUTC()); err != nil {
		return err
	}
	return h.Projector.Apply(ctx, tierDiscount)
}

type GetTierDiscountQuery struct {
	ID uuid.UUID
}

func (req GetTierDiscountQuery) Validate(err *ValidationError) {
	ValidateUUIDNotZero("ID", req.ID, err)
}

type GetTierDiscountHandler struct {
	TierDiscounts TierDiscountStore
}

func (h GetTierDiscountHandler) Handle(ctx context.Context, req GetTierDiscountQuery) (*TierDiscount, error) {
	tierDiscount, err := h.TierDiscounts.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if tierDiscount == nil {
		return nil, NewNotFoundError("TierDiscount", "ID", req.ID.String())
	}
	return tierDiscount, nil
}
