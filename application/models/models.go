package models

type PagedDto[T any] struct {
	Cursor *string
	Items  []T
}
