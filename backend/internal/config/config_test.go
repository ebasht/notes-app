package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		in    string
		key   string
		value string
		ok    bool
	}{
		{in: "DATABASE_URL=postgres://localhost/notes", key: "DATABASE_URL", value: "postgres://localhost/notes", ok: true},
		{in: `HTTP_ADDR="127.0.0.1:8080"`, key: "HTTP_ADDR", value: "127.0.0.1:8080", ok: true},
		{in: "export FOO=bar", key: "FOO", value: "bar", ok: true},
		{in: "TOKEN=abc=def", key: "TOKEN", value: "abc=def", ok: true},
		{in: "# comment", ok: false},
		{in: "", ok: false},
		{in: "BROKEN", ok: false},
	}

	for _, test := range tests {
		key, value, ok := parseEnvLine(test.in)
		if key != test.key || value != test.value || ok != test.ok {
			t.Fatalf("parse %q: got (%q, %q, %v)", test.in, key, value, ok)
		}
	}
}

func TestApplyEnvFileDoesNotOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	contents := "NOTES_TEST_KEY=from-file\nNOTES_TEST_OTHER=from-file\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("NOTES_TEST_KEY", "already")
	t.Setenv("NOTES_TEST_OTHER", "")

	if err := applyEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("NOTES_TEST_KEY") != "already" {
		t.Fatalf("existing env was overwritten: %q", os.Getenv("NOTES_TEST_KEY"))
	}
	if os.Getenv("NOTES_TEST_OTHER") != "from-file" {
		t.Fatalf("missing env was not loaded: %q", os.Getenv("NOTES_TEST_OTHER"))
	}
}
