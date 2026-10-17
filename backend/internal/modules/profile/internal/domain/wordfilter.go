package domain

import (
	"slices"
	"strings"
	"unicode"
)

// The Phase 1 word filter (schema-identity §3): a small TR/EN list of
// profanity plus names reserved for staff and the brand. Moderation
// (Phase 8) replaces it with a managed list; until then it blocks the
// obvious cases without false positives on ordinary names.

// blockedTokens must match a whole word after normalisation (short words
// that also occur inside innocent words, e.g. "sik" in "musiki").
var blockedTokens = []string{
	"amk", "aq", "sik", "sikik", "siktir", "piç", "pic", "ibne", "göt", "yarrak", "amcık", "amcik", "kahpe",
	"fuck", "shit", "bitch", "cunt", "dick", "cock", "nigger", "nigga", "fag", "faggot", "whore", "slut",
}

// blockedSubstrings are long enough to be blocked anywhere in the text.
var blockedSubstrings = []string{
	"orospu", "siktir", "yarrak", "amina", "ananisik", "pezevenk", "gavat",
	"fuck", "motherf", "nigger", "faggot",
}

// reserved names may not be used to impersonate staff or the brand.
var reserved = []string{
	"brainvolt", "admin", "administrator", "moderator", "mod", "support", "destek",
	"yonetici", "system", "sistem", "official", "resmi", "staff", "root",
}

// leet maps look-alike digits and symbols to letters before matching.
var leet = strings.NewReplacer(
	"0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "7", "t", "@", "a", "$", "s",
)

// turkishFold maps Turkish letters to ASCII. It is applied only for the long
// substrings: folding whole words would turn ordinary words into blocked
// ones (e.g. "şık" → "sik").
var turkishFold = strings.NewReplacer("ı", "i", "i̇", "i", "ş", "s", "ğ", "g", "ü", "u", "ö", "o", "ç", "c")

// normalize lower-cases (Turkish-aware), maps look-alikes and splits into
// words; joined is the ASCII-folded text with every separator removed.
func normalize(s string) (words []string, joined string) {
	s = leet.Replace(strings.ToLowerSpecial(unicode.TurkishCase, s))
	words = strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
	return words, turkishFold.Replace(strings.Join(words, ""))
}

// Rejected reports whether text contains blocked or reserved words.
func Rejected(text string) bool {
	words, joined := normalize(text)
	for _, w := range words {
		if slices.Contains(blockedTokens, w) || slices.Contains(reserved, turkishFold.Replace(w)) {
			return true
		}
	}
	for _, b := range blockedSubstrings {
		if strings.Contains(joined, b) {
			return true
		}
	}
	return slices.Contains(reserved, joined)
}
