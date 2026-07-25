package core

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Assert(expr bool, format string, args ...any) {
	if !expr {
		panic(fmt.Sprintf(format, args...))
	}
}

// DomainEvent interface is needed to treat events uniformly. Otherwise, it
// isn't possible to add multiple types as DomainEvent to an Aggregate.
type DomainEvent interface {
	isDomainEvent()
	At() time.Time
}

type domainEventCommon struct {
	OccurredAt time.Time `json:"-"`
}

func (domainEventCommon) isDomainEvent() {}

func (d domainEventCommon) At() time.Time {
	return d.OccurredAt
}

type Entity struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type Identifiable interface {
	GetID() uuid.UUID
}

func (e Entity) GetID() uuid.UUID {
	return e.ID
}

// EntityEqual compares any two pointers to types that implement Identifiable.
// Because Go doesn't support generic methods, on concrete entity types the
// Equal method is still best used to minimize friction with slices.IndexFunc
// and similar.
func EntityEqual[T Identifiable](a, b *T) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return (*a).GetID() == (*b).GetID()
}

func ValueEquals[T comparable](v, other T) bool {
	return v == other
}

type AggregateRoot struct {
	Entity
	Version      int32
	DomainEvents []DomainEvent
}

func (a *AggregateRoot) AddDomainEvent(e DomainEvent) {
	if a.DomainEvents == nil {
		a.DomainEvents = []DomainEvent{}
	}
	a.DomainEvents = append(a.DomainEvents, e)
}

func (a *AggregateRoot) ClearDomainEvents() {
	a.DomainEvents = []DomainEvent{}
}

type Aggregate interface {
	GetAggregateRoot() *AggregateRoot
}

func (a *AggregateRoot) GetAggregateRoot() *AggregateRoot {
	return a
}

func stringsToMap(s ...string) map[string]string {
	if len(s)%2 != 0 {
		panic("expected equal number of elements")
	}
	m := make(map[string]string, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		_, ok := m[s[i]]
		if ok {
			panic("expected unique field")
		}
		m[s[i]] = s[i+1]
	}
	return m
}

type DataStaleError struct {
	Aggregate string
	ID        uuid.UUID
}

func NewDataStaleError(aggregate string, id uuid.UUID) *DataStaleError {
	return &DataStaleError{Aggregate: aggregate, ID: id}
}

func (e *DataStaleError) Error() string {
	return fmt.Sprintf("data stale for aggregate %s with id: %s", e.Aggregate, e.ID)
}

type ConflictError struct {
	Entity      string
	FieldValues map[string]string
}

func NewConflictError(entity string, fieldValues ...string) *ConflictError {
	return &ConflictError{Entity: entity, FieldValues: stringsToMap(fieldValues...)}
}

func (e *ConflictError) HasErrors() bool {
	return len(e.FieldValues) > 0
}

func (e *ConflictError) Error() string {
	if !e.HasErrors() {
		return ""
	}

	var b = strings.Builder{}
	b.WriteString(e.Entity)
	b.WriteString(" conflicts: ")

	i := 0
	for f, v := range e.FieldValues {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f)
		b.WriteString(": ")
		b.WriteString(v)
		i++
	}
	return b.String()
}

type NotFoundError struct {
	Entity      string
	FieldValues map[string]string
}

func NewNotFoundError(entity string, fieldValues ...string) *NotFoundError {
	return &NotFoundError{Entity: entity, FieldValues: stringsToMap(fieldValues...)}
}

func (e *NotFoundError) HasErrors() bool {
	return len(e.FieldValues) > 0
}

func (e *NotFoundError) Error() string {
	if !e.HasErrors() {
		return ""
	}

	var b = strings.Builder{}
	b.WriteString(e.Entity)
	b.WriteString(" not found: ")

	i := 0
	for f, v := range e.FieldValues {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f)
		b.WriteString(": ")
		b.WriteString(v)
		i++
	}
	return b.String()
}

type DomainError struct {
	Code    int
	Message string
}

func NewDomainError(code int, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("domain error %d: %s", e.Code, e.Message)
}

type Clock interface {
	NowUTC() time.Time
	Today() Date
}

type StoreProjector interface {
	Apply(context.Context, ...Aggregate) error
}

// TODO(rh): Instead of FieldParseError, use existing RequestParserError. If key
// is nil, Parse function should assume field was what's passed to Parse. For
// multiple fields, such as DiscountPercentage compound value object, key would
// be the three field names. The assumption being whenever in a request the
// discounts are present, they're always named the same as field names are hard-
// code in the parser. It would be more elegant if it weren't so because now
// Parse is passed field name for simple value objects.
type FieldParseError struct {
	Messages []string
}

