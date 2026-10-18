package config

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

// EnvFile is a parsed .env file: KEY=VALUE lines, # comments and blank lines.
type EnvFile struct {
	Keys []string          // in file order
	Raw  map[string]string // values as written, ${VAR} references unexpanded
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ReadEnvFile parses path.
func ReadEnvFile(path string) (EnvFile, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path is a developer-supplied .env file; reading it is the point
	if err != nil {
		return EnvFile{}, err
	}
	f := EnvFile{Raw: map[string]string{}}
	n := 0
	for line := range strings.Lines(string(data)) {
		n++
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || !envKey.MatchString(k) {
			return EnvFile{}, fmt.Errorf("%s:%d: want KEY=VALUE, got %q", path, n, line)
		}
		if _, dup := f.Raw[k]; !dup {
			f.Keys = append(f.Keys, k)
		}
		f.Raw[k] = v
	}
	return f, nil
}

var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Expand resolves ${VAR} references the way Docker Compose, the Makefile and
// `set -a; . ./.env` do: from keys defined earlier in the file, otherwise
// from lookup (the process environment). Undefined references expand to "".
func (f EnvFile) Expand(lookup func(string) (string, bool)) map[string]string {
	out := make(map[string]string, len(f.Keys))
	for _, k := range f.Keys {
		out[k] = envRef.ReplaceAllStringFunc(f.Raw[k], func(ref string) string {
			name := envRef.FindStringSubmatch(ref)[1]
			if v, ok := out[name]; ok {
				return v
			}
			v, _ := lookup(name)
			return v
		})
	}
	return out
}

// CheckEnvFile compares a developer's .env with .env.example and returns one
// readable problem per issue: keys missing from .env, and ${VAR} references
// that would expand to nothing (a classic cause of "localhost:" addresses).
func CheckEnvFile(example, actual EnvFile, lookup func(string) (string, bool)) []string {
	var problems []string
	for _, k := range example.Keys {
		if _, ok := actual.Raw[k]; !ok {
			problems = append(problems, fmt.Sprintf("%s is missing from .env (copy it from .env.example: %s=%s)", k, k, example.Raw[k]))
		}
	}
	defined := map[string]bool{}
	for _, k := range actual.Keys {
		for _, m := range envRef.FindAllStringSubmatch(actual.Raw[k], -1) {
			name := m[1]
			if defined[name] {
				continue
			}
			if _, ok := lookup(name); ok {
				continue
			}
			msg := fmt.Sprintf("%s uses ${%s}, which is not defined above it in .env, so it expands to an empty string", k, name)
			if !slices.Contains(actual.Keys, name) {
				if v, ok := example.Raw[name]; ok {
					msg += fmt.Sprintf(" (add %s=%s)", name, v)
				}
			} else {
				msg += " (move its definition above this line)"
			}
			problems = append(problems, msg)
		}
		defined[k] = true
	}
	return problems
}
