package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
)

type SqlDomainEventStore struct {
	Tx *sql.Tx
}

func (r SqlDomainEventStore) GetByAggregateId(ctx context.Context, id uuid.UUID, limit dseedwork.Limit, cursor *dseedwork.Cursor) dseedwork.Paged[seedwork.PersistedDomainEvent] {
	const sql = `
		select id, aggregate_id, aggregate_type, event_type, event_payload, created_at
		from domain_events
		where aggregate_id = ?
		and created_at > ?
		order by created_at
		limit ?`

	offset, err := cursorToOffset(cursor)
	if err != nil {
		panic(err)
	}
	res, err := r.Tx.QueryContext(ctx, sql, id, offset, limit.Value)
	if err != nil {
		panic(err)
	}

	var (
		eventId, aggregateId, aggregateType, eventType, eventPayload string
		createdAt                                                    int64
	)

	events := make([]seedwork.PersistedDomainEvent, 0)
	for res.Next() {
		err := res.Scan(&eventId, &aggregateId, &aggregateType, &eventType, &eventPayload, &createdAt)
		if err != nil {
			panic(err)
		}
		eventId_, err := uuid.Parse(eventId)
		if err != nil {
			panic(err)
		}
		aggregateId_, err := uuid.Parse(aggregateId)
		if err != nil {
			panic(err)
		}

		e := seedwork.PersistedDomainEvent{
			Id:            eventId_,
			AggregateId:   aggregateId_,
			AggregateType: aggregateType,
			EventType:     eventType,
			EventPayload:  eventPayload,
			CreatedAt:     parseCreatedAt(createdAt),
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		return dseedwork.Paged[seedwork.PersistedDomainEvent]{
			Cursor: nil,
			Items:  events,
		}
	}

	pageEndOffset := events[len(events)-1].CreatedAt.UnixNano()
	globalEndOffset := getLargestCreatedAt("domain_events", r.Tx)
	newCursor := offsetsToCursor(pageEndOffset, globalEndOffset)
	return dseedwork.Paged[seedwork.PersistedDomainEvent]{
		Cursor: newCursor,
		Items:  events,
	}
}

type Clock struct{}

func (c Clock) UtcNow() time.Time {
	return time.Now().UTC()
}