func (e *FieldParseError) Error() string {
	return strings.Join(e.Messages, "; ")
}

func (e *FieldParseError) Add(message string) {
	e.Messages = append(e.Messages, message)
}

func (e *FieldParseError) NilOrError() error {
	if len(e.Messages) == 0 {
		return nil
	}
	return e
}

// TODO(rh): Can this be a member of RequestParseError now Go support method generics? Then RequestParseError can become private. Part of Go 1.27 to be released in Aug 2026. Preview available.
// Parse, don't validate pattern.
func Parse[In any, Out any](e *RequestParserError, field string, val In, fn func(In) (Out, error)) Out {
	res, err := fn(val)
	if err != nil {
		// TOOD(rh): include field name in FieldParseError or consumer will have a hard time relating multiple errors to fields. Have Parse
		// append field + "." + FieldParseError.Field if field is set.
		if errs2, ok := err.(*FieldParseError); ok {
			// When parsing a value object may lead to zero or many errors.
			for _, msg := range errs2.Messages {
				e.Add(field, msg)
			}
		} else {
			// When parsing a value object may lead to zero or one error.
			e.Add(field, err.Error())
		}
	}
	return res
}

// ISO 4217 country codes per https://en.wikipedia.org/wiki/ISO_4217.
var CountryCodes = map[string]struct{}{
	"AD": {}, "AE": {}, "AF": {}, "AG": {}, "AI": {}, "AL": {}, "AM": {},
	"AN": {}, "BE": {}, "BF": {}, "BG": {}, "BH": {}, "BI": {}, "BJ": {},
	"BL": {}, "BM": {}, "BN": {}, "BO": {}, "BQ": {}, "AR": {}, "AS": {},
	"AO": {}, "AQ": {}, "AT": {}, "AU": {}, "AW": {}, "AX": {}, "AZ": {},
	"BA": {}, "BB": {}, "BD": {}, "CG": {}, "CH": {}, "CI": {}, "CK": {},
	"CL": {}, "CM": {}, "CN": {}, "CO": {}, "CR": {}, "CU": {}, "CV": {},
	"CW": {}, "CX": {}, "CY": {}, "CZ": {}, "DE": {}, "DJ": {}, "DK": {},
	"DM": {}, "DO": {}, "DZ": {}, "EC": {}, "EE": {}, "EG": {}, "EH": {},
	"ER": {}, "ES": {}, "ET": {}, "FI": {}, "FJ": {}, "FK": {}, "FM": {},
	"FO": {}, "FR": {}, "GA": {}, "GB": {}, "GD": {}, "BR": {}, "BS": {},
	"BT": {}, "BV": {}, "BW": {}, "BY": {}, "BZ": {}, "CA": {}, "CC": {},
	"CD": {}, "CF": {}, "GF": {}, "GG": {}, "GH": {}, "GI": {}, "GL": {},
	"GE": {}, "GM": {}, "GN": {}, "GP": {}, "GQ": {}, "GR": {}, "GS": {},
	"GT": {}, "GU": {}, "GW": {}, "GY": {}, "HK": {}, "HM": {}, "HN": {},
	"HR": {}, "HT": {}, "HU": {}, "ID": {}, "IE": {}, "IL": {}, "IM": {},
	"IN": {}, "IO": {}, "IQ": {}, "IR": {}, "IS": {}, "IT": {}, "JE": {},
	"JM": {}, "JO": {}, "JP": {}, "KE": {}, "KG": {}, "KH": {}, "KI": {},
	"KM": {}, "KN": {}, "KP": {}, "KR": {}, "KW": {}, "KY": {}, "KZ": {},
	"LA": {}, "LB": {}, "LC": {}, "LI": {}, "LK": {}, "LR": {}, "LS": {},
	"LT": {}, "LU": {}, "LV": {}, "LY": {}, "MA": {}, "MC": {}, "MD": {},
	"ME": {}, "MF": {}, "MG": {}, "MV": {}, "MT": {}, "MU": {}, "MW": {},
	"MX": {}, "MY": {}, "MZ": {}, "NA": {}, "NC": {}, "NE": {}, "NF": {},
	"NG": {}, "NI": {}, "NL": {}, "NO": {}, "NP": {}, "NR": {}, "NU": {},
	"NZ": {}, "OM": {}, "PA": {}, "PE": {}, "PF": {}, "PG": {}, "PH": {},
	"ML": {}, "MI": {}, "MH": {}, "MK": {}, "MM": {}, "MN": {}, "MO": {},
	"MP": {}, "MQ": {}, "MR": {}, "MS": {}, "RU": {}, "RW": {}, "RO": {},
	"RS": {}, "SA": {}, "SB": {}, "SC": {}, "SD": {}, "SE": {}, "SG": {},
	"PN": {}, "PK": {}, "PL": {}, "PM": {}, "PR": {}, "PS": {}, "PT": {},
	"PW": {}, "PY": {}, "QA": {}, "RE": {}, "VI": {}, "VA": {}, "VC": {},
	"VE": {}, "VG": {}, "VN": {}, "VU": {}, "WF": {}, "WK": {}, "WS": {},
	"XK": {}, "XT": {}, "YE": {}, "YT": {}, "ZA": {}, "ZM": {}, "ZW": {},
	"SH": {}, "SI": {}, "SJ": {}, "SK": {}, "SL": {}, "SM": {}, "SN": {},
	"SO": {}, "SR": {}, "SS": {}, "ST": {}, "SV": {}, "SX": {}, "SY": {},
	"SZ": {}, "TC": {}, "TD": {}, "TF": {}, "TG": {}, "TH": {}, "TJ": {},
	"TK": {}, "TL": {}, "TM": {}, "TN": {}, "TO": {}, "TR": {}, "TT": {},
	"TV": {}, "TW": {}, "TZ": {}, "UA": {}, "UG": {}, "UM": {}, "US": {},
	"UY": {}, "UZ": {},
}

