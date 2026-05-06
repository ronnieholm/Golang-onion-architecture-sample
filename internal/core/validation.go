package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type ValidationError struct {
	FieldValues map[string][]string
}

func (e *ValidationError) Add(key, message string) {
	// Allocate the map here so allocation happens only on the error path.
	if e.FieldValues == nil {
		e.FieldValues = make(map[string][]string)
	}
	e.FieldValues[key] = append(e.FieldValues[key], message)
}

func (e *ValidationError) HasErrors() bool {
	return len(e.FieldValues) > 0
}

func (e *ValidationError) Error() string {
	if !e.HasErrors() {
		return ""
	}

	var b strings.Builder
	b.WriteString("validation error: ")

	i := 0
	for f, v := range e.FieldValues {
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

func Validate[R validator](req R) error {
	err := &ValidationError{}
	req.validate(err)

	// Having this function return error instead of ValidationError is more
	// idiomatic. Then nil may be returned on no error, which changes a call
	// site from
	//
	// if err := Validate(req); err.HasErrors() { ... }
	//
	// to
	//
	// if err := Validate(req); err != nil { ... }
	if !err.HasErrors() {
		return nil
	}
	return err
}

func ValidateUUIDNotZero(field string, value uuid.UUID, err *ValidationError) {
	if value == uuid.Nil {
		message := fmt.Sprintf("Must be non-zero, but was %s", value.String())
		err.Add(field, message)
	}
}

func ValidateStringCurrencyCode(field string, value string, err *ValidationError) {
	_, ok := CurrencyCodes[value]
	if !ok {
		message := fmt.Sprintf("Must be one of the allowed currency codes, but was %s", value)
		err.Add(field, message)
	}
}

func ValidateFloat64InclusiveRange(field string, value float64, min, max float64, err *ValidationError) {
	if value < min || value > max {
		message := fmt.Sprintf("Must be between %g and %g inclusive, but was %g", min, max, value)
		err.Add(field, message)
	}
}

func ValidateFloat64DecimalPlaces(field string, value float64, min, max int, err *ValidationError) {
	s := strconv.FormatFloat(value, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	places := 0
	if i != -1 {
		places = len(s) - i - 1
	}
	if places < min || places > max {
		message := fmt.Sprintf("Decimal places must be between %d and %d inclusive, but %f has %d", min, max, value, places)
		err.Add(field, message)
	}
}

func ValidateDateInclusiveRange(field string, value, min, max Date, err *ValidationError) {
	if value.Before(min) || value.After(max) {
		message := fmt.Sprintf("Must be between %s and %s inclusive, but was %s", min, max, value.String())
		err.Add(field, message)
	}
}

func ValidateDiscountPercentages(authorizedField, advancedField, premiumField string, authorizedValue, advancedValue, premiumValue float64, err *ValidationError) {
	ValidateFloat64InclusiveRange(authorizedField, authorizedValue, 0, 100, err)
	ValidateFloat64InclusiveRange(advancedField, advancedValue, 0, 100, err)
	ValidateFloat64InclusiveRange(premiumField, premiumValue, 0, 100, err)

	if authorizedValue > advancedValue {
		message := fmt.Sprintf("Must be between 0 and %g inclusive, but was %g", advancedValue, authorizedValue)
		err.Add(authorizedField, message)
	}
	if advancedValue > premiumValue {
		message := fmt.Sprintf("Must be between %g and %g inclusive, but was %g", authorizedValue, premiumValue, advancedValue)
		err.Add(advancedField, message)
	}
}
