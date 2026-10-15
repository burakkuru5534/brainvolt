// Package ids generates and parses entity ids: UUIDv7 strings, never
// integers (docs/api/conventions.md, ADR-0002). UUIDv7 is time-ordered, so
// new rows land at the end of Postgres b-tree indexes and ids sort by
// creation time.
package ids

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ID is a UUID; the zero value is the nil UUID.
type ID = uuid.UUID

// ErrInvalid is returned for anything that is not a canonical UUID string.
var ErrInvalid = errors.New("invalid id")

// New returns a new UUIDv7. Ids created by one process are strictly
// increasing, also within the same millisecond.
func New() ID {
	// NewV7 only fails if the OS random source fails, which leaves the process
	// unable to do anything safely; panicking is the documented uuid.Must behaviour.
	return uuid.Must(uuid.NewV7())
}

// NewString returns New().String().
func NewString() string { return New().String() }

// Parse accepts only the canonical 36-character lowercase or uppercase form
// ("0192c7e0-…"); the urn:, braced and hyphen-less forms that uuid.Parse also
// accepts are rejected so every id has exactly one spelling on the wire.
func Parse(s string) (ID, error) {
	if len(s) != 36 {
		return uuid.Nil, ErrInvalid
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}

// ParseV7 is Parse restricted to version-7 ids, for ids the server issued.
func ParseV7(s string) (ID, error) {
	id, err := Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	if id.Version() != 7 {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}

// CreatedAt returns the millisecond timestamp embedded in a UUIDv7.
// It reports false for other versions.
func CreatedAt(id ID) (time.Time, bool) {
	if id.Version() != 7 {
		return time.Time{}, false
	}
	ms := int64(id[0])<<40 | int64(id[1])<<32 | int64(id[2])<<24 |
		int64(id[3])<<16 | int64(id[4])<<8 | int64(id[5])
	return time.UnixMilli(ms).UTC(), true
}
