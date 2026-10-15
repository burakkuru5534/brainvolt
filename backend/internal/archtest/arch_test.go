package archtest

import (
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	modulePath = "github.com/burakkuru5534/brainvolt/backend"
	// Relative to this package directory (backend/internal/archtest).
	moduleRoot = "../.."
	docPath    = "../../../docs/architecture/backend-modules.md"
)

// TestImportRules loads every package of the backend module (tests included)
// and fails on any import edge that backend-modules.md does not allow.
func TestImportRules(t *testing.T) {
	f, err := os.Open(docPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rules, err := ParseRules(f)
	if err != nil {
		t.Fatalf("%s: %v", docPath, err)
	}

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Dir:   moduleRoot,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, modulePath+"/...")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("packages failed to load")
	}

	seen := map[Import]bool{}
	var imports []Import
	for _, p := range pkgs {
		from, ok := relative(p.PkgPath)
		if !ok {
			continue
		}
		if m, isModule := moduleOf(from); isModule && !slices.Contains(rules.Modules, m) {
			t.Errorf("%s: %q is not a module of the backend-modules.md §1 catalogue", from, m)
		}
		for path := range p.Imports {
			to, ok := relative(path)
			if !ok {
				continue // standard library or third-party
			}
			imp := Import{From: from, To: to}
			if from != to && !seen[imp] {
				seen[imp] = true
				imports = append(imports, imp)
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no internal import edges found; is the package loader pointed at the backend module?")
	}

	for _, v := range rules.Check(imports) {
		t.Error(v)
	}
}

// relative strips the module path; ok is false for packages outside the
// backend module and for generated test-main packages ("….test").
func relative(pkgPath string) (string, bool) {
	rest, ok := strings.CutPrefix(pkgPath, modulePath+"/")
	if !ok || strings.HasSuffix(rest, ".test") {
		return "", false
	}
	return rest, true
}
