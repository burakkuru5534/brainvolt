package contentfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// OpenTDB is the raw snapshot format: the API's own JSON, fetched with
// encode=url3986 so no HTML entities need guessing (schema-content §6.1).
type OpenTDB struct {
	FetchedAt time.Time         `json:"fetched_at"`
	Source    string            `json:"source"`
	Results   []OpenTDBQuestion `json:"results"`
}

// OpenTDBQuestion is one question as the API returns it (URL-encoded).
type OpenTDBQuestion struct {
	Type             string   `json:"type"` // multiple | boolean
	Difficulty       string   `json:"difficulty"`
	Category         string   `json:"category"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

// decoded is a raw question with its text decoded.
type decoded struct {
	Ref        string
	Type       string
	Difficulty string
	Category   string
	Question   string
	Correct    string
	Incorrect  []string
}

func (q OpenTDBQuestion) decode() (decoded, error) {
	un := url.QueryUnescape
	d := decoded{Type: q.Type, Difficulty: q.Difficulty}
	var err error
	if d.Category, err = un(q.Category); err != nil {
		return d, err
	}
	if d.Question, err = un(q.Question); err != nil {
		return d, err
	}
	if d.Correct, err = un(q.CorrectAnswer); err != nil {
		return d, err
	}
	for _, s := range q.IncorrectAnswers {
		v, err := un(s)
		if err != nil {
			return d, err
		}
		d.Incorrect = append(d.Incorrect, v)
	}
	d.Ref = OpenTDBRef(d.Question, d.Correct)
	return d, nil
}

// OpenTDBRef is the stable reference of an OpenTDB question: a hash of its
// English text and correct answer (OpenTDB has no public ids).
func OpenTDBRef(question, correct string) string {
	sum := sha256.Sum256([]byte(question + "\x1f" + correct))
	return "otdb-" + hex.EncodeToString(sum[:6])
}

// difficulty maps easy/medium/hard to 2/3/4 (schema-content §6.3).
var difficulty = map[string]int{"easy": 2, "medium": 3, "hard": 4}

// selectedEntry is one question picked from the raw snapshot, with its
// sub-category and Turkish text (opentdb/selected.yaml).
type selectedEntry struct {
	Ref         string     `yaml:"ref"`
	Subcategory string     `yaml:"subcategory"`
	Difficulty  int        `yaml:"difficulty"` // optional override
	TR          *TextEntry `yaml:"tr"`
}

// loadOpenTDB joins opentdb/selected.yaml with the raw snapshot: English
// text from raw (origin original), Turkish from the selection.
func loadOpenTDB(dir string, subs map[string]bool, sources map[string]Source, probs *ValidationErrors) ([]Question, error) {
	selPath := filepath.Join(dir, "opentdb", "selected.yaml")
	if _, err := os.Stat(selPath); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	var sel struct {
		Questions []selectedEntry `yaml:"questions"`
	}
	if err := readYAML(selPath, &sel); err != nil {
		return nil, err
	}
	if len(sel.Questions) == 0 {
		return nil, nil
	}
	raw, err := LoadOpenTDBRaw(filepath.Join(dir, "opentdb", "raw"))
	if err != nil {
		return nil, err
	}
	var out []Question
	for _, s := range sel.Questions {
		r, ok := raw[s.Ref]
		if !ok {
			probs.addf("opentdb/selected.yaml: %s is not in opentdb/raw", s.Ref)
			continue
		}
		e := QuestionEntry{Ref: s.Ref, Subcategory: s.Subcategory, Locale: "en", Difficulty: s.Difficulty, TR: s.TR}
		if e.Difficulty == 0 {
			e.Difficulty = difficulty[r.Difficulty]
		}
		switch r.Type {
		case "boolean":
			e.Type = "true_false"
			answer := r.Correct == "True"
			e.Answer = &answer
			e.EN = &TextEntry{Prompt: r.Question}
		case "multiple":
			e.EN = &TextEntry{Prompt: r.Question, Options: append([]string{r.Correct}, r.Incorrect...)}
		default:
			probs.addf("opentdb/selected.yaml: %s: unknown OpenTDB type %q", s.Ref, r.Type)
			continue
		}
		if q, ok := buildQuestion("opentdb/selected.yaml", "opentdb", e, subs, sources, probs); ok {
			out = append(out, q)
		}
	}
	return out, nil
}

// ListOpenTDB prints the raw questions that opentdb/selected.yaml does not
// pick yet, one per line: ref, type, difficulty, category, question,
// answer — the input for choosing and translating questions.
func ListOpenTDB(dir string, w io.Writer) error {
	raw, err := LoadOpenTDBRaw(filepath.Join(dir, "opentdb", "raw"))
	if err != nil {
		return err
	}
	picked := map[string]bool{}
	selPath := filepath.Join(dir, "opentdb", "selected.yaml")
	if _, err := os.Stat(selPath); err == nil {
		var sel struct {
			Questions []selectedEntry `yaml:"questions"`
		}
		if err := readYAML(selPath, &sel); err != nil {
			return err
		}
		for _, s := range sel.Questions {
			picked[s.Ref] = true
		}
	}
	refs := make([]string, 0, len(raw))
	for ref := range raw {
		if !picked[ref] {
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		a, b := raw[refs[i]], raw[refs[j]]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return refs[i] < refs[j]
	})
	for _, ref := range refs {
		q := raw[ref]
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t→ %s\n", ref, q.Type, q.Difficulty, q.Category, q.Question, q.Correct); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, "%d not selected of %d\n", len(refs), len(raw))
	return err
}

// LoadOpenTDBRaw reads every snapshot in dir, keyed by ref.
func LoadOpenTDBRaw(dir string) (map[string]decoded, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	out := map[string]decoded{}
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // G304: snapshot files under content/opentdb/raw
		if err != nil {
			return nil, err
		}
		var snap OpenTDB
		if err := json.Unmarshal(data, &snap); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		for _, q := range snap.Results {
			d, err := q.decode()
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f, err)
			}
			out[d.Ref] = d
		}
	}
	return out, nil
}

// fetchInterval keeps to OpenTDB's limit of one request per 5 seconds.
var fetchInterval = 5200 * time.Millisecond

// FetchOpenTDB downloads the whole OpenTDB question bank with a session
// token (no repeats) and writes it unchanged to outFile. OpenTDB allows one
// request per 5 seconds, so a full fetch takes several minutes.
func FetchOpenTDB(ctx context.Context, client *http.Client, base, outFile string, logf func(string, ...any)) (int, error) {
	get := func(u string, v any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
			return fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(body, v)
	}
	wait := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(fetchInterval):
			return nil
		}
	}
	var tok struct {
		ResponseCode int    `json:"response_code"`
		Token        string `json:"token"`
	}
	if err := get(base+"/api_token.php?command=request", &tok); err != nil {
		return 0, err
	}
	if tok.Token == "" {
		return 0, fmt.Errorf("opentdb: no session token (response_code %d)", tok.ResponseCode)
	}
	snap := OpenTDB{FetchedAt: time.Now().UTC(), Source: base}
	amount := 50
	for {
		if err := wait(); err != nil {
			return 0, err
		}
		var page struct {
			ResponseCode int               `json:"response_code"`
			Results      []OpenTDBQuestion `json:"results"`
		}
		if err := get(fmt.Sprintf("%s/api.php?amount=%d&encode=url3986&token=%s", base, amount, url.QueryEscape(tok.Token)), &page); err != nil {
			return 0, err
		}
		switch page.ResponseCode {
		case 0:
			snap.Results = append(snap.Results, page.Results...)
			logf("fetched %d questions", len(snap.Results))
		case 5: // rate limited: wait and retry
			continue
		case 1: // fewer than `amount` left: ask for less
			if amount > 1 {
				amount = max(1, amount/5)
				continue
			}
			fallthrough
		case 4: // token exhausted: everything fetched
			data, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return 0, err
			}
			if err := os.MkdirAll(filepath.Dir(outFile), 0o750); err != nil {
				return 0, err
			}
			return len(snap.Results), os.WriteFile(outFile, append(data, '\n'), 0o600)
		default:
			return 0, fmt.Errorf("opentdb: response_code %d", page.ResponseCode)
		}
	}
}
