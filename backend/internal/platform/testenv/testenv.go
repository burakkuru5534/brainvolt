// Package testenv gives integration tests the addresses of real Postgres,
// Redis and Firebase Auth emulator instances (ADR-0002: real services, no mocks).
//
// Tests read dedicated BRAINVOLT_TEST_* variables, never the application's
// own URLs, so a test can never touch a database by accident. `make test`
// sets them from .env (local Docker Compose); CI sets them for its service
// containers. When a variable is missing the test is skipped with a message
// saying how to run it, so `go test ./...` still works without Docker.
package testenv

import (
	"os"
	"testing"
)

const (
	// PostgresVar names the Postgres URL used by integration tests.
	PostgresVar = "BRAINVOLT_TEST_POSTGRES_URL"
	// RedisVar names the Redis URL used by integration tests.
	RedisVar = "BRAINVOLT_TEST_REDIS_URL"
	// FirebaseEmulatorVar names the Firebase Auth emulator host:port.
	FirebaseEmulatorVar = "BRAINVOLT_TEST_FIREBASE_AUTH_EMULATOR_HOST"
)

// PostgresURL returns the test Postgres URL or skips the test.
func PostgresURL(t testing.TB) string {
	t.Helper()
	return lookup(t, PostgresVar)
}

// RedisURL returns the test Redis URL or skips the test.
func RedisURL(t testing.TB) string {
	t.Helper()
	return lookup(t, RedisVar)
}

// FirebaseEmulatorHost returns the Auth emulator host:port or skips the test.
func FirebaseEmulatorHost(t testing.TB) string {
	t.Helper()
	return lookup(t, FirebaseEmulatorVar)
}

func lookup(t testing.TB, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("integration test: %s not set (run `make up` and `make test`, or export it)", name)
	}
	return v
}
