package core

const (
	ExpectedDifferentResellerRole = 1200
	ExpectedDifferentCurrencyCode = 1201
)

type ResellerBillingItem struct {
	Entity
	ProductCode          ProductCode // Example of foreign key not being a UUID because domain
	GrossRevenue         Money
	CalculatedNetRevenue Money
}

type ResellerBillingKind string

var ResellerBillingKinds = map[ResellerBillingKind]struct{}{
	"Invoice": {}, "CreditMemo": {},
}

type ResellerRole string

var ResellerRoles = map[ResellerRole]struct{}{
	"Orphan": {}, "Head": {}, "Member": {},
}

type DocumentNumber string

type ResellerBilling struct {
	Entity
	DocumentNumber       DocumentNumber
	BookedAt             Date
	CurrencyCode         string
	ResellerBillingKind  ResellerBillingKind
	CalculatedNetRevenue Money
	ResellerBillingItems []ResellerBillingItem
}

type Reseller struct {
	AggregateRoot
	CountryCode                    string
	CurrencyCode                   string
	EnrolledAt                     Date
	ResellerRole                   ResellerRole
	CalculatedNetRevenueYearToDate *Money
	CalculatedNetRevenueLastYear   *Money
	ResellerBillings               []ResellerBilling
}
