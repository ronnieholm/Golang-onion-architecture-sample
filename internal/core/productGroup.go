package core

import "time"

const (
	ProductGroupCodeExpectedFutureFromForAdd            = 1100
	ProductGroupCodeExpectedFutureFromForUpdate         = 1101
	ProductGroupCodeExpectedFutureFromForWeightRemoval  = 1102
	ProductGroupCodeExpectedDifferentProductGroupWeight = 1103
)

type ProductGroupCode string

type ProductGroupWeight struct {
	Entity
	Percentage float64
	From       time.Time
}

type ProductGroup struct {
	AggregateRoot
	Code                ProductGroupCode
	ProductGroupWeights []ProductGroupWeight
}
