package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustClose(t *testing.T, f *os.File) {
	t.Helper()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

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
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := runCheck([]string{path, "a"})
	mustClose(t, w)
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	mustClose(t, r)
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
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldErr := os.Stderr
	os.Stderr = wErr
	code := runCheck([]string{path, "a"})
	mustClose(t, wErr)
	var errBuf bytes.Buffer
	if _, err := errBuf.ReadFrom(rErr); err != nil {
		t.Fatal(err)
	}
	mustClose(t, rErr)
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
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runLint([]string{path})
	mustClose(t, wOut)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rOut); err != nil {
		t.Fatal(err)
	}
	mustClose(t, rOut)
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
