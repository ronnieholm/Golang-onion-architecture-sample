package validation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func UuidNotEmpty(id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("should be non-empty")
	}
	return nil
}

func StringNotNilOrWhitespace(s string) error {
	if len(strings.TrimSpace(s)) == 0 {
		return fmt.Errorf("should be non-nil, non-empty or non-whitespace")
	}
	return nil
}

func StringMaxLength(length int, s string) error {
	if len(s) > length {
		return fmt.Errorf("should contain less than or equal to %d characters", length)
	}
	return nil
}

func IntBetween(from, to, value int) error {
	if value < from && value > to {
		return fmt.Errorf("should be between %d and %d, both inclusive", from, to)
	}
	return nil
}
