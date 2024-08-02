package domainEventRequest

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/models"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/domain/validation"
)

type Store interface {
	GetByAggregateId(ctx context.Context, id uuid.UUID, limit dseedwork.Limit, cursor *dseedwork.Cursor) dseedwork.Paged[seedwork.PersistedDomainEvent]
}

// GetByAggregateIdQuery

type GetByAggregateIdQuery struct {
	Id     uuid.UUID
	Limit  int
	Cursor *string
}

type GetByAggregateIdValidatedQuery struct {
	Id     uuid.UUID
	Limit  dseedwork.Limit
	Cursor *dseedwork.Cursor
}

type PersistedDomainEventDto struct {
	Id            uuid.UUID
	AggregateId   uuid.UUID
	AggregateType string
	EventType     string
	EventPayload  string
	CreatedAt     time.Time
}

func fromPersistedDomainEvent(event seedwork.PersistedDomainEvent) PersistedDomainEventDto {
	return PersistedDomainEventDto{
		Id:            event.Id,
		AggregateId:   event.AggregateId,
		AggregateType: event.AggregateType,
		EventType:     event.EventType,
		EventPayload:  event.AggregateType,
		CreatedAt:     event.CreatedAt,
	}
}

func (q GetByAggregateIdQuery) validate() (*GetByAggregateIdValidatedQuery, error) {
	errs := seedwork.NewValidationErrors()
	err := validation.UuidNotEmpty(q.Id)
	errs.Set("Id", err)
	limit, err := dseedwork.NewLimit(q.Limit)
	errs.Set("Limit", err)

	var cursor *dseedwork.Cursor = nil
	if q.Cursor != nil {
		cursor, err = dseedwork.NewCursor(*q.Cursor)
		errs.Set("Cursor", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &GetByAggregateIdValidatedQuery{
		Id:     q.Id,
		Limit:  *limit,
		Cursor: cursor,
	}, nil
}

func (q GetByAggregateIdQuery) Run(ctx context.Context, identity seedwork.ScrumIdentity, events Store) (*models.PagedDto[PersistedDomainEventDto], error) {
	if !identity.IsInRole(seedwork.ScrumRoleAdmin) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleAdmin}
	}

	qry, err := q.validate()
	if err != nil {
		return nil, err
	}

	eventsPage := events.GetByAggregateId(ctx, qry.Id, qry.Limit, qry.Cursor)
	eventDtos := make([]PersistedDomainEventDto, 0, len(eventsPage.Items))
	for _, event := range eventsPage.Items {
		eventDtos = append(eventDtos, fromPersistedDomainEvent(event))
	}

	var cursor *string = nil
	if eventsPage.Cursor != nil {
		cursor = &eventsPage.Cursor.Value
	}

	return &models.PagedDto[PersistedDomainEventDto]{
		Cursor: cursor,
		Items:  eventDtos,
	}, nil
}
