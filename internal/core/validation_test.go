package core

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateUUidNotZero(t *testing.T) {
	tests := []struct {
		uuid     string
		expected int
	}{
		{"00000000-0000-0000-0000-000000000001", 0},
		{"00000000-0000-0000-0000-000000000000", 1},
	}

	for _, tt := range tests {
		t.Run(tt.uuid, func(t *testing.T) {
			e := &ValidationError{make(map[string][]string)}
			ValidateUUIDNotZero("field", uuid.MustParse(tt.uuid), e)
			require.Equal(t, tt.expected, len(e.FieldValues))
			if len(e.FieldValues) == 1 {
				v := e.FieldValues["field"]
				require.Equal(t, 1, len(v))
				require.Contains(t, e.Error(), tt.uuid)
			}
		})
	}
}

func TestValidateStringCurrencyCode(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{"DKK", 0},
		{"ABC", 1},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			e := &ValidationError{make(map[string][]string)}
			ValidateStringCurrencyCode("field", tt.code, e)
			require.Equal(t, tt.expected, len(e.FieldValues))
			if len(e.FieldValues) == 1 {
				v := e.FieldValues["field"]
				require.Equal(t, 1, len(v))
				require.Contains(t, e.Error(), tt.code)
			}
		})
	}
}

func TestValidateFloat64InclusiveRange(t *testing.T) {
	tests := map[string]struct {
		value    float64
		min      float64
		max      float64
		expected int
	}{
		"below":        {1.5, 2.0, 3.0, 1},
		"below equal":  {2.0, 2.0, 3.0, 0},
		"within":       {2.5, 2.0, 3.0, 0},
		"higher equal": {3.0, 2.0, 3.0, 0},
		"higher":       {3.5, 2.0, 3.0, 1},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			e := &ValidationError{make(map[string][]string)}
			ValidateFloat64InclusiveRange("field", tt.value, tt.min, tt.max, e)
			require.Equal(t, tt.expected, len(e.FieldValues))
			if len(e.FieldValues) == 1 {
				v := e.FieldValues["field"]
				require.Equal(t, 1, len(v))
				require.Contains(t, e.Error(), fmt.Sprintf("%g", tt.value))
				require.Contains(t, e.Error(), fmt.Sprintf("%g", tt.min))
				require.Contains(t, e.Error(), fmt.Sprintf("%g", tt.max))
			}
		})
	}
}

func TestValidateFloat64DecimalPlaces(t *testing.T) {
	tests := map[string]struct {
		value    float64
		min      int
		max      int
		places   int
		expected int
	}{
		"no decimals":  {1, 0, 0, 0, 0},
		"one decimal":  {1.1, 0, 1, 1, 0},
		"two decimals": {1.12, 0, 1, 2, 1},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			e := &ValidationError{make(map[string][]string)}
			ValidateFloat64DecimalPlaces("field", tt.value, tt.min, tt.max, e)
			require.Equal(t, tt.expected, len(e.FieldValues))
			if len(e.FieldValues) == 1 {
				v := e.FieldValues["field"]
				require.Equal(t, 1, len(v))
				require.Contains(t, e.Error(), fmt.Sprintf("%f", tt.value))
				require.Contains(t, e.Error(), fmt.Sprintf("%d", tt.min))
				require.Contains(t, e.Error(), fmt.Sprintf("%d", tt.max))
				require.Contains(t, e.Error(), fmt.Sprintf("%d", tt.places))
			}
		})
	}
}

func TestValidateDateIncludeRange(t *testing.T) {
	tests := map[string]struct {
		value    Date
		min      Date
		max      Date
		expected int
	}{
		"below":        {NewDate(2026, 6, 30), NewDate(2026, 7, 1), NewDate(2026, 7, 31), 1},
		"below equal":  {NewDate(2026, 7, 1), NewDate(2026, 7, 1), NewDate(2026, 7, 31), 0},
		"within":       {NewDate(2026, 7, 15), NewDate(2026, 7, 1), NewDate(2026, 7, 31), 0},
		"higher equal": {NewDate(2026, 7, 31), NewDate(2026, 7, 1), NewDate(2026, 7, 31), 0},
		"higher":       {NewDate(2026, 8, 1), NewDate(2026, 7, 1), NewDate(2026, 7, 31), 1},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			e := &ValidationError{make(map[string][]string)}
			ValidateDateInclusiveRange("field", tt.value, tt.min, tt.max, e)
			require.Equal(t, tt.expected, len(e.FieldValues))
			if len(e.FieldValues) == 1 {
				v := e.FieldValues["field"]
				require.Equal(t, 1, len(v))
				require.Contains(t, e.Error(), tt.value.String())
				require.Contains(t, e.Error(), tt.min.String())
				require.Contains(t, e.Error(), tt.max.String())
			}
		})
	}
}
