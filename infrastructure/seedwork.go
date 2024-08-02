package infrastructure

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
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

// TODO: should cursorToOffset return error or AssertNoErr? A user could've
// tampered with the values (or bug in client app adding those to URL) so we
// don't want to record it as a server bug. What does F# do?

func cursorToOffset(c *dseedwork.Cursor) (*int64, error) {
	if c != nil {
		bytes, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to decode cursor %s: %w", c.Value, err)
		}
		decoded := string(bytes)
		offset, err := strconv.ParseInt(decoded, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse cursor %s: %w", decoded, err)
		}
		return &offset, nil

	}
	var defaultOffset int64 = 0
	return &defaultOffset, nil
}

func offsetsToCursor(pageEndOffset, globalEndOffset int64) *dseedwork.Cursor {
	if pageEndOffset == globalEndOffset {
		return nil
	}
	a := strconv.FormatInt(pageEndOffset, 10)
	b := base64.StdEncoding.EncodeToString([]byte(a))
	c, err := dseedwork.NewCursor(b)
	if err != nil {
		panic(fmt.Errorf("invalid cursor %s: %w", b, err))
	}
	return c
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
