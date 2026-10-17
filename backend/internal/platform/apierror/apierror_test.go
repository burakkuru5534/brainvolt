package apierror

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
)

var codeInCell = regexp.MustCompile("`([A-Z_]+)`")

// parseRegistry reads the code → HTTP status table from errors.md.
// Combined rows such as "`ROOM_FULL` / `ROOM_CLOSED` | 409 / 410" map
// pairwise; a single status applies to every code of the row; "—" means
// WebSocket-only.
func parseRegistry(r io.Reader) (map[Code]int, error) {
	out := map[Code]int{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 2 {
			return nil, fmt.Errorf("short row %q", line)
		}
		var codes []Code
		for _, m := range codeInCell.FindAllStringSubmatch(cells[0], -1) {
			codes = append(codes, Code(m[1]))
		}
		var statuses []int
		for s := range strings.SplitSeq(cells[1], "/") {
			s = strings.TrimSpace(s)
			if s == "—" {
				statuses = append(statuses, wsOnly)
				continue
			}
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("row %q: status %q", line, s)
			}
			statuses = append(statuses, n)
		}
		for i, c := range codes {
			switch {
			case len(statuses) == len(codes):
				out[c] = statuses[i]
			case len(statuses) == 1:
				out[c] = statuses[0]
			default:
				return nil, fmt.Errorf("row %q: %d codes but %d statuses", line, len(codes), len(statuses))
			}
		}
	}
	return out, sc.Err()
}

// TestRegistryMatchesDocument keeps codes.go identical to errors.md: same
// codes, same HTTP statuses.
func TestRegistryMatchesDocument(t *testing.T) {
	f, err := os.Open("../../../../docs/api/errors.md")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := parseRegistry(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc) < 60 {
		t.Fatalf("parsed only %d codes from errors.md; parser out of date?", len(doc))
	}
	if diff := cmp.Diff(doc, statuses); diff != "" {
		t.Fatalf("codes.go differs from errors.md (-doc +code):\n%s", diff)
	}
}

func TestEveryCodeHasMessages(t *testing.T) {
	for code := range statuses {
		for name, table := range map[string]map[Code]string{"tr": messagesTR, "en": messagesEN} {
			if strings.TrimSpace(table[code]) == "" {
				t.Errorf("%s: missing %s message", code, name)
			}
		}
	}
	for name, table := range map[string]map[Code]string{"tr": messagesTR, "en": messagesEN} {
		for code := range table {
			if !code.Known() {
				t.Errorf("%s message for unregistered code %s", name, code)
			}
		}
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := map[Code]int{
		AccountUpgradeRequired: http.StatusForbidden,
		InviteCodeExpired:      http.StatusGone,
		AnswerLate:             http.StatusInternalServerError, // WebSocket-only
		Code("NO_SUCH_CODE"):   http.StatusInternalServerError,
	}
	for c, want := range tests {
		if got := c.HTTPStatus(); got != want {
			t.Errorf("%s.HTTPStatus() = %d, want %d", c, got, want)
		}
	}
}

func TestMessage(t *testing.T) {
	if got := Message(AccountUpgradeRequired, i18n.TR); got != "Bu işlem için hesabını yükseltmelisin." {
		t.Errorf("TR message = %q", got)
	}
	if got := Message(AccountUpgradeRequired, i18n.EN); got != "Upgrade your account to do this." {
		t.Errorf("EN message = %q", got)
	}
	if got := Message(Code("NO_SUCH_CODE"), i18n.EN); got != messagesEN[Internal] {
		t.Errorf("unknown code message = %q", got)
	}
}

func TestErrorWrappingAndCopies(t *testing.T) {
	cause := errors.New("db down")
	base := Wrap(Unavailable, cause)
	withDetail := base.With("component", "postgres").WithRetryAfter(2 * time.Second)

	if base.Details != nil || base.RetryAfter != 0 {
		t.Fatal("With/WithRetryAfter modified the original error")
	}
	if withDetail.Details["component"] != "postgres" || withDetail.RetryAfter != 2*time.Second {
		t.Fatalf("copy = %+v", withDetail)
	}
	if !errors.Is(withDetail, cause) {
		t.Fatal("cause not reachable through errors.Is")
	}
	if got := withDetail.Error(); got != "UNAVAILABLE: db down" {
		t.Fatalf("Error() = %q", got)
	}
	if From(fmt.Errorf("handler: %w", withDetail)) != withDetail {
		t.Fatal("From did not find the wrapped *Error")
	}
	if From(cause) != nil {
		t.Fatal("From found an *Error in a plain error")
	}
	if New(NotFound).Error() != "NOT_FOUND" {
		t.Fatal("Error() without cause")
	}
}

func TestValidationDetails(t *testing.T) {
	e := Validation(FieldError{Field: "display_name", Rule: "length", Params: map[string]any{"min": 2, "max": 24}})
	b, err := json.Marshal(e.Details)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"fields":[{"field":"display_name","max":24,"min":2,"rule":"length"}]}`
	if string(b) != want {
		t.Fatalf("details = %s, want %s", b, want)
	}
	if e.Code.HTTPStatus() != http.StatusBadRequest {
		t.Fatal("VALIDATION_FAILED must be 400")
	}
}

func TestFields(t *testing.T) {
	var f Fields
	if f.Err() != nil {
		t.Fatal("empty Fields must give nil")
	}
	f.Add("locale", "enum", "allowed", []string{"tr", "en"})
	f.Add("app_version", "required")
	e := From(f.Err())
	if e == nil || e.Code != ValidationFailed {
		t.Fatalf("Err = %v", f.Err())
	}
	b, _ := json.Marshal(e.Details)
	want := `{"fields":[{"allowed":["tr","en"],"field":"locale","rule":"enum"},{"field":"app_version","rule":"required"}]}`
	if string(b) != want {
		t.Fatalf("details = %s", b)
	}
}
