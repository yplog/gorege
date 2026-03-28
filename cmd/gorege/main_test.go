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
	if code := runCheck([]string{"/nonexistent/gorege-config-xyz.json"}); code != 1 {
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

func TestRunExplainMissingPath(t *testing.T) {
	if code := runExplain(nil); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunExplainLoadError(t *testing.T) {
	if code := runExplain([]string{"/nonexistent/gorege-explain-xyz.json"}); code != 1 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunExplainArityError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"ALLOW","conditions":["a"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runExplain([]string{path}); code != 1 {
		t.Fatalf("expected arity error exit 1, got %d", code)
	}
}

func TestRunExplainMatched(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a","b"]}],
  "rules": [
    {"action":"ALLOW","name":"allow-a","conditions":["a"]},
    {"action":"DENY","name":"deny-rest","conditions":["*"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runExplain([]string{path, "a"})
	mustClose(t, wOut)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rOut); err != nil {
		t.Fatal(err)
	}
	mustClose(t, rOut)
	os.Stdout = oldOut
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "matched: true") || !strings.Contains(out, "allowed: true") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "rule_index: 0") || !strings.Contains(out, "rule_name: allow-a") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "action: ALLOW") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestRunClosestMissingPath(t *testing.T) {
	if code := runClosest(nil); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunClosestArityMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [
    {"name":"role","values":["u","v"]},
    {"name":"flag","values":["0","1"]}
  ],
  "rules": [
    {"action":"DENY","conditions":["u","0"]},
    {"action":"ALLOW","conditions":["*","*"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runClosest([]string{path, "u"}); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunClosestFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [
    {"name":"role","values":["u","v"]},
    {"name":"flag","values":["0","1"]}
  ],
  "rules": [
    {"action":"DENY","conditions":["u","0"]},
    {"action":"ALLOW","conditions":["*","*"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runClosest([]string{path, "u", "0"})
	mustClose(t, wOut)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rOut); err != nil {
		t.Fatal(err)
	}
	mustClose(t, rOut)
	os.Stdout = oldOut
	if code != 0 {
		t.Fatalf("code=%d out=%q", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "found: true") || !strings.Contains(out, "distance: 1") || !strings.Contains(out, "dim_index:") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestRunClosestNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"DENY","conditions":["*"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runClosest([]string{path, "a"})
	mustClose(t, wOut)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rOut); err != nil {
		t.Fatal(err)
	}
	mustClose(t, rOut)
	os.Stdout = oldOut
	if code != 1 {
		t.Fatalf("code=%d out=%q", code, buf.String())
	}
	if !strings.Contains(buf.String(), "found: false") {
		t.Fatalf("stdout=%q", buf.String())
	}
}

func TestRunClosestInMissingArgs(t *testing.T) {
	if code := runClosestIn([]string{"x.json"}); code != 2 {
		t.Fatalf("code=%d", code)
	}
	if code := runClosestIn(nil); code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunClosestInByIndexAndName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [
    {"name":"role","values":["u","v"]},
    {"name":"flag","values":["0","1"]}
  ],
  "rules": [
    {"action":"DENY","conditions":["u","0"]},
    {"action":"ALLOW","conditions":["*","*"]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dimArg := range []string{"1", "flag"} {
		rOut, wOut, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		oldOut := os.Stdout
		os.Stdout = wOut
		code := runClosestIn([]string{path, dimArg, "u", "0"})
		mustClose(t, wOut)
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rOut); err != nil {
			t.Fatal(err)
		}
		mustClose(t, rOut)
		os.Stdout = oldOut
		if code != 0 {
			t.Fatalf("dim=%q code=%d out=%q", dimArg, code, buf.String())
		}
		out := buf.String()
		if !strings.Contains(out, "found: true") || !strings.Contains(out, "distance: 1") || !strings.Contains(out, `"1"`) {
			t.Fatalf("dim=%q stdout=%q", dimArg, out)
		}
	}
}

func TestRunExplainImplicitDeny(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{
  "dimensions": [{"name":"x","values":["a","b"]}],
  "rules": [{"action":"DENY","conditions":["a"]}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut
	code := runExplain([]string{path, "b"})
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
	out := buf.String()
	if !strings.Contains(out, "matched: false") || !strings.Contains(out, "implicit deny") {
		t.Fatalf("stdout=%q", out)
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
