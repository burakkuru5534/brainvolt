package archtest

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const sampleDoc = "# Backend modules\n\n" +
	"## 1. Module catalogue\n\n" +
	"| # | Module | Owns | Owns data | First phase |\n" +
	"|---|--------|------|-----------|-------------|\n" +
	"| 1 | `identity` | users | users | 1 |\n" +
	"| 2 | `profile` | profiles | profiles | 1 |\n" +
	"| 3 | `ranking` | ratings | ratings | 7 |\n" +
	"| 4 | `matchmaking` | queues | queues | 5 |\n" +
	"| 5 | `progression` | xp | xp | 7 |\n\n" +
	"## 2. Allowed dependencies (compile-time)\n\n" +
	"| Module | May import |\n" +
	"|--------|-----------|\n" +
	"| identity | — |\n" +
	"| profile | identity |\n" +
	"| ranking | identity, profile |\n" +
	"| matchmaking | identity, profile, ranking* |\n" +
	"| progression | identity, ranking (seasons, for seasonal quests) |\n\n" +
	"**Rules**\n\n" +
	"## 3. Runtime communication\n\n" +
	"| Module | May import |\n" +
	"| ignored | identity |\n"

func mustParse(t *testing.T, doc string) Rules {
	t.Helper()
	r, err := ParseRules(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ParseRules: %v", err)
	}
	return r
}

func TestParseRules(t *testing.T) {
	got := mustParse(t, sampleDoc)
	want := Rules{
		Modules: []string{"identity", "profile", "ranking", "matchmaking", "progression"},
		Allowed: map[string][]string{
			"identity":    nil,
			"profile":     {"identity"},
			"ranking":     {"identity", "profile"},
			"matchmaking": {"identity", "profile"}, // ranking* is interface-wired, not an import
			"progression": {"identity", "ranking"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("ParseRules mismatch (-want +got):\n%s", diff)
	}
}

func TestParseRulesRejectsInconsistentDoc(t *testing.T) {
	tests := map[string]struct{ from, to string }{
		"catalogue module missing from table": {"| profile | identity |\n", ""},
		"table module missing from catalogue": {"| identity | — |\n", "| identity | — |\n| ghost | identity |\n"},
		"unknown dependency":                  {"| profile | identity |", "| profile | identity, ghost |"},
		"self dependency":                     {"| profile | identity |", "| profile | identity, profile |"},
		"duplicate row":                       {"| identity | — |\n", "| identity | — |\n| identity | — |\n"},
		"unparsable dependency":               {"| profile | identity |", "| profile | identity; ranking |"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := strings.Replace(sampleDoc, tc.from, tc.to, 1)
			if doc == sampleDoc {
				t.Fatal("test case did not change the document")
			}
			if _, err := ParseRules(strings.NewReader(doc)); err == nil {
				t.Fatal("ParseRules accepted an inconsistent document")
			}
		})
	}
}

func TestCheck(t *testing.T) {
	rules := mustParse(t, sampleDoc)
	const m = "internal/modules/"

	allowed := []Import{
		{m + "profile", m + "identity"},
		{m + "profile/internal/service", m + "identity"},
		{m + "profile/internal/service", m + "profile/internal/domain"},
		{m + "profile", "internal/platform/clock"},
		{m + "progression/internal/service", m + "ranking"},
		{"internal/platform/httpserver", "internal/platform/logging"},
		{"internal/app", m + "ranking"},
		{"internal/app", "internal/platform/config"},
		{"cmd/api", "internal/app"},
		{"cmd/api", m + "identity/internal/domain"},
	}
	if v := rules.Check(allowed); len(v) != 0 {
		t.Fatalf("allowed imports reported as violations:\n%s", strings.Join(v, "\n"))
	}

	forbidden := map[string]Import{
		"edge not in table":            {m + "identity", m + "profile"},
		"upward read marked with *":    {m + "matchmaking", m + "ranking"},
		"non-public package":           {m + "ranking", m + "profile/internal/domain"},
		"public sub-package":           {m + "ranking", m + "profile/events"},
		"platform imports module":      {"internal/platform/httpserver", m + "identity"},
		"platform imports app":         {"internal/platform/config", "internal/app"},
		"module imports app":           {m + "profile", "internal/app"},
		"module imports other package": {m + "profile", "internal/archtest"},
		"anything imports cmd":         {"internal/app", "cmd/api"},
		"uncatalogued source module":   {m + "ghost", "internal/platform/clock"},
		"uncatalogued target module":   {"internal/app", m + "ghost"},
	}
	for name, imp := range forbidden {
		t.Run(name, func(t *testing.T) {
			if v := rules.Check([]Import{imp}); len(v) != 1 {
				t.Fatalf("Check(%v) = %q, want exactly one violation", imp, v)
			}
		})
	}
}

// TestDocumentRules makes sure the real document parses and matches the
// §1 catalogue, so a malformed edit of backend-modules.md fails the build.
func TestDocumentRules(t *testing.T) {
	f, err := os.Open(docPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rules, err := ParseRules(f)
	if err != nil {
		t.Fatalf("%s: %v", docPath, err)
	}
	if got := len(rules.Modules); got != 17 {
		t.Errorf("catalogue has %d modules, want 17 (update this test together with the document)", got)
	}
	if !slices.Equal(rules.Allowed["game"], []string{"identity", "profile", "content", "realtime", "team", "commerce"}) {
		t.Errorf("game may import %v, does not match backend-modules.md §2", rules.Allowed["game"])
	}
	if slices.Contains(rules.Allowed["matchmaking"], "ranking") {
		t.Error("matchmaking → ranking must stay interface-wired, not a package import")
	}
}
