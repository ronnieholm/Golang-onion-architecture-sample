package seedwork

import (
	"time"

	"github.com/ronnieholm/golang-onion-architecture-sample/domain/validation"
)

type Entity[TId any] struct {
	Id        TId
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type AggregateRoot[TId any] struct {
	Id        TId
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type DomainEvent struct {
	OccurredAt time.Time
}

type Limit struct {
	Value int
}

func NewLimit(value int) (*Limit, error) {
	if err := validation.IntBetween(1, 100, value); err != nil {
		return nil, err
	}
	return &Limit{value}, nil
}

type Cursor struct {
	Value string
}

func NewCursor(value string) (*Cursor, error) {
	if err := validation.StringNotNilOrWhitespace(value); err != nil {
		return nil, err
	}
	return &Cursor{value}, nil
}

type Paged[T any] struct {
	Cursor *Cursor
	Items  []T
}
