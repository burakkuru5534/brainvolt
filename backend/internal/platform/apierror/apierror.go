// Package apierror holds the error-code registry shared by REST and
// WebSocket (docs/api/errors.md): stable codes, their HTTP status, TR/EN
// fallback messages, and the Error value that business code returns.
// platform/httpserver turns an Error into the REST envelope
// (conventions.md §6); platform/wsserver will turn it into sys.error.
package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
)

// Error is an error with a registry code. Cause is logged, never sent.
type Error struct {
	Code    Code
	Details map[string]any
	// RetryAfter, when set, is sent as the Retry-After header (and as
	// details.retry_after_s for RATE_LIMITED).
	RetryAfter time.Duration
	Cause      error
}

// New returns an Error with code.
func New(code Code) *Error { return &Error{Code: code} }

// Wrap returns an Error with code whose Cause is err.
func Wrap(code Code, err error) *Error { return &Error{Code: code, Cause: err} }

// With returns a copy of e with details[key] = value.
func (e *Error) With(key string, value any) *Error {
	c := *e
	c.Details = maps.Clone(e.Details)
	if c.Details == nil {
		c.Details = map[string]any{}
	}
	c.Details[key] = value
	return &c
}

// WithRetryAfter returns a copy of e with RetryAfter set.
func (e *Error) WithRetryAfter(d time.Duration) *Error {
	c := *e
	c.RetryAfter = d
	return &c
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Cause)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

// Message returns the fallback message for e in locale l.
func (e *Error) Message(l i18n.Locale) string { return Message(e.Code, l) }

// From returns the *Error inside err, or nil.
func From(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Message returns the fallback message for code in locale l, falling back to
// the default locale and then to INTERNAL's message for unknown codes.
func Message(code Code, l i18n.Locale) string {
	table := messagesTR
	if l == i18n.EN {
		table = messagesEN
	}
	if m, ok := table[code]; ok {
		return m
	}
	return table[Internal]
}

// FieldError is one entry of details.fields for VALIDATION_FAILED, e.g.
// {"field": "display_name", "rule": "length", "min": 2, "max": 24}.
type FieldError struct {
	Field string
	Rule  string
	// Params are rule parameters such as min/max, flattened into the object.
	Params map[string]any
}

// MarshalJSON flattens Params next to field and rule.
func (f FieldError) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, len(f.Params)+2)
	maps.Copy(m, f.Params)
	m["field"] = f.Field
	m["rule"] = f.Rule
	return json.Marshal(m)
}

// Validation returns VALIDATION_FAILED with details.fields.
func Validation(fields ...FieldError) *Error {
	return New(ValidationFailed).With("fields", fields)
}
