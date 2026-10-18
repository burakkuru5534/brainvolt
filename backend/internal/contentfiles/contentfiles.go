// Package contentfiles reads the reviewable question files under content/
// (repo-layout.md, content/README.md): taxonomy.yaml, sources.yaml, the
// BrainVolt team pool in brainvolt/*.yaml and the OpenTDB selection in
// opentdb/selected.yaml with its raw snapshot in opentdb/raw/*.json. It
// checks everything that can be checked without a database, so CI rejects
// a broken file before anyone imports it (D-006: no question without a
// source and license).
package contentfiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// LocalizedText is a {tr: …, en: …} map.
type LocalizedText map[string]string

// Subcategory is one entry of taxonomy.yaml.
type Subcategory struct {
	Slug        string        `yaml:"slug"`
	Name        LocalizedText `yaml:"name"`
	Description LocalizedText `yaml:"description"`
}

// Category is one entry of taxonomy.yaml.
type Category struct {
	Slug          string        `yaml:"slug"`
	Icon          string        `yaml:"icon"`
	Name          LocalizedText `yaml:"name"`
	Description   LocalizedText `yaml:"description"`
	Subcategories []Subcategory `yaml:"subcategories"`
}

// Source is one entry of sources.yaml.
type Source struct {
	Key         string    `yaml:"key"`
	ID          uuid.UUID `yaml:"id"`
	Kind        string    `yaml:"kind"`
	Name        string    `yaml:"name"`
	URL         string    `yaml:"url"`
	License     string    `yaml:"license"`
	LicenseURL  string    `yaml:"license_url"`
	Attribution string    `yaml:"attribution"`
	ShareAlike  bool      `yaml:"share_alike"`
	Notes       string    `yaml:"notes"`
}

// TextEntry is a question's wording in one locale. For multiple choice the
// first option is the correct one; the server shuffles options per match.
type TextEntry struct {
	Prompt      string   `yaml:"prompt"`
	Options     []string `yaml:"options"`
	Explanation string   `yaml:"explanation"`
	Origin      string   `yaml:"origin"`
}

// QuestionEntry is one question of a pool file.
type QuestionEntry struct {
	Ref         string `yaml:"ref"`
	Subcategory string `yaml:"subcategory"` // "category/subcategory"
	Type        string `yaml:"type"`        // default multiple_choice
	Difficulty  int    `yaml:"difficulty"`
	Locale      string `yaml:"locale"` // original language, default tr
	// Answer is the correct answer of a true/false question.
	Answer *bool      `yaml:"answer"`
	TR     *TextEntry `yaml:"tr"`
	EN     *TextEntry `yaml:"en"`
}

// Question is a validated question ready for import.
type Question struct {
	File        string
	Ref         string
	SourceKey   string
	Category    string
	Subcategory string
	Draft       content.Draft
}

// Bundle is everything under content/.
type Bundle struct {
	Categories []Category
	Sources    []Source
	Questions  []Question
}

// ValidationErrors collects every error found while loading, so one run reports
// all of them.
type ValidationErrors []string

func (p ValidationErrors) Error() string { return strings.Join(p, "\n") }

func (p *ValidationErrors) addf(format string, args ...any) {
	*p = append(*p, fmt.Sprintf(format, args...))
}

