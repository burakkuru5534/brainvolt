// Package i18n negotiates the response locale (D-001, D-002,
// docs/api/conventions.md §4): Turkish and English are supported; a request
// without a usable Accept-Language falls back to the user's profile locale
// (applied by the identity layer once the user is known) and then to Turkish.
package i18n

import (
	"context"
	"strconv"
	"strings"
)

// Locale is a supported language.
type Locale string

const (
	// TR is Turkish, the initial content language (D-001).
	TR Locale = "tr"
	// EN is English (D-002).
	EN Locale = "en"
	// Default is used when nothing else decides.
	Default = TR
)

// Supported lists every locale in preference order for ties.
var Supported = []Locale{TR, EN}

// Parse accepts "tr", "en" and regional variants such as "en-GB" or "tr_TR".
func Parse(tag string) (Locale, bool) {
	primary, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	primary, _, _ = strings.Cut(primary, "_")
	for _, l := range Supported {
		if primary == string(l) {
			return l, true
		}
	}
	return "", false
}

// Negotiate picks the supported locale with the highest q-value from an
// Accept-Language header (RFC 9110 §12.5.4). ok is false when the header
// names no supported locale, so the caller can apply its own fallback.
func Negotiate(header string) (best Locale, ok bool) {
	bestQ := 0.0
	for part := range strings.SplitSeq(header, ",") {
		tag, params, _ := strings.Cut(part, ";")
		l, supported := Parse(tag)
		if !supported {
			continue
		}
		q := 1.0
		if v, found := strings.CutPrefix(strings.TrimSpace(params), "q="); found {
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil || parsed < 0 || parsed > 1 {
				continue
			}
			q = parsed
		}
		if q > bestQ {
			best, bestQ, ok = l, q, true
		}
	}
	return best, ok
}

type ctxKey struct{}

// WithLocale returns a copy of ctx carrying l.
func WithLocale(ctx context.Context, l Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the locale stored in ctx, or Default.
func FromContext(ctx context.Context) Locale {
	if l, ok := ctx.Value(ctxKey{}).(Locale); ok {
		return l
	}
	return Default
}
