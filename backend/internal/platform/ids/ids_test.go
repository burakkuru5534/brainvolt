package ids

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewIsV7AndStrictlyIncreasing(t *testing.T) {
	prev := New()
	for range 10_000 {
		id := New()
		if id.Version() != 7 || id.Variant() != uuid.RFC4122 {
			t.Fatalf("id %s: version %d variant %v, want v7 RFC 4122", id, id.Version(), id.Variant())
		}
		if strings.Compare(prev.String(), id.String()) >= 0 {
			t.Fatalf("ids not strictly increasing: %s then %s", prev, id)
		}
		prev = id
	}
}

func TestCreatedAt(t *testing.T) {
	before := time.Now().Truncate(time.Millisecond)
	id := New()
	after := time.Now()

	got, ok := CreatedAt(id)
	if !ok {
		t.Fatal("CreatedAt on a v7 id reported false")
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("CreatedAt = %v, want between %v and %v", got, before, after)
	}
	if _, ok := CreatedAt(uuid.New()); ok {
		t.Fatal("CreatedAt on a v4 id reported true")
	}
}

func TestParse(t *testing.T) {
	id := New()
	s := id.String()

	for _, in := range []string{s, strings.ToUpper(s)} {
		got, err := Parse(in)
		if err != nil || got != id {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, id)
		}
	}

	bad := []string{
		"",
		"not-a-uuid",
		"urn:uuid:" + s,
		"{" + s + "}",
		strings.ReplaceAll(s, "-", ""),
		s[:35] + "g",
	}
	for _, in := range bad {
		if _, err := Parse(in); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalid", in, err)
		}
	}
}

func TestParseV7(t *testing.T) {
	if _, err := ParseV7(NewString()); err != nil {
		t.Fatalf("ParseV7(v7) = %v", err)
	}
	if _, err := ParseV7(uuid.NewString()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ParseV7(v4) = %v, want ErrInvalid", err)
	}
	if _, err := ParseV7("x"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ParseV7(x) = %v, want ErrInvalid", err)
	}
}
