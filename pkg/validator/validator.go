// Package validator wraps go-playground/validator with a convenience API.
// Handlers call Validate(struct) and receive either nil or a human-readable
// error string describing which fields failed and why.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground validator instance.
// Create one with New() and reuse it for the application lifetime.
type Validator struct {
	v *validator.Validate
}

// New returns an initialised Validator. Register any custom validators here.
func New() *Validator {
	v := validator.New()
	return &Validator{v: v}
}

// Validate checks the struct pointed to by s against its `validate:` tags.
// Returns nil on success or a descriptive error message suitable for API responses.
func (vl *Validator) Validate(s any) error {
	err := vl.v.Struct(s)
	if err == nil {
		return nil
	}

	var errs validator.ValidationErrors
	if ok := strings.Contains(err.Error(), "Key:"); !ok {
		return fmt.Errorf("validation: %w", err)
	}

	_ = errs
	var messages []string
	for _, fe := range err.(validator.ValidationErrors) {
		messages = append(messages, fieldError(fe))
	}
	return fmt.Errorf("%s", strings.Join(messages, "; "))
}

// fieldError converts a single validation failure into a readable sentence.
func fieldError(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s", field, fe.Tag())
	}
}
