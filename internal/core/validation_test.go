package core

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestValidateUUidNotZero(t *testing.T) {
	tests := []struct {
		uuid     string
		expected bool
	}{
		{"00000000-0000-0000-0000-000000000001", false},
		{"00000000-0000-0000-0000-000000000000", true},
	}

	for _, tt := range tests {
		t.Run(tt.uuid, func(t *testing.T) {
			err := ValidateUUIDNotZero(uuid.MustParse(tt.uuid))
			if tt.expected {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.uuid)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateStringCurrencyCode(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"DKK", false},
		{"ABC", true},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := ValidateStringCurrencyCode(tt.code)
			if tt.expected {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.code)
			}
		})
	}
}

func TestValidateFloat64InclusiveRange(t *testing.T) {
	tests := map[string]struct {
		value    float64
		min      float64
		max      float64
		expected bool
	}{
		"below":        {1.5, 2.0, 3.0, true},
		"below equal":  {2.0, 2.0, 3.0, false},
		"within":       {2.5, 2.0, 3.0, false},
		"higher equal": {3.0, 2.0, 3.0, false},
		"higher":       {3.5, 2.0, 3.0, true},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			err := ValidateFloat64InclusiveRange(tt.value, tt.min, tt.max)
			if tt.expected {
				require.Error(t, err)
				require.Contains(t, err.Error(), fmt.Sprintf("%g", tt.value))
				require.Contains(t, err.Error(), fmt.Sprintf("%g", tt.min))
				require.Contains(t, err.Error(), fmt.Sprintf("%g", tt.max))
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
		expected bool
	}{
		"no decimals":  {1, 0, 0, 0, false},
		"one decimal":  {1.1, 0, 1, 1, false},
		"two decimals": {1.12, 0, 1, 2, true},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			err := ValidateFloat64DecimalPlaces(tt.value, tt.min, tt.max)
			if tt.expected {
				require.Error(t, err)
				require.Contains(t, err.Error(), fmt.Sprintf("%f", tt.value))
				require.Contains(t, err.Error(), fmt.Sprintf("%d", tt.min))
				require.Contains(t, err.Error(), fmt.Sprintf("%d", tt.max))
				require.Contains(t, err.Error(), fmt.Sprintf("%d", tt.places))
			}
		})
	}
}

func TestValidateDateIncludeRange(t *testing.T) {
	tests := map[string]struct {
		value    Date
		min      Date
		max      Date
		expected bool
	}{
		"below":        {NewDate(2026, 6, 30), NewDate(2026, 7, 1), NewDate(2026, 7, 31), true},
		"below equal":  {NewDate(2026, 7, 1), NewDate(2026, 7, 1), NewDate(2026, 7, 31), false},
		"within":       {NewDate(2026, 7, 15), NewDate(2026, 7, 1), NewDate(2026, 7, 31), false},
		"higher equal": {NewDate(2026, 7, 31), NewDate(2026, 7, 1), NewDate(2026, 7, 31), false},
		"higher":       {NewDate(2026, 8, 1), NewDate(2026, 7, 1), NewDate(2026, 7, 31), true},
	}

	for name, tt := range tests {
		t.Run(string(name), func(t *testing.T) {
			err := ValidateDateInclusiveRange(tt.value, tt.min, tt.max)
			if tt.expected {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.value.String())
				require.Contains(t, err.Error(), tt.min.String())
				require.Contains(t, err.Error(), tt.max.String())
			}
		})
	}
}
