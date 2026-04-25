package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yplog/gorege"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func writeJSON(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const diffOld = `{
  "dimensions": [
    {"name":"role","values":["admin","user","guest"]},
    {"name":"action","values":["read","write"]}
  ],
  "rules": [
    {"action":"ALLOW","name":"admin-all","conditions":["admin","*"]},
    {"action":"ALLOW","name":"user-read","conditions":["user","read"]},
    {"action":"DENY","name":"deny-rest","conditions":["*","*"]}
  ]
}`

const diffNew = `{
  "dimensions": [
    {"name":"role","values":["admin","user","guest"]},
    {"name":"action","values":["read","write"]}
  ],
  "rules": [
    {"action":"ALLOW","name":"admin-all","conditions":["admin","*"]},
    {"action":"ALLOW","name":"user-write","conditions":["user","write"]},
    {"action":"DENY","name":"deny-rest","conditions":["*","*"]}
  ]
}`

func TestRunDiffDecisionChanges(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "old.json", diffOld)
	newP := writeJSON(t, dir, "new.json", diffNew)

	out := captureStdout(t, func() {
		if code := runDiff([]string{oldP, newP, "--format", "json"}); code != 1 {
			t.Errorf("expected exit 1 (decision changes), got %d", code)
		}
	})
	var s diffSummary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if s.AllowToDeny != 1 || s.DenyToAllow != 1 {
		t.Errorf("got A→D=%d D→A=%d, want 1 each", s.AllowToDeny, s.DenyToAllow)
	}
	if s.Total != 6 {
		t.Errorf("total=%d, want 6", s.Total)
	}
}

func TestRunDiffNoChanges(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld)
	if code := runDiff([]string{oldP, newP}); code != 0 {
		t.Errorf("identical configs should exit 0, got %d", code)
	}
}

func TestRunDiffDimensionMismatch(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "old.json", diffOld)
	altered := strings.Replace(diffOld, `"role"`, `"actor"`, 1)
	newP := writeJSON(t, dir, "new.json", altered)
	if code := runDiff([]string{oldP, newP}); code != 1 {
		t.Errorf("dim name change should exit 1, got %d", code)
	}
}

func TestRunDiffLimitExceeded(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld)
	if code := runDiff([]string{oldP, newP, "--limit", "1"}); code != 1 {
		t.Errorf("expected exit 1 on limit exceeded, got %d", code)
	}
}

func TestRunDiffMissingArgs(t *testing.T) {
	if code := runDiff([]string{"only-one"}); code != 2 {
		t.Errorf("expected usage exit 2, got %d", code)
	}
}

func TestRunDiffUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld)
	if code := runDiff([]string{oldP, newP, "--format", "xml"}); code != 2 {
		t.Errorf("unknown format should exit 2, got %d", code)
	}
}

func TestClassifyTransition(t *testing.T) {
	cases := []struct {
		name string
		ox   gorege.Explanation
		nx   gorege.Explanation
		want transitionKind
	}{
		{"both-allow-same-rule",
			gorege.Explanation{Matched: true, Allowed: true, RuleIndex: 0, RuleName: "r"},
			gorege.Explanation{Matched: true, Allowed: true, RuleIndex: 0, RuleName: "r"},
			transUnchanged},
		{"allow-to-deny",
			gorege.Explanation{Matched: true, Allowed: true},
			gorege.Explanation{Matched: true, Allowed: false},
			transAllowToDeny},
		{"implicit-deny-to-allow",
			gorege.Explanation{Matched: false},
			gorege.Explanation{Matched: true, Allowed: true},
			transDenyToAllow},
		{"same-decision-different-rule",
			gorege.Explanation{Matched: true, Allowed: true, RuleIndex: 0, RuleName: "a"},
			gorege.Explanation{Matched: true, Allowed: true, RuleIndex: 1, RuleName: "b"},
			transRuleChanged},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyTransition(c.ox, c.nx); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
