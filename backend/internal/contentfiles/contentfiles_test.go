package contentfiles

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
)

// TestRepositoryContent keeps the committed files under content/ valid:
// CI fails before a broken question reaches an import (D-006).
func TestRepositoryContent(t *testing.T) {
	b, err := Load(filepath.Join("..", "..", "..", "content"))
	if err != nil {
		t.Fatalf("content/ has problems:\n%v", err)
	}
	if len(b.Categories) == 0 || len(b.Sources) == 0 || len(b.Questions) == 0 {
		t.Fatalf("empty content: %d categories, %d sources, %d questions", len(b.Categories), len(b.Sources), len(b.Questions))
	}
	for _, q := range b.Questions {
		if _, ok := q.Draft.Texts["en"]; !ok {
			t.Errorf("%s: %s has no English text (D-002)", q.File, q.Ref)
		}
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const taxonomy = `categories:
  - slug: science
    icon: science
    name: {tr: Bilim, en: Science}
    subcategories:
      - slug: space
        name: {tr: Uzay, en: Space}
`

const sources = `sources:
  - key: team
    id: 01a14f8e-5782-7822-b3ce-2c32fbf52f2a
    kind: brainvolt_team
    name: BrainVolt
    license: LicenseRef-BrainVolt
    attribution: BrainVolt team
  - key: opentdb
    id: 01a14f8e-5782-791c-bccb-1369f8008e95
    kind: opentdb
    name: Open Trivia Database
    license: CC-BY-SA-4.0
    attribution: Open Trivia Database, CC BY-SA 4.0
    share_alike: true
`

func TestLoadReportsEveryProblem(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "taxonomy.yaml", taxonomy)
	write(t, dir, "sources.yaml", sources)
	write(t, dir, "brainvolt/a.yaml", `source: team
questions:
  - ref: ok-1
    subcategory: science/space
    difficulty: 1
    tr: {prompt: "Kızıl Gezegen hangisidir?", options: [Mars, Venüs, Jüpiter, Satürn]}
    en: {prompt: "Which is the Red Planet?", options: [Mars, Venus, Jupiter, Saturn]}
  - ref: ok-1
    subcategory: science/space
    difficulty: 2
    tr: {prompt: "Başka bir soru metni?", options: [a, b, c, d]}
  - ref: bad-sub
    subcategory: science/nope
    difficulty: 1
    tr: {prompt: "Bilinmeyen alt kategori?", options: [a, b, c, d]}
  - ref: bad-shape
    subcategory: science/space
    difficulty: 7
    tr: {prompt: "Üç seçenek mi?", options: [a, b, c]}
  - ref: tf-no-answer
    subcategory: science/space
    type: true_false
    difficulty: 1
    tr: {prompt: "Cevabı olmayan doğru/yanlış."}
  - ref: dup
    subcategory: science/space
    difficulty: 1
    tr: {prompt: "kızıl gezegen HANGİSİDİR", options: [Satürn, Jüpiter, Venüs, Mars]}
`)
	write(t, dir, "brainvolt/b.yaml", `source: nobody
questions:
  - ref: no-source
    subcategory: science/space
    difficulty: 1
    tr: {prompt: "Kaynağı olmayan soru?", options: [a, b, c, d]}
`)
	_, err := Load(dir)
	var probs ValidationErrors
	if !errors.As(err, &probs) {
		t.Fatalf("want Problems, got %v", err)
	}
	got := probs.Error()
	for _, want := range []string{
		"ref ok-1 already used",
		`unknown sub-category "science/nope"`,
		"bad-shape: difficulty: range",
		"translations.tr.options: count",
		"true_false needs answer",
		"dup duplicates ok-1",
		`source "nobody" is not in sources.yaml`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestLoadUnknownFieldIsAnError(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "taxonomy.yaml", taxonomy)
	write(t, dir, "sources.yaml", sources)
	write(t, dir, "brainvolt/a.yaml", "source: team\nquestions:\n  - ref: x\n    subcategory: science/space\n    correct: 2\n")
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "field correct not found") {
		t.Fatalf("a typo in a field name must fail: %v", err)
	}
}

func rawSnapshot() OpenTDB {
	e := url.QueryEscape
	return OpenTDB{Results: []OpenTDBQuestion{
		{Type: "multiple", Difficulty: "medium", Category: e("Science: Computers"), Question: e("What does \"CPU\" stand for?"),
			CorrectAnswer: e("Central Processing Unit"), IncorrectAnswers: []string{e("Computer Personal Unit"), e("Central Process Unit"), e("Central Processor Unit")}},
		{Type: "boolean", Difficulty: "easy", Category: e("Science & Nature"), Question: e("The Sun is a star."),
			CorrectAnswer: "True", IncorrectAnswers: []string{"False"}},
	}}
}

func TestOpenTDBJoin(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "taxonomy.yaml", taxonomy)
	write(t, dir, "sources.yaml", sources)
	raw, _ := json.Marshal(rawSnapshot())
	write(t, dir, "opentdb/raw/opentdb-20261018.json", string(raw))
	cpu := OpenTDBRef(`What does "CPU" stand for?`, "Central Processing Unit")
	sun := OpenTDBRef("The Sun is a star.", "True")
	write(t, dir, "opentdb/selected.yaml", `questions:
  - ref: `+cpu+`
    subcategory: science/space
    tr:
      prompt: '"CPU" neyin kısaltmasıdır?'
      options: [Merkezi İşlem Birimi, Bilgisayar Kişisel Birimi, Merkezi Süreç Birimi, Merkezi İşlemci Birimi]
  - ref: `+sun+`
    subcategory: science/space
    difficulty: 1
    tr:
      prompt: Güneş bir yıldızdır.
      origin: machine_translation
`)
	b, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Questions) != 2 {
		t.Fatalf("questions = %d", len(b.Questions))
	}
	q := b.Questions[0].Draft
	if b.Questions[0].SourceKey != "opentdb" || q.OriginalLocale != "en" || q.Difficulty != 3 || q.Correct != 1 ||
		q.Texts["en"].Prompt != `What does "CPU" stand for?` || q.Texts["en"].Origin != content.OriginOriginal ||
		q.Texts["tr"].Origin != content.OriginHumanTranslation || q.SourceRef != cpu {
		t.Fatalf("multiple = %+v", q)
	}
	tf := b.Questions[1].Draft
	if tf.Type != content.TrueFalse || tf.Correct != 1 || tf.Difficulty != 1 || tf.Texts["en"].Options[0] != "True" ||
		tf.Texts["tr"].Origin != content.OriginMachineTranslation {
		t.Fatalf("boolean = %+v", tf)
	}

	var list strings.Builder
	if err := ListOpenTDB(dir, &list); err != nil || list.String() != "0 not selected of 2\n" {
		t.Fatalf("list after selecting both = %q, %v", list.String(), err)
	}

	write(t, dir, "opentdb/selected.yaml", "questions:\n  - ref: otdb-000000000000\n    subcategory: science/space\n")
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "otdb-000000000000 is not in opentdb/raw") {
		t.Fatalf("unknown ref: %v", err)
	}
	list.Reset()
	if err := ListOpenTDB(dir, &list); err != nil || !strings.Contains(list.String(), sun+"\tboolean\teasy\tScience & Nature\tThe Sun is a star.\t→ True") ||
		!strings.HasSuffix(list.String(), "2 not selected of 2\n") {
		t.Fatalf("list = %q, %v", list.String(), err)
	}
}

func TestFetchOpenTDB(t *testing.T) {
	defer func(d time.Duration) { fetchInterval = d }(fetchInterval)
	fetchInterval = time.Millisecond
	var calls atomic.Int32
	snap := rawSnapshot()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api_token.php"):
			_, _ = w.Write([]byte(`{"response_code":0,"token":"tok"}`))
		case r.URL.Query().Get("token") != "tok" || r.URL.Query().Get("encode") != "url3986":
			http.Error(w, "bad query", http.StatusBadRequest)
		default:
			switch calls.Add(1) {
			case 1:
				_ = json.NewEncoder(w).Encode(map[string]any{"response_code": 0, "results": snap.Results})
			default:
				_, _ = w.Write([]byte(`{"response_code":4,"results":[]}`))
			}
		}
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "raw", "snap.json")
	n, err := FetchOpenTDB(t.Context(), srv.Client(), srv.URL, out, t.Logf)
	if err != nil || n != 2 {
		t.Fatalf("fetch = %d, %v", n, err)
	}
	raw, err := LoadOpenTDBRaw(filepath.Dir(out))
	if err != nil || len(raw) != 2 {
		t.Fatalf("saved snapshot = %d, %v", len(raw), err)
	}
}
