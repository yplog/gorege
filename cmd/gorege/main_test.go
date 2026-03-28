package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealMainUsage(t *testing.T) {
	if code := realMain([]string{"gorege"}); code != 2 {
		t.Fatalf("code=%d", code)
	}
	if code := realMain([]string{"gorege", "nope"}); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunCheckMissingPath(t *testing.T) {
	if code := runCheck(nil); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunCheckLoadError(t *testing.T) {
	if code := runCheck([]string{"/nonexistent/gorege-config-xyz"}); code != 1 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunCheckArityError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"ALLOW","conditions":["a"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runCheck([]string{path}); code != 1 {
		t.Fatalf("expected arity error exit 1, got %d", code)
	}
}

func TestRunCheckOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"ALLOW","conditions":["a"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := runCheck([]string{path, "a"})
	w.Close()
	_, _ = buf.ReadFrom(r)
	r.Close()
	os.Stdout = old
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if strings.TrimSpace(buf.String()) != "true" {
		t.Fatalf("stdout=%q", buf.String())
	}
}

func TestRunCheckDenyExit1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a","b"]}],
  "rules": [
    {"action":"DENY","conditions":["a"]},
    {"action":"ALLOW","conditions":["*"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runCheck([]string{path, "a"}); code != 1 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunCheckWarningsPrinted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a","b"]}],
  "rules": [
    {"action":"ALLOW","conditions":["*"]},
    {"action":"DENY","conditions":["a"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rErr, wErr, _ := os.Pipe()
	oldErr := os.Stderr
	os.Stderr = wErr
	code := runCheck([]string{path, "a"})
	wErr.Close()
	var errBuf bytes.Buffer
	_, _ = errBuf.ReadFrom(rErr)
	rErr.Close()
	os.Stderr = oldErr
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "warning:") {
		t.Fatalf("expected warning on stderr: %q", errBuf.String())
	}
}

func TestRunLintArgs(t *testing.T) {
	if code := runLint(nil); code != 2 {
		t.Fatalf("code=%d", code)
	}
	if code := runLint([]string{"a", "b"}); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunLintLoadError(t *testing.T) {
	if code := runLint([]string{"/nope"}); code != 1 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunLintOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"ALLOW","conditions":["a"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rOut, wOut, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runLint([]string{path})
	wOut.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(rOut)
	rOut.Close()
	os.Stdout = oldOut
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if strings.TrimSpace(buf.String()) != "ok" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestRunLintWarningsExit1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a","b"]}],
  "rules": [
    {"action":"ALLOW","conditions":["*"]},
    {"action":"DENY","conditions":["a"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runLint([]string{path}); code != 1 {
		t.Fatalf("code=%d", code)
	}
}
