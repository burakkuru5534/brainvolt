// Package domain holds profile's pure rules (docs/domain/schema-identity.md
// §3): generated names for new users, field rules, the 30-day change limits
// and the basic word filter of Phase 1. No I/O.
package domain

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Avatars are the built-in preset avatar ids (D-021: no uploads at launch).
// The app ships one image per id; new ids are added, never renamed.
var Avatars = []string{
	"fox", "owl", "cat", "panda", "lion", "penguin",
	"rabbit", "bear", "koala", "tiger", "wolf", "dolphin",
}

// ChangeCooldown is how often username and country may change.
const ChangeCooldown = 30 * 24 * time.Hour

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_.]{3,20}$`)
	countryPattern  = regexp.MustCompile(`^[A-Z]{2}$`)
)

// randomInt returns a uniform number in [0, n).
func randomInt(n int64) int64 {
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err) // the OS random source failed; nothing sensible can continue
	}
	return v.Int64()
}

// GeneratedUsername returns "oyuncu" + digits; attempt widens the number
// range after collisions (5, then 7, then 9 digits).
func GeneratedUsername(attempt int) string {
	digits := int64(5)
	switch {
	case attempt >= 6:
		digits = 9
	case attempt >= 3:
		digits = 7
	}
	low := int64(1)
	for range digits - 1 {
		low *= 10
	}
	return fmt.Sprintf("oyuncu%d", low+randomInt(9*low))
}

// GeneratedDisplayName returns "Misafir 4821" / "Guest 4821".
func GeneratedDisplayName(locale string) string {
	n := 1000 + randomInt(9000)
	if locale == "en" {
		return fmt.Sprintf("Guest %d", n)
	}
	return fmt.Sprintf("Misafir %d", n)
}

// RandomAvatar picks a preset.
func RandomAvatar() string { return Avatars[randomInt(int64(len(Avatars)))] }

// ValidAvatar reports whether id is a preset.
func ValidAvatar(id string) bool { return slices.Contains(Avatars, id) }

// NormalizeUsername lower-cases and trims a requested username.
func NormalizeUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// ValidUsername checks the schema rule ^[a-z0-9_.]{3,20}$.
func ValidUsername(s string) bool { return usernamePattern.MatchString(s) }

// NormalizeCountry upper-cases and trims a country code.
func NormalizeCountry(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// ValidCountry checks the ISO 3166-1 alpha-2 shape.
func ValidCountry(s string) bool { return countryPattern.MatchString(s) }

// NormalizeDisplayName trims and collapses inner whitespace.
func NormalizeDisplayName(s string) string { return strings.Join(strings.Fields(s), " ") }

// ValidDisplayName checks 2–24 characters without control characters.
func ValidDisplayName(s string) bool {
	n := utf8.RuneCountInString(s)
	if n < 2 || n > 24 {
		return false
	}
	return !strings.ContainsFunc(s, unicode.IsControl)
}

// NextChange returns when a field changed at last may change again;
// nil means now (never changed by the user).
func NextChange(last *time.Time) *time.Time {
	if last == nil {
		return nil
	}
	t := last.Add(ChangeCooldown)
	return &t
}

// CanChange reports whether a field changed at last may change at now.
func CanChange(last *time.Time, now time.Time) bool {
	next := NextChange(last)
	return next == nil || !now.Before(*next)
}

// Locales are the supported profile locales (D-002).
var Locales = []string{"tr", "en"}

// ChallengesFrom are the allowed values of settings.challenges_from.
var ChallengesFrom = []string{"everyone", "friends", "nobody"}
