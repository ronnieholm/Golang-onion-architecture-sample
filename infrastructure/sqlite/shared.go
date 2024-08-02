package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func parseCreatedAt(t int64) time.Time {
	return time.Unix(0, t)
}

func parseUpdatedAt(t *int64) *time.Time {
	if t != nil {
		t := time.Unix(0, *t)
		return &t
	}
	return nil
}

func getLargestCreatedAt(table string, tx *sql.Tx) int64 {
	sql := fmt.Sprintf("select created_at from %s order by created_at desc limit 1", table)
	res := tx.QueryRow(sql, table)
	var createdAt int64
	err := res.Scan(&createdAt)
	if err != nil {
		panic(fmt.Errorf("unable to scan createdAt: %s", err))
	}
	return createdAt
}

func persistDomainEvent(tx *sql.Tx, aggregateType string, aggregateId uuid.UUID, eventType string, payload string, createdAt time.Time) {
	const sql = "insert into domain_events (id, aggregate_type, aggregate_id, event_type, event_payload, created_at) values (?, ?, ?, ?, ?, ?)"
	res, err := tx.Exec(sql, uuid.New(), aggregateType, aggregateId.String(), eventType, payload, createdAt.Nanosecond())
	if err != nil {
		panic(fmt.Errorf("unable to insert domain event: %w", err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		panic(fmt.Errorf("unable to get rows affected: %w", err))
	}
	if n != 1 {
		panic(fmt.Sprintf("rows affected expected 1, was %d", n))
	}
}