// ISO 4217 country codes per https://en.wikipedia.org/wiki/ISO_4217.
var CurrencyCodes = map[string]struct{}{
	"USD": {}, "EUR": {}, "DKK": {},
}

type CurrencyCode struct {
	v string
}

func (c CurrencyCode) V() string { return c.v }

func ParseCurrencyCode(v string) (CurrencyCode, error) {
	if err := ValidateStringCurrencyCode(v); err != nil {
		return CurrencyCode{}, err
	}
	return CurrencyCode{v}, nil
}

func MustParseCurrencyCode(v string) CurrencyCode {
	v1, err := ParseCurrencyCode(v)
	if err != nil {
		panic(err)
	}
	return v1
}

// Money in this system doesn't require the accuracy afforded by a decimal type.
type Money struct {
	Amount float64
	Code   string
}

const layout = "2006-01-02"

// TODO(rh): add tests of Date.

// Date holds a date without time of day.
type Date struct {
	time.Time
}

// NewDate creates a Date from year, month, day.
func NewDate(year int, month time.Month, day int) Date {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return Date{t}
}

// DateFromTime converts a time.Time to Date.
func DateFromTime(t time.Time) Date {
	u := t.In(time.UTC)
	return NewDate(u.Year(), u.Month(), u.Day())
}

// Parse parses "YYYY-MM-DD".
func ParseDate(s string) (Date, error) {
	if s == "" {
		return Date{}, errors.New("empty date string")
	}
	t, err := time.ParseInLocation(layout, s, time.UTC)
	if err != nil {
		return Date{}, fmt.Errorf("unable to parse string %s: %w", s, err)
	}
	t2 := DateFromTime(t)
	return t2, nil
}

// String returns "YYYY-MM-DD".
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Format(layout)
}

// MarshalJSON implements json.Marshaler (string "YYYY-MM-DD").
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(b []byte) error {
	// handle null
	if string(b) == "null" || len(b) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	nd, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = nd
	return nil
}

// Value implements driver.Valuer for database/sql (stores as "YYYY-MM-DD").
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.String(), nil
}

// Scan implements sql.Scanner for database/sql.
func (d *Date) Scan(source any) error {
	if source == nil {
		*d = Date{}
		return nil
	}
	switch v := source.(type) {
	case time.Time:
		*d = DateFromTime(v)
		return nil
	case []byte:
		nd, err := ParseDate(string(v))
		if err != nil {
			return err
		}
		*d = nd
		return nil
	case string:
		nd, err := ParseDate(v)
		if err != nil {
			return err
		}
		*d = nd
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Date", source)
	}
}

func (d Date) Equal(dt Date) bool {
	return d.Year() == dt.Year() && d.Month() == dt.Month() && d.Day() == dt.Day()
}

func (d Date) Before(dt Date) bool { return d.Time.Before(dt.Time) }
func (d Date) After(dt Date) bool  { return d.Time.After(dt.Time) }

func (d Date) Compare(dt Date) int {
	if d.Before(dt) {
		return -1
	}
	if d.After(dt) {
		return 1
	}
	return 0

}

func (d Date) DaysBetween(dt Date) int {
	diff := math.Abs(float64(d.Unix() - dt.Unix()))
	return int(diff / 86400)
}

func (d Date) AddDate(year, month, days int) Date {
	dt := d.Time.AddDate(year, month, days)
	return DateFromTime(dt)
}
