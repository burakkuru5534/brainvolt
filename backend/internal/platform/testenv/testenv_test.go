package testenv

import "testing"

func TestChecks(t *testing.T) {
	for v, ok := range map[string]bool{
		"localhost:9099": true, "firebase-auth:9099": true, "localhost:": false, "localhost": false,
		":9099": false, "localhost:abc": false, "localhost:70000": false,
	} {
		if err := CheckHostPort(v); (err == nil) != ok {
			t.Errorf("CheckHostPort(%q) = %v", v, err)
		}
	}
	for v, ok := range map[string]bool{
		"postgres://u:p@localhost:5433/db?sslmode=disable": true,
		"redis://:p@localhost:6379/15":                     true,
		"postgres://u:p@localhost:/db":                     false,
		"redis://:p@localhost/15":                          false,
	} {
		if err := CheckURL(v); (err == nil) != ok {
			t.Errorf("CheckURL(%q) = %v", v, err)
		}
	}
}
