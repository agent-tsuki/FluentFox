// Package validator wraps go-playground/validator with a structured error API.
// Handlers call Validate(struct) and receive either nil or a *ValidationError
// containing per-field details suitable for JSON API responses.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError is returned by Validate when one or more fields fail.
// It implements the error interface so it can flow through HandleError.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		msgs[i] = f.Field + ": " + f.Message
	}
	return strings.Join(msgs, "; ")
}

// Validator wraps the go-playground validator instance.
// Create one with New() and reuse it for the application lifetime.
type Validator struct {
	v *validator.Validate
}

// New returns an initialised Validator. Register any custom validators here.
func New() *Validator {
	return &Validator{v: validator.New()}
}

// Validate checks the struct pointed to by s against its `validate:` tags.
// Returns nil on success, *ValidationError on field failures, or a plain error
// for unexpected internal failures.
func (vl *Validator) Validate(s any) error {
	err := vl.v.Struct(s)
	if err == nil {
		return nil
	}

	valErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("validation: %w", err)
	}

	fields := make([]FieldError, 0, len(valErrs))
	for _, fe := range valErrs {
		fields = append(fields, FieldError{
			Field:   strings.ToLower(fe.Field()),
			Message: fieldMessage(fe),
		})
	}
	return &ValidationError{Fields: fields}
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}
