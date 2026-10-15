package i18n

import (
	"context"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want Locale
	}{{"tr", TR}, {"TR", TR}, {"tr-TR", TR}, {"tr_TR", TR}, {" en-GB ", EN}, {"en", EN}}
	for _, tc := range tests {
		if got, ok := Parse(tc.in); !ok || got != tc.want {
			t.Errorf("Parse(%q) = %q, %v; want %q", tc.in, got, ok, tc.want)
		}
	}
	for _, in := range []string{"", "de", "*", "english"} {
		if got, ok := Parse(in); ok {
			t.Errorf("Parse(%q) = %q, want unsupported", in, got)
		}
	}
}

func TestNegotiate(t *testing.T) {
	tests := []struct {
		header string
		want   Locale
		ok     bool
	}{
		{"tr", TR, true},
		{"en-US,en;q=0.9", EN, true},
		{"de-DE,de;q=0.9,en;q=0.8,tr;q=0.7", EN, true},
		{"tr;q=0.5, en;q=0.6", EN, true},
		{"en;q=0.5, tr", TR, true},
		{"en, tr", EN, true},          // equal q: first listed wins
		{"en;q=0, tr;q=0", "", false}, // q=0 means "not acceptable"
		{"en;q=abc, tr;q=0.1", TR, true},
		{"de, fr", "", false},
		{"", "", false},
		{"*", "", false},
	}
	for _, tc := range tests {
		got, ok := Negotiate(tc.header)
		if got != tc.want || ok != tc.ok {
			t.Errorf("Negotiate(%q) = %q, %v; want %q, %v", tc.header, got, ok, tc.want, tc.ok)
		}
	}
}

func TestContext(t *testing.T) {
	if got := FromContext(context.Background()); got != Default {
		t.Fatalf("FromContext(empty) = %q, want %q", got, Default)
	}
	if got := FromContext(WithLocale(context.Background(), EN)); got != EN {
		t.Fatalf("FromContext = %q, want en", got)
	}
}
