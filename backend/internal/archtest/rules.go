// Package archtest enforces the module dependency rules of
// docs/architecture/backend-modules.md (roadmap 1.3.4).
//
// The allowed-import table is read from that document at test time, so the
// document stays the single source of truth: adding an edge means editing
// the document in the same commit (backend-modules.md §2, rule 3).
package archtest

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

// Rules is the parsed content of backend-modules.md.
type Rules struct {
	// Modules lists every module of the §1 catalogue.
	Modules []string
	// Allowed maps a module to the modules whose public package it may import (§2).
	Allowed map[string][]string
}

var (
	catalogueRow  = regexp.MustCompile("^\\|\\s*\\d+\\s*\\|\\s*`([a-z]+)`\\s*\\|")
	parenthetical = regexp.MustCompile(`\([^)]*\)`)
	moduleName    = regexp.MustCompile(`^[a-z]+$`)
)

// ParseRules reads the §1 module catalogue and the §2 "May import" table.
func ParseRules(r io.Reader) (Rules, error) {
	rules := Rules{Allowed: map[string][]string{}}
	section := ""
	inImportTable := false

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "## ") {
			section = line
			inImportTable = false
			continue
		}
		switch {
		case strings.HasPrefix(section, "## 1."):
			if m := catalogueRow.FindStringSubmatch(line); m != nil {
				rules.Modules = append(rules.Modules, m[1])
			}
		case strings.HasPrefix(section, "## 2."):
			if strings.HasPrefix(line, "| Module | May import |") {
				inImportTable = true
				continue
			}
			if !inImportTable {
				continue
			}
			if !strings.HasPrefix(line, "|") {
				inImportTable = false
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) != 2 || strings.HasPrefix(strings.TrimSpace(cells[0]), "---") {
				continue
			}
			from := strings.TrimSpace(cells[0])
			deps, err := parseDeps(cells[1])
			if err != nil {
				return Rules{}, fmt.Errorf("module %q: %w", from, err)
			}
			if _, dup := rules.Allowed[from]; dup {
				return Rules{}, fmt.Errorf("module %q listed twice in the import table", from)
			}
			rules.Allowed[from] = deps
		}
	}
	if err := sc.Err(); err != nil {
		return Rules{}, err
	}
	if len(rules.Modules) == 0 {
		return Rules{}, fmt.Errorf("no modules found in the §1 catalogue")
	}
	if len(rules.Allowed) == 0 {
		return Rules{}, fmt.Errorf("no rows found in the §2 import table")
	}
	return rules, rules.validate()
}

// parseDeps turns a "May import" cell into module names.
// "—" means none; "(…)" comments are dropped; a name marked "*" is an
// interface-wired upward read (e.g. matchmaking → ranking*), which the
// document explicitly says is NOT a package import, so it is excluded.
func parseDeps(cell string) ([]string, error) {
	cell = strings.TrimSpace(parenthetical.ReplaceAllString(cell, ""))
	if cell == "—" || cell == "-" || cell == "" {
		return nil, nil
	}
	var deps []string
	for _, part := range strings.Split(cell, ",") {
		name := strings.TrimSpace(part)
		if strings.HasSuffix(name, "*") {
			continue
		}
		if !moduleName.MatchString(name) {
			return nil, fmt.Errorf("cannot parse dependency %q", part)
		}
		deps = append(deps, name)
	}
	return deps, nil
}

// validate checks that the catalogue and the import table agree.
func (r Rules) validate() error {
	for _, m := range r.Modules {
		if _, ok := r.Allowed[m]; !ok {
			return fmt.Errorf("module %q is in the §1 catalogue but missing from the §2 import table", m)
		}
	}
	for from, deps := range r.Allowed {
		if !slices.Contains(r.Modules, from) {
			return fmt.Errorf("module %q is in the §2 import table but not in the §1 catalogue", from)
		}
		for _, d := range deps {
			if !slices.Contains(r.Modules, d) {
				return fmt.Errorf("module %q may import unknown module %q", from, d)
			}
			if d == from {
				return fmt.Errorf("module %q lists itself as a dependency", from)
			}
		}
	}
	return nil
}

// Import is one import edge between two packages of the backend module,
// both given relative to the module root (e.g. "internal/modules/game").
type Import struct {
	From, To string
}

const (
	// migrationsPkg is the schema package: importable by everyone, imports
	// no internal package itself.
	migrationsPkg  = "migrations"
	platformPrefix = "internal/platform"
	modulesPrefix  = "internal/modules/"
	appPkg         = "internal/app"
)

// Check returns one message per import that breaks the rules:
//   - platform packages import only other platform packages (§2 rule 4);
//   - module packages import platform, their own module, and the public
//     package (exactly internal/modules/<m>) of modules allowed by §2;
//   - nothing below internal/ imports cmd/, and only cmd/ imports internal/app;
//   - anyone may import the schema package migrations, which itself imports
//     no internal package;
//   - every folder under internal/modules/ is a catalogued module.
func (r Rules) Check(imports []Import) []string {
	var violations []string
	add := func(imp Import, why string) {
		violations = append(violations, fmt.Sprintf("%s imports %s: %s", imp.From, imp.To, why))
	}
	for _, imp := range imports {
		if isUnder(imp.From, migrationsPkg) {
			add(imp, "the migrations package must not import internal packages")
			continue
		}
		if imp.To == migrationsPkg {
			continue
		}
		fromModule, fromIsModule := moduleOf(imp.From)
		toModule, toIsModule := moduleOf(imp.To)

		if fromIsModule && !slices.Contains(r.Modules, fromModule) {
			add(imp, fmt.Sprintf("%q is not a module of the backend-modules.md §1 catalogue", fromModule))
			continue
		}
		if toIsModule && !slices.Contains(r.Modules, toModule) {
			add(imp, fmt.Sprintf("%q is not a module of the backend-modules.md §1 catalogue", toModule))
			continue
		}

		switch {
		case isUnder(imp.To, "cmd"):
			add(imp, "cmd/ packages are entry points and are never imported")
		case isUnder(imp.To, appPkg) && !isUnder(imp.From, "cmd") && !isUnder(imp.From, appPkg):
			add(imp, "only cmd/ may import the composition root internal/app")
		case isUnder(imp.From, platformPrefix) && !isUnder(imp.To, platformPrefix):
			add(imp, "platform packages may only import other platform packages")
		case fromIsModule && toIsModule && fromModule != toModule:
			if imp.To != modulesPrefix+toModule {
				add(imp, fmt.Sprintf("only the public package internal/modules/%s may be imported", toModule))
			} else if !slices.Contains(r.Allowed[fromModule], toModule) {
				add(imp, fmt.Sprintf("edge %s → %s is not allowed by backend-modules.md §2", fromModule, toModule))
			}
		case fromIsModule && !toIsModule && !isUnder(imp.To, platformPrefix):
			add(imp, "modules may only import platform packages and allowed modules")
		}
	}
	return violations
}

// moduleOf reports the module a package belongs to, if it is under internal/modules/.
func moduleOf(pkg string) (string, bool) {
	rest, ok := strings.CutPrefix(pkg, modulesPrefix)
	if !ok || rest == "" {
		return "", false
	}
	name, _, _ := strings.Cut(rest, "/")
	return name, true
}

// isUnder reports whether pkg is dir itself or a package below it.
func isUnder(pkg, dir string) bool {
	return pkg == dir || strings.HasPrefix(pkg, dir+"/")
}
