package seedwork

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PersistedDomainEvent struct {
	Id            uuid.UUID
	AggregateId   uuid.UUID
	AggregateType string
	EventType     string
	EventPayload  string
	CreatedAt     time.Time
}

// Common errors

var ErrAuthorization = AuthorizationError{}
var ErrValidation = ValidationError{}
var ErrApplication = ApplicationError{}
var ErrEntityConflict = EntityConflictError{}
var ErrEntityNotFound = EntityNotFoundError{}

type AuthorizationError struct {
	Role ScrumRole
}

func (e AuthorizationError) Error() string {
	return fmt.Sprintf("Role required: %d", e.Role) // TODO: Format as string role.
}

type ValidationError struct {
	Field   string
	Message string
}

type ValidationErrors struct {
	Errors []ValidationError
}

func NewValidationErrors() ValidationErrors {
	return ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

func (e *ValidationErrors) Set(field string, err error) {
	if err != nil {
		e.Errors = append(e.Errors, ValidationError{
			Field:   field,
			Message: err.Error(),
		})
	}
}

func (e ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Errors)
}

type ApplicationError struct {
	Message string
}

func (e ApplicationError) Error() string {
	return e.Message
}

type EntityConflictError struct {
	Entity string
	Id     uuid.UUID
}

func (e EntityConflictError) Error() string {
	return fmt.Sprintf("entity conflict for type %s with id %s", e.Entity, e.Id.String())
}

type EntityNotFoundError struct {
	Entity string
	Id     uuid.UUID
}

func (e EntityNotFoundError) Error() string {
	return fmt.Sprintf("entity %s with id %s not found", e.Entity, e.Id.String())
}

// Ports and adapters

type Clock interface {
	UtcNow() time.Time
}

type ScrumRole int

const (
	ScrumRoleMember ScrumRole = 1 << iota
	ScrumRoleAdmin
)

type ScrumIdentity interface {
	IsInRole(role ScrumRole) bool
}

type ScrumIdentityAnonymous struct{}

func (i ScrumIdentityAnonymous) IsInRole(role ScrumRole) bool {
	return false
}

type ScrumIdentityAuthenticated struct {
	UserId string
	Roles  []ScrumRole
}

func (i ScrumIdentityAuthenticated) IsInRole(role ScrumRole) bool {
	for _, r := range i.Roles {
		if r == role {
			return true
		}
	}
	return false
}
