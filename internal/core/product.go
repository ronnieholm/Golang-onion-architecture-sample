package core

import (
	"time"

	"github.com/google/uuid"
)

type ProductRepository interface {
}

type ProductCode string

type ProductCreatedEvent struct {
	ProductId   uuid.UUID
	ProductCode ProductCode
}

type ProductGroupAssignedEvent struct {
	ProductCode      ProductCode
	ProductGroupCode ProductGroupCode
}

type ProductGroupUnassignedEvent struct {
	ProductCode ProductCode
}

const (
	ProductExpectedDifferentProductGroup = 1000
	ProductExpectedProductGroupSet       = 1001
)

type Product struct {
	AggregateRoot
	Code ProductCode
	// TODO(rh): Navigation property: ProductGroup ProductGroup
}

func NewProduct(id uuid.UUID, code ProductCode, createdAt time.Time) Product {
	return Product{}
}

func (p *Product) AssignProductGroup(productGroup ProductGroup, updatedAt time.Time) {

}

func (p *Product) UnassignProductGroup(updatedAt time.Time) {

}
