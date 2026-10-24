// Package testenv gives integration tests the addresses of real Postgres
// and Firebase Auth emulator instances (ADR-0002: real services, no mocks).
//
// Tests read dedicated BRAINVOLT_TEST_* variables, never the application's
// own URLs, so a test can never touch a database by accident. `make test`
// sets them from .env (local Docker Compose); CI sets them for its service
// containers. When a variable is missing the test is skipped with a message
// saying how to run it, so `go test ./...` still works without Docker. A
// variable that is set but malformed (e.g. "localhost:" because a ${PORT}
// reference in .env expanded to nothing) fails the test with its name, since
// the services were clearly meant to be used. CI sets RequireVar so a missing
// variable fails instead of silently skipping the integration tests.
package testenv

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
)

const (
	// PostgresVar names the Postgres URL used by integration tests.
	PostgresVar = "BRAINVOLT_TEST_POSTGRES_URL"
	// FirebaseEmulatorVar names the Firebase Auth emulator host:port.
	FirebaseEmulatorVar = "BRAINVOLT_TEST_FIREBASE_AUTH_EMULATOR_HOST"
	// RequireVar names the switch that, set to any value, turns a missing
	// service variable into a test failure (CI).
	RequireVar = "BRAINVOLT_TEST_REQUIRE_SERVICES"
)

// PostgresURL returns the test Postgres URL or skips the test.
func PostgresURL(t testing.TB) string {
	t.Helper()
	return checked(t, PostgresVar, CheckURL)
}

// FirebaseEmulatorHost returns the Auth emulator host:port or skips the test.
func FirebaseEmulatorHost(t testing.TB) string {
	t.Helper()
	return checked(t, FirebaseEmulatorVar, CheckHostPort)
}

func checked(t testing.TB, name string, check func(string) error) string {
	t.Helper()
	v := lookup(t, name)
	if err := check(v); err != nil {
		t.Fatalf("%s=%q: %v. Run `make check-env` to find the .env line that causes it.", name, v, err)
	}
	return v
}

// CheckHostPort accepts "host:port" with a numeric port.
func CheckHostPort(v string) error {
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return fmt.Errorf("want host:port: %w", err)
	}
	if host == "" {
		return errors.New("host is empty")
	}
	if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
		return fmt.Errorf("port %q is not a number between 1 and 65535 (an undefined ${...} in .env?)", port)
	}
	return nil
}

// CheckURL accepts a URL whose host has a valid port.
func CheckURL(v string) error {
	u, err := url.Parse(v)
	if err != nil {
		return errors.New("not a valid URL")
	}
	return CheckHostPort(u.Host)
}

func lookup(t testing.TB, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" && os.Getenv(RequireVar) != "" {
		t.Fatalf("integration test: %s not set, but %s requires the services", name, RequireVar)
	}
	if v == "" {
		t.Skipf("integration test: %s not set (run `make up` and `make test`, or export it)", name)
	}
	return v
}
