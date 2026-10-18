package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

func validDraft() Draft {
	src := uuid.New()
	return Draft{
		SubcategoryID: uuid.New(), Type: MultipleChoice, Difficulty: 2, OriginalLocale: "tr",
		SourceID: &src, Correct: 1,
		Texts: map[string]Text{
			"tr": {Prompt: "  Türkiye'nin başkenti neresidir? ", Options: []string{"Ankara", "İstanbul", "İzmir", "Bursa"}, Origin: OriginOriginal},
			"en": {Prompt: "What is the capital of Turkey?", Options: []string{"Ankara", "Istanbul", "Izmir", "Bursa"}, Origin: OriginHumanTranslation},
		},
	}
}

func fields(t *testing.T, err error) []string {
	t.Helper()
	e := apierror.From(err)
	if e == nil || e.Code != apierror.ValidationFailed {
		t.Fatalf("want VALIDATION_FAILED, got %v", err)
	}
	var out []string
	for _, f := range e.Details["fields"].([]apierror.FieldError) {
		out = append(out, f.Field+":"+f.Rule)
	}
	return out
}

func TestValidate(t *testing.T) {
	d := validDraft()
	d.Normalize()
	if err := d.Validate(); err != nil {
		t.Fatalf("valid draft: %v", err)
	}
	if d.Texts["tr"].Prompt != "Türkiye'nin başkenti neresidir?" {
		t.Fatalf("prompt not trimmed: %q", d.Texts["tr"].Prompt)
	}

	noSource := validDraft()
	noSource.SourceID = nil
	if e := apierror.From(noSource.Validate()); e == nil || e.Code != apierror.QuestionSourceRequired {
		t.Fatalf("missing source: %v", e)
	}

	bad := validDraft()
	bad.Difficulty = 9
	bad.Correct = 5
	bad.OriginalLocale = "de"
	bad.Texts["en"] = Text{Prompt: "Hi?", Options: []string{"a", "A", "b"}, Origin: "robot"}
	got := strings.Join(fields(t, bad.Validate()), " ")
	for _, w := range []string{"difficulty:range", "correct_position:range", "original_locale:enum",
		"translations.en.prompt:length", "translations.en.origin:enum", "translations.en.options:count"} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %s in %s", w, got)
		}
	}

	dup := validDraft()
	dup.Texts["tr"] = Text{Prompt: "Hangisi bir renktir?", Options: []string{"Mavi", "mavi.", "Taş", "Su"}, Origin: OriginOriginal}
	if got := fields(t, dup.Validate()); len(got) != 1 || got[0] != "translations.tr.options.1:duplicate" {
		t.Fatalf("duplicate options: %v", got)
	}

	noOriginal := validDraft()
	delete(noOriginal.Texts, "tr")
	if got := fields(t, noOriginal.Validate()); len(got) != 1 || got[0] != "translations.tr:required" {
		t.Fatalf("missing original text: %v", got)
	}
}

func TestTrueFalseOptionsAreFilledIn(t *testing.T) {
	src := uuid.New()
	d := Draft{
		SubcategoryID: uuid.New(), Type: TrueFalse, Difficulty: 1, OriginalLocale: "tr", SourceID: &src, Correct: 2,
		Texts: map[string]Text{
			"tr": {Prompt: "Örümcekler böcektir.", Origin: OriginOriginal},
			"en": {Prompt: "Spiders are insects.", Origin: OriginHumanTranslation},
		},
	}
	d.Normalize()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := d.Texts["tr"].Options; len(got) != 2 || got[0] != "Doğru" || got[1] != "Yanlış" {
		t.Fatalf("tr options = %v", got)
	}
	if got := d.Texts["en"].Options; got[0] != "True" || got[1] != "False" {
		t.Fatalf("en options = %v", got)
	}
	d.Correct = 3
	if got := fields(t, d.Validate()); got[0] != "correct_position:range" {
		t.Fatalf("true/false has 2 positions: %v", got)
	}
}

func TestFingerprint(t *testing.T) {
	a := Text{Prompt: "İstanbul hangi yıl fethedildi?", Options: []string{"1453", "1071", "1299", "1517"}}
	b := Text{Prompt: "  istanbul HANGİ yıl   fethedildi ", Options: []string{"1517", "1299", "1071", "1453"}}
	if Fingerprint("tr", a) != Fingerprint("tr", b) {
		t.Fatal("case, punctuation, spacing and option order must not matter")
	}
	c := a
	c.Options = []string{"1453", "1071", "1299", "1518"}
	if Fingerprint("tr", a) == Fingerprint("tr", c) {
		t.Fatal("a different option set is a different question")
	}
	if Fingerprint("tr", a) == Fingerprint("en", a) {
		t.Fatal("locale is part of the fingerprint")
	}
}

func TestStatusRules(t *testing.T) {
	if !CanSubmit(StatusDraft) || CanSubmit(StatusInReview) || CanSubmit(StatusVerified) {
		t.Fatal("only drafts are submitted")
	}
	if !CanReview(StatusInReview) || CanReview(StatusDraft) || CanReview(StatusVerified) {
		t.Fatal("only items in review are reviewed")
	}
	for from, to := range map[Status]Status{
		StatusVerified: StatusInReview, StatusRejected: StatusDraft,
		StatusDraft: StatusDraft, StatusInReview: StatusInReview,
	} {
		if got := AfterEdit(from); got != to {
			t.Errorf("AfterEdit(%s) = %s, want %s", from, got, to)
		}
	}
	if e := StateError(StatusDraft); e.Code != apierror.QuestionStateInvalid || e.Details["status"] != StatusDraft {
		t.Fatalf("StateError = %+v", e)
	}
}
