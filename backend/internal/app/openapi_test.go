package app

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "rewrite docs/api/openapi.yaml from the code")

// TestOpenAPIUpToDate is the drift check of roadmap 1.6.7: the committed
// document must equal the one generated from the routes and Go types.
// Regenerate with `make gen` (go test ./internal/app -run TestOpenAPIUpToDate -update).
func TestOpenAPIUpToDate(t *testing.T) {
	got, err := OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "..", OpenAPIPath)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (generate it with `make gen`)", err)
	}
	if !bytes.Equal(committed, got) {
		t.Fatalf("%s is out of date: run `make gen` and commit the result", OpenAPIPath)
	}
}

func TestOpenAPIContents(t *testing.T) {
	raw, err := OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		OpenAPI string                               `yaml:"openapi"`
		Paths   map[string]map[string]map[string]any `yaml:"paths"`
		Comps   struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("openapi = %q", doc.OpenAPI)
	}
	// Every Phase 1 endpoint of rest.md §1–§2 plus health.
	for _, route := range []string{
		"get /healthz", "get /readyz",
		"post /v1/session", "post /v1/session/abandon-guest",
		"get /v1/me", "post /v1/me/deletion", "delete /v1/me/deletion",
		"post /v1/me/devices", "delete /v1/me/devices/{installation_id}",
		"patch /v1/me/profile", "get /v1/me/settings", "patch /v1/me/settings",
		"get /v1/profiles/{user_id}",
	} {
		method, path, _ := strings.Cut(route, " ")
		if doc.Paths[path][method] == nil {
			t.Errorf("missing %s", route)
		}
	}
	abandon := doc.Paths["/v1/session/abandon-guest"]["post"]
	if !strings.Contains(string(mustYAML(t, abandon)), "Idempotency-Key") {
		t.Error("abandon-guest must document the required Idempotency-Key header")
	}
	if _, ok := doc.Paths["/healthz"]["get"]["security"]; ok {
		t.Error("/healthz must be public")
	}
	for _, name := range []string{"SessionRequest", "SessionResponse", "Me", "OwnProfile", "ErrorEnvelope"} {
		if doc.Comps.Schemas[name] == nil {
			t.Errorf("schema %s missing", name)
		}
	}
	tier := string(mustYAML(t, doc.Comps.Schemas["SessionResponse"]))
	if !strings.Contains(tier, "unverified") {
		t.Error("tier enum values missing from SessionResponse")
	}
}

func mustYAML(t *testing.T, v any) []byte {
	t.Helper()
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
