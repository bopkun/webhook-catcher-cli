package app

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// captureStdout captures stdout during fn execution and returns captured string.
func captureStdout(fn func()) string {
	// Save original stdout
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function
	fn()

	// Restore and read
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestTryPrettyJSON_Malformed(t *testing.T) {
	if s, ok := TryPrettyJSON([]byte("{")); ok || s != "" {
		t.Fatalf("expected malformed JSON to return ok=false, got ok=%v s=%q", ok, s)
	}
}

func TestSaveEnvVar_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env.new"
	if err := SaveEnvVar(path, "A", "1"); err != nil {
		t.Fatalf("saveEnvVar new file failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read new env file failed: %v", err)
	}
	if strings.TrimSpace(string(b)) != "A=1" {
		t.Fatalf("expected 'A=1', got %q", string(b))
	}
}

func TestTryPrettyJSON(t *testing.T) {
	// empty
	if s, ok := TryPrettyJSON([]byte("")); ok || s != "" {
		t.Fatalf("expected empty=false, got ok=%v s=%q", ok, s)
	}
	// invalid
	if s, ok := TryPrettyJSON([]byte("notjson")); ok || s != "" {
		t.Fatalf("expected invalid=false, got ok=%v s=%q", ok, s)
	}
	// object
	s, ok := TryPrettyJSON([]byte(`{"a":1}`))
	if !ok || !strings.Contains(s, `"a": 1`) {
		t.Fatalf("expected pretty JSON object, got ok=%v s=%q", ok, s)
	}
	// array
	s, ok = TryPrettyJSON([]byte(`[1,2]`))
	if !ok || !strings.Contains(s, "1") || !strings.Contains(s, "2") {
		t.Fatalf("expected pretty JSON array, got ok=%v s=%q", ok, s)
	}
}

func TestColorMethod(t *testing.T) {
	cases := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, m := range cases {
		col := ColorMethod(m)
		if !strings.Contains(col, m) {
			t.Fatalf("expected colored string to contain method %s, got %q", m, col)
		}
	}
}

func TestLoadDotEnvAndSaveEnvVar(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/envtest"
	content := "" +
		"# comment\n" +
		"FOO=bar\n" +
		"QUOTED=\"baz qux\"\n" +
		"SPACED = value\n" +
		"BADLINE\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp env failed: %v", err)
	}

	LoadDotEnv(path)
	if os.Getenv("FOO") != "bar" {
		t.Fatalf("expected FOO=bar, got %q", os.Getenv("FOO"))
	}
	if os.Getenv("QUOTED") != "baz qux" {
		t.Fatalf("expected QUOTED=\"baz qux\", got %q", os.Getenv("QUOTED"))
	}
	if os.Getenv("SPACED") != "value" {
		t.Fatalf("expected SPACED=value, got %q", os.Getenv("SPACED"))
	}

	// saveEnvVar updates existing
	if err := SaveEnvVar(path, "FOO", "new"); err != nil {
		t.Fatalf("saveEnvVar update failed: %v", err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "FOO=new") {
		t.Fatalf("expected file to contain updated FOO=new, got: %s", string(b))
	}
	// saveEnvVar appends new
	if err := SaveEnvVar(path, "NEW", "x"); err != nil {
		t.Fatalf("saveEnvVar append failed: %v", err)
	}
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "NEW=x") {
		t.Fatalf("expected file to contain NEW=x, got: %s", string(b))
	}
}

func TestLoadDotEnv_CRLF(t *testing.T) {
    dir := t.TempDir()
    p := dir + "/.env"
    content := "#cmt\r\nFOO=bar\r\nQUOTED=\"hi\"\r\n"
    if err := os.WriteFile(p, []byte(content), 0600); err != nil {
        t.Fatalf("write failed: %v", err)
    }
    _ = os.Unsetenv("FOO")
    _ = os.Unsetenv("QUOTED")
    LoadDotEnv(p)
    if os.Getenv("FOO") != "bar" {
        t.Fatalf("expected FOO=bar, got %q", os.Getenv("FOO"))
    }
    if os.Getenv("QUOTED") != "hi" {
        t.Fatalf("expected QUOTED=hi, got %q", os.Getenv("QUOTED"))
    }
}
