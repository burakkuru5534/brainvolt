// Package domain holds content's rules (docs/domain/schema-content.md):
// question shape and text limits (§2.1), source requirement (D-006), the
// verification status machine (§4.1) and duplicate fingerprints.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// Locales content is written in (D-001, D-002).
var Locales = []string{"tr", "en"}

// ValidLocale reports whether l is a content locale.
func ValidLocale(l string) bool { return slices.Contains(Locales, l) }

// QuestionType is multiple_choice or true_false.
type QuestionType string

// Question types.
const (
	MultipleChoice QuestionType = "multiple_choice"
	TrueFalse      QuestionType = "true_false"
)

// OptionCount is the number of options of the type (§2.1).
func (t QuestionType) OptionCount() int {
	if t == TrueFalse {
		return 2
	}
	return 4
}

// Valid reports whether t is a known type.
func (t QuestionType) Valid() bool { return t == MultipleChoice || t == TrueFalse }

// TrueFalseOptions are the fixed option texts of true/false questions:
// position 1 = true, 2 = false (§2.1).
var TrueFalseOptions = map[string][2]string{
	"tr": {"Doğru", "Yanlış"},
	"en": {"True", "False"},
}

// Status is the verification status of a question or a translation.
type Status string

// Statuses (§4.1).
const (
	StatusDraft    Status = "draft"
	StatusInReview Status = "in_review"
	StatusVerified Status = "verified"
	StatusRejected Status = "rejected"
)

// Origin says how a translation was written.
type Origin string

// Origins.
const (
	OriginOriginal           Origin = "original"
	OriginHumanTranslation   Origin = "human_translation"
	OriginMachineTranslation Origin = "machine_translation"
)

// Valid reports whether o is known.
func (o Origin) Valid() bool {
	return o == OriginOriginal || o == OriginHumanTranslation || o == OriginMachineTranslation
}

// Text limits (schema-content §2, limits.md).
const (
	PromptMin      = 5
	PromptMax      = 300
	OptionMax      = 120
	ExplanationMax = 500
	DifficultyMin  = 1
	DifficultyMax  = 5
)

// Text is a question's wording in one locale.
type Text struct {
	Prompt      string
	Explanation string
	// Options in canonical order; for true/false they are filled in from
	// TrueFalseOptions when empty.
	Options []string
	Origin  Origin
}

// Draft is a question as created or imported.
type Draft struct {
	SubcategoryID  uuid.UUID
	Type           QuestionType
	Difficulty     int
	OriginalLocale string
	SourceID       *uuid.UUID
	SourceRef      string
	// Correct is the 1-based position of the correct option.
	Correct int
	Texts   map[string]Text
}

// Normalize trims text and fills in the fixed true/false options.
func (d *Draft) Normalize() {
	for l, t := range d.Texts {
		t.Prompt = strings.TrimSpace(t.Prompt)
		t.Explanation = strings.TrimSpace(t.Explanation)
		if d.Type == TrueFalse && len(t.Options) == 0 {
			if fixed, ok := TrueFalseOptions[l]; ok {
				t.Options = fixed[:]
			}
		}
		opts := make([]string, len(t.Options))
		for i, o := range t.Options {
			opts[i] = strings.TrimSpace(o)
		}
		t.Options = opts
		d.Texts[l] = t
	}
}

