package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, buf.String())
	}
	return m
}

func TestJSONIncludesContextIDs(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Format: FormatJSON, Level: slog.LevelInfo})

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = WithUserID(ctx, "user-1")
	ctx = WithMatchID(ctx, "match-1")
	log.InfoContext(ctx, "answer accepted", "round", 3)

	got := decode(t, &buf)
	want := map[string]any{
		"msg": "answer accepted", "level": "INFO", "round": float64(3),
		"request_id": "req-1", "user_id": "user-1", "match_id": "match-1",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestIDsSurviveWithAndWithGroup(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Format: FormatJSON}).With("module", "game").WithGroup("detail")

	log.InfoContext(WithRequestID(context.Background(), "req-2"), "x", "k", "v")

	got := decode(t, &buf)
	if got["module"] != "game" {
		t.Errorf("module = %v, want game", got["module"])
	}
	detail, _ := got["detail"].(map[string]any)
	if detail["request_id"] != "req-2" || detail["k"] != "v" {
		t.Errorf("detail group = %v, want request_id and k inside it", detail)
	}
}

func TestNoContextIDsWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, Options{Format: FormatJSON}).Info("plain")
	got := decode(t, &buf)
	for _, k := range []string{RequestIDAttr, UserIDAttr, MatchIDAttr} {
		if _, ok := got[k]; ok {
			t.Errorf("unexpected %s in %v", k, got)
		}
	}
}

func TestTextFormatAndLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, Options{Format: FormatText, Level: slog.LevelWarn})

	log.Info("hidden")
	log.WarnContext(WithMatchID(context.Background(), "m-9"), "shown")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("info line written at warn level:\n%s", out)
	}
	if !strings.Contains(out, "level=WARN msg=shown match_id=m-9") {
		t.Errorf("unexpected text output:\n%s", out)
	}
}

func TestAccessorsOnEmptyContext(t *testing.T) {
	ctx := context.Background()
	if RequestID(ctx) != "" || UserID(ctx) != "" || MatchID(ctx) != "" {
		t.Fatal("ids on an empty context must be empty")
	}
	//nolint:staticcheck // SA1012: nil context must be tolerated by the accessors
	if RequestID(nil) != "" {
		t.Fatal("nil context must give empty id")
	}
}

func TestFormatUnmarshalText(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Format
	}{{"json", FormatJSON}, {" TEXT ", FormatText}} {
		var f Format
		if err := f.UnmarshalText([]byte(tc.in)); err != nil || f != tc.want {
			t.Errorf("UnmarshalText(%q) = %q, %v; want %q", tc.in, f, err, tc.want)
		}
	}
	var f Format
	if err := f.UnmarshalText([]byte("xml")); err == nil {
		t.Error("UnmarshalText(xml) succeeded, want error")
	}
}