// Load reads and validates dir (the repository's content/ directory).
func Load(dir string) (*Bundle, error) {
	var probs ValidationErrors
	b := &Bundle{}
	var tax struct {
		Categories []Category `yaml:"categories"`
	}
	if err := readYAML(filepath.Join(dir, "taxonomy.yaml"), &tax); err != nil {
		return nil, err
	}
	b.Categories = tax.Categories
	var src struct {
		Sources []Source `yaml:"sources"`
	}
	if err := readYAML(filepath.Join(dir, "sources.yaml"), &src); err != nil {
		return nil, err
	}
	b.Sources = src.Sources

	subs := checkTaxonomy(b.Categories, &probs)
	sources := checkSources(b.Sources, &probs)

	pool, err := filepath.Glob(filepath.Join(dir, "brainvolt", "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(pool)
	for _, f := range pool {
		var file struct {
			Source    string          `yaml:"source"`
			Questions []QuestionEntry `yaml:"questions"`
		}
		if err := readYAML(f, &file); err != nil {
			probs.addf("%v", err)
			continue
		}
		rel, _ := filepath.Rel(dir, f)
		for _, e := range file.Questions {
			if q, ok := buildQuestion(rel, file.Source, e, subs, sources, &probs); ok {
				b.Questions = append(b.Questions, q)
			}
		}
	}

	otdb, err := loadOpenTDB(dir, subs, sources, &probs)
	if err != nil {
		return nil, err
	}
	b.Questions = append(b.Questions, otdb...)

	checkDuplicates(b.Questions, &probs)
	if len(probs) > 0 {
		return b, probs
	}
	return b, nil
}

func readYAML(path string, v any) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: paths come from the content directory given on the command line
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) { // empty file = no entries
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// describe turns a validation error into "field: rule" text.
func describe(err error) string {
	e := apierror.From(err)
	if e == nil {
		return err.Error()
	}
	fields, ok := e.Details["fields"].([]apierror.FieldError)
	if !ok || len(fields) == 0 {
		return string(e.Code)
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		p := f.Field + ": " + f.Rule
		if len(f.Params) > 0 {
			p += fmt.Sprintf(" %v", f.Params)
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, "; ")
}

// checkTaxonomy returns the set of "category/subcategory" keys.
func checkTaxonomy(cats []Category, probs *ValidationErrors) map[string]bool {
	subs := map[string]bool{}
	seenCat := map[string]bool{}
	for _, c := range cats {
		if seenCat[c.Slug] {
			probs.addf("taxonomy.yaml: duplicate category %q", c.Slug)
		}
		seenCat[c.Slug] = true
		checkSlug("taxonomy.yaml: category", c.Slug, probs)
		checkNames("taxonomy.yaml: category "+c.Slug, c.Name, probs)
		if c.Icon == "" {
			probs.addf("taxonomy.yaml: category %s: icon is required", c.Slug)
		}
		if len(c.Subcategories) == 0 {
			probs.addf("taxonomy.yaml: category %s has no sub-categories", c.Slug)
		}
		for _, s := range c.Subcategories {
			key := c.Slug + "/" + s.Slug
			if subs[key] {
				probs.addf("taxonomy.yaml: duplicate sub-category %q", key)
			}
			subs[key] = true
			checkSlug("taxonomy.yaml: sub-category", s.Slug, probs)
			checkNames("taxonomy.yaml: sub-category "+key, s.Name, probs)
		}
	}
	return subs
}

func checkSlug(what, slug string, probs *ValidationErrors) {
	if len(slug) < 2 || len(slug) > 40 || strings.Trim(slug, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		probs.addf("%s %q: slug must be 2–40 of a-z, 0-9, -", what, slug)
	}
}

func checkNames(what string, names LocalizedText, probs *ValidationErrors) {
	for _, l := range content.Locales {
		if n := []rune(names[l]); len(n) < 2 || len(n) > 40 {
			probs.addf("%s: %s name must be 2–40 characters", what, l)
		}
	}
}

func checkSources(srcs []Source, probs *ValidationErrors) map[string]Source {
	out := map[string]Source{}
	ids := map[uuid.UUID]bool{}
	for i := range srcs {
		s := &srcs[i]
		if s.Key == "" || out[s.Key].Key != "" {
			probs.addf("sources.yaml: missing or duplicate key %q", s.Key)
		}
		if s.ID == uuid.Nil || ids[s.ID] {
			probs.addf("sources.yaml: %s: missing or duplicate id", s.Key)
		}
		ids[s.ID] = true
		if !slices.Contains([]string{"opentdb", "brainvolt_team", "creator", "partner"}, s.Kind) {
			probs.addf("sources.yaml: %s: unknown kind %q", s.Key, s.Kind)
		}
		if s.Name == "" || s.License == "" || s.Attribution == "" {
			probs.addf("sources.yaml: %s: name, license and attribution are required (D-006)", s.Key)
		}
		out[s.Key] = *s
	}
	return out
}

// SourceSpecs converts the sources for content.UpsertSources.
func (b *Bundle) SourceSpecs() []content.SourceSpec {
	out := make([]content.SourceSpec, 0, len(b.Sources))
	for i := range b.Sources {
		s := &b.Sources[i]
		out = append(out, content.SourceSpec{
			ID: s.ID, Kind: s.Kind, Name: s.Name, URL: s.URL, LicenseCode: s.License,
			LicenseURL: s.LicenseURL, AttributionText: s.Attribution, ShareAlike: s.ShareAlike, Notes: s.Notes,
		})
	}
	return out
}

// CategorySpecs converts the taxonomy for content.UpsertTaxonomy.
func (b *Bundle) CategorySpecs() []content.CategorySpec {
	names := func(n, d LocalizedText) map[string]content.Name {
		out := map[string]content.Name{}
		for _, l := range content.Locales {
			out[l] = content.Name{Name: n[l], Description: d[l]}
		}
		return out
	}
	out := make([]content.CategorySpec, 0, len(b.Categories))
	for _, c := range b.Categories {
		spec := content.CategorySpec{Slug: c.Slug, IconKey: c.Icon, Names: names(c.Name, c.Description)}
		for _, s := range c.Subcategories {
			spec.Subcategories = append(spec.Subcategories, content.SubcategorySpec{Slug: s.Slug, Names: names(s.Name, s.Description)})
		}
		out = append(out, spec)
	}
	return out
}

// placeholderSubcategory stands in for the database id during offline
// validation.
var placeholderSubcategory = uuid.MustParse("00000000-0000-7000-8000-000000000001")

func buildQuestion(file, sourceKey string, e QuestionEntry, subs map[string]bool, sources map[string]Source, probs *ValidationErrors) (Question, bool) {
	where := file + ": " + e.Ref
	src, ok := sources[sourceKey]
	if !ok {
		probs.addf("%s: source %q is not in sources.yaml (D-006)", where, sourceKey)
		return Question{}, false
	}
	if e.Ref == "" {
		probs.addf("%s: a question without ref", file)
		return Question{}, false
	}
	if !subs[e.Subcategory] {
		probs.addf("%s: unknown sub-category %q", where, e.Subcategory)
		return Question{}, false
	}
	cat, sub, _ := strings.Cut(e.Subcategory, "/")
	if e.Type == "" {
		e.Type = string(content.MultipleChoice)
	}
	if e.Locale == "" {
		e.Locale = "tr"
	}
	d := content.Draft{
		SubcategoryID: placeholderSubcategory, Type: content.QuestionType(e.Type), Difficulty: e.Difficulty,
		OriginalLocale: e.Locale, SourceID: &src.ID, SourceRef: e.Ref, Texts: map[string]content.Text{},
	}
	switch d.Type {
	case content.TrueFalse:
		if e.Answer == nil {
			probs.addf("%s: true_false needs answer: true|false", where)
			return Question{}, false
		}
		d.Correct = 2
		if *e.Answer {
			d.Correct = 1
		}
	default:
		d.Correct = 1 // the first option is the correct one
	}
	for l, t := range map[string]*TextEntry{"tr": e.TR, "en": e.EN} {
		if t == nil {
			continue
		}
		origin := t.Origin
		if origin == "" {
			origin = string(content.OriginHumanTranslation)
			if l == e.Locale {
				origin = string(content.OriginOriginal)
			}
		}
		d.Texts[l] = content.Text{Prompt: t.Prompt, Options: t.Options, Explanation: t.Explanation, Origin: content.Origin(origin)}
	}
	d.Normalize()
	if err := d.Validate(); err != nil {
		probs.addf("%s: %s", where, describe(err))
		return Question{}, false
	}
	return Question{File: file, Ref: e.Ref, SourceKey: sourceKey, Category: cat, Subcategory: sub, Draft: d}, true
}

func checkDuplicates(qs []Question, probs *ValidationErrors) {
	refs := map[string]string{}
	prints := map[string]string{}
	for i := range qs {
		q := &qs[i]
		key := q.SourceKey + ":" + q.Ref
		if other, ok := refs[key]; ok {
			probs.addf("%s: ref %s already used in %s", q.File, q.Ref, other)
		}
		refs[key] = q.File
		fp := content.Fingerprint(q.Draft.OriginalLocale, q.Draft.Texts[q.Draft.OriginalLocale])
		if other, ok := prints[fp]; ok {
			probs.addf("%s: %s duplicates %s", q.File, q.Ref, other)
		}
		prints[fp] = q.Ref
	}
}
