package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// RequestParserError collects errors from command and query requests. The
// request is a set of fields where a single field may fail multiple
// validations. For instance, a password may fail multiple of minimum length,
// upper case character, lower case character, and digit at the same time.
//
// The error is intended for human and machine consumption. If the caller is a
// web client, the error must enable the caller to correlate form fields to
// request fields to validation errors so errors can be shown next to UI
// elements.
type RequestParserError struct {
	FieldErrors map[string][]string
}

func (e *RequestParserError) Add(field, error string) {
	// Allocate the map here so allocation happens only on the error path.
	if e.FieldErrors == nil {
		e.FieldErrors = make(map[string][]string)
	}
	e.FieldErrors[field] = append(e.FieldErrors[field], error)
}

func (e *RequestParserError) HasErrors() bool {
	return len(e.FieldErrors) > 0
}

func (e *RequestParserError) Error() string {
	if !e.HasErrors() {
		return ""
	}

	var b strings.Builder
	b.WriteString("request parse errors: ")

	i := 0
	for f, v := range e.FieldErrors {
		for _, m := range v {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f)
			b.WriteString(": ")
			b.WriteString(m)
			i++
		}
	}
	return b.String()
}

func ValidateUUIDNotZero(value uuid.UUID) error {
	if value == uuid.Nil {
		return fmt.Errorf("must be non-zero, but was %s", value.String())
	}
	return nil
}

func ValidateStringCurrencyCode(value string) error {
	_, ok := CurrencyCodes[value]
	if !ok {
		return fmt.Errorf("must be one of the allowed currency codes, but was %s", value)
	}
	return nil
}

func ValidateFloat64InclusiveRange(value float64, min, max float64) error {
	// TODO(rh): bug: pass in 1.123456789 and the error becomes ""Decimal places must be between 0 and 6 inclusive, but 1.123457 has 9"
	if value < min || value > max {
		return fmt.Errorf("must be between %g and %g inclusive, but was %g", min, max, value)
	}
	return nil
}

func ValidateFloat64DecimalPlaces(value float64, min, max int) error {
	s := strconv.FormatFloat(value, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	places := 0
	if i != -1 {
		places = len(s) - i - 1
	}
	if places < min || places > max {
		return fmt.Errorf("decimal places must be between %d and %d inclusive, but %f has %d", min, max, value, places)
	}
	return nil
}

func ValidateDateInclusiveRange(value, min, max Date) error {
	if value.Before(min) || value.After(max) {
		return fmt.Errorf("must be between %s and %s inclusive, but was %s", min, max, value.String())
	}
	return nil
}