// Validate applies the §2.1 rules; field errors use the JSON names of the
// REST body. A missing source is QUESTION_SOURCE_REQUIRED (D-006).
func (d Draft) Validate() error {
	if d.SourceID == nil {
		return apierror.New(apierror.QuestionSourceRequired)
	}
	var f apierror.Fields
	if d.SubcategoryID == uuid.Nil {
		f.Add("subcategory_id", "required")
	}
	if !d.Type.Valid() {
		f.Add("question_type", "enum", "allowed", []QuestionType{MultipleChoice, TrueFalse})
	}
	if d.Difficulty < DifficultyMin || d.Difficulty > DifficultyMax {
		f.Add("difficulty", "range", "min", DifficultyMin, "max", DifficultyMax)
	}
	if !ValidLocale(d.OriginalLocale) {
		f.Add("original_locale", "enum", "allowed", Locales)
	} else if _, ok := d.Texts[d.OriginalLocale]; !ok {
		f.Add("translations."+d.OriginalLocale, "required")
	}
	n := d.Type.OptionCount()
	if d.Correct < 1 || d.Correct > n {
		f.Add("correct_position", "range", "min", 1, "max", n)
	}
	for _, l := range sortedKeys(d.Texts) {
		validateText(&f, "translations."+l, l, d.Texts[l], n)
	}
	return f.Err()
}

// ValidateText checks one translation of a question with n options.
func ValidateText(locale string, t Text, n int) error {
	var f apierror.Fields
	validateText(&f, "translations."+locale, locale, t, n)
	return f.Err()
}

func validateText(f *apierror.Fields, path, locale string, t Text, n int) {
	if !ValidLocale(locale) {
		f.Add(path, "enum", "allowed", Locales)
		return
	}
	if c := utf8.RuneCountInString(t.Prompt); c < PromptMin || c > PromptMax {
		f.Add(path+".prompt", "length", "min", PromptMin, "max", PromptMax)
	}
	if utf8.RuneCountInString(t.Explanation) > ExplanationMax {
		f.Add(path+".explanation", "length", "max", ExplanationMax)
	}
	if !t.Origin.Valid() {
		f.Add(path+".origin", "enum", "allowed", []Origin{OriginOriginal, OriginHumanTranslation, OriginMachineTranslation})
	}
	if len(t.Options) != n {
		f.Add(path+".options", "count", "count", n)
		return
	}
	seen := map[string]bool{}
	for i, o := range t.Options {
		if c := utf8.RuneCountInString(o); c < 1 || c > OptionMax {
			f.Add(path+".options."+strconv.Itoa(i), "length", "min", 1, "max", OptionMax)
		}
		k := normalize(locale, o)
		if seen[k] {
			f.Add(path+".options."+strconv.Itoa(i), "duplicate")
		}
		seen[k] = true
	}
}

// Fingerprint identifies a question by its original-locale prompt and its
// option set (order-free), so the same question is not imported twice.
func Fingerprint(locale string, t Text) string {
	opts := make([]string, len(t.Options))
	for i, o := range t.Options {
		opts[i] = normalize(locale, o)
	}
	slices.Sort(opts)
	h := sha256.New()
	h.Write([]byte(locale + "\x1f" + normalize(locale, t.Prompt)))
	for _, o := range opts {
		h.Write([]byte("\x1f" + o))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// normalize lowercases with the locale's rules, drops punctuation and
// collapses whitespace.
func normalize(locale, s string) string {
	if locale == "tr" {
		s = strings.ToLowerSpecial(unicode.TurkishCase, s)
	} else {
		s = strings.ToLower(s)
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Review decisions.
const (
	DecisionApprove = "approve"
	DecisionReject  = "reject"
)

// CanSubmit reports whether a question in s may be submitted for review.
func CanSubmit(s Status) bool { return s == StatusDraft }

// CanReview reports whether a question or translation in s may be approved
// or rejected.
func CanReview(s Status) bool { return s == StatusInReview }

// AfterEdit is the status a question or translation moves to when its
// content changes (§4.1): verified goes back to review, rejected back to
// draft, the others stay.
func AfterEdit(s Status) Status {
	switch s {
	case StatusVerified:
		return StatusInReview
	case StatusRejected:
		return StatusDraft
	default:
		return s
	}
}

// StateError is QUESTION_STATE_INVALID with the current status.
func StateError(s Status) *apierror.Error {
	return apierror.New(apierror.QuestionStateInvalid).With("status", s)
}
