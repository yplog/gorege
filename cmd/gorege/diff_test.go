package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yplog/gorege"
)

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

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

func TestPrintSummaryTextReturnsWriteError(t *testing.T) {
	want := errors.New("write failed")
	err := printSummaryText(failingWriter{err: want}, diffSummary{})
	if !errors.Is(err, want) {
		t.Fatalf("got error %v, want %v", err, want)
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
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyTransition(c.ox, c.nx); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// strSlicesEq is a helper that compares two string slices for equality.
func strSlicesEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSplitDiffArgs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		args      []string
		wantFlags []string
		wantPos   []string
		wantErr   bool
	}{
		{"empty", nil, nil, nil, false},
		{"positional-only", []string{"a.json", "b.json"}, nil, []string{"a.json", "b.json"}, false},
		{"limit-inline-double-dash", []string{"--limit=100", "a.json"}, []string{"--limit=100"}, []string{"a.json"}, false},
		{"limit-inline-single-dash", []string{"-limit=5"}, []string{"-limit=5"}, nil, false},
		{"format-inline-double-dash", []string{"--format=json"}, []string{"--format=json"}, nil, false},
		{"format-inline-single-dash", []string{"-format=text"}, []string{"-format=text"}, nil, false},
		{"include-unchanged-double", []string{"--include-unchanged"}, []string{"--include-unchanged"}, nil, false},
		{"include-unchanged-single", []string{"-include-unchanged"}, []string{"-include-unchanged"}, nil, false},
		{"limit-separate-args", []string{"--limit", "50", "x.json"}, []string{"--limit", "50"}, []string{"x.json"}, false},
		{"format-separate-args", []string{"--format", "json"}, []string{"--format", "json"}, nil, false},
		{"limit-missing-value", []string{"--limit"}, nil, nil, true},
		{"format-missing-value", []string{"--format"}, nil, nil, true},
		{"unknown-flag", []string{"--verbose"}, nil, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			flags, pos, err := splitDiffArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("splitDiffArgs(%v): expected error", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitDiffArgs(%v): unexpected error: %v", tc.args, err)
			}
			if !strSlicesEq(flags, tc.wantFlags) {
				t.Errorf("flags: got %v, want %v", flags, tc.wantFlags)
			}
			if !strSlicesEq(pos, tc.wantPos) {
				t.Errorf("positional: got %v, want %v", pos, tc.wantPos)
			}
		})
	}
}

func TestSortTransitions(t *testing.T) {
	t.Parallel()
	ts := []transition{
		{Kind: "UNCHANGED", Tuple: []string{"b"}},
		{Kind: "RULE_CHANGED", Tuple: []string{"c"}},
		{Kind: "DENY\u2192ALLOW", Tuple: []string{"a"}},
		{Kind: "ALLOW\u2192DENY", Tuple: []string{"z"}},
		{Kind: "ALLOW\u2192DENY", Tuple: []string{"a"}},
	}
	sortTransitions(ts)
	wantKinds := []string{
		"ALLOW\u2192DENY", "ALLOW\u2192DENY",
		"DENY\u2192ALLOW",
		"RULE_CHANGED",
		"UNCHANGED",
	}
	for i, tr := range ts {
		if tr.Kind != wantKinds[i] {
			t.Errorf("pos %d: kind=%q want %q", i, tr.Kind, wantKinds[i])
		}
	}
	// Within ALLOW→DENY, tuple tiebreak: "a" < "z".
	if ts[0].Tuple[0] != "a" || ts[1].Tuple[0] != "z" {
		t.Errorf("tuple tiebreak: pos0=%v pos1=%v, want a then z", ts[0].Tuple, ts[1].Tuple)
	}
}

func TestTransitionKindStringDefault(t *testing.T) {
	t.Parallel()
	if got := transitionKind(99).String(); got != "?" {
		t.Errorf("got %q, want %q", got, "?")
	}
}

func TestDimensionsCompatibleCountMismatch(t *testing.T) {
	t.Parallel()
	a := []gorege.Dimension{gorege.Dim("x", "a")}
	b := []gorege.Dimension{gorege.Dim("x", "a"), gorege.Dim("y", "b")}
	if err := dimensionsCompatible(a, b); err == nil {
		t.Fatal("expected error for dimension count mismatch")
	}
}

func TestDimensionsCompatibleValuesMismatch(t *testing.T) {
	t.Parallel()
	a := []gorege.Dimension{gorege.Dim("role", "admin", "user")}
	b := []gorege.Dimension{gorege.Dim("role", "admin", "guest")}
	if err := dimensionsCompatible(a, b); err == nil {
		t.Fatal("expected error when dimension values differ")
	}
}

func TestPrintSummaryTextWithChanges(t *testing.T) {
	t.Parallel()
	s := diffSummary{
		Total:            3,
		DimensionProduct: 3,
		Limit:            1000,
		AllowToDeny:      1,
		DenyToAllow:      1,
		RuleChanged:      1,
		Transitions: []transition{
			{Kind: "ALLOW\u2192DENY", Tuple: []string{"a"}},
			{Kind: "DENY\u2192ALLOW", Tuple: []string{"b"}},
			{Kind: "RULE_CHANGED", Tuple: []string{"c"}},
		},
	}
	var buf strings.Builder
	if err := printSummaryText(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "first 50 changes") {
		t.Errorf("expected 'first 50 changes' header, got:\n%s", out)
	}
	if !strings.Contains(out, "ALLOW") {
		t.Errorf("expected transition detail in output, got:\n%s", out)
	}
}

func TestPrintSummaryTextTruncation(t *testing.T) {
	t.Parallel()
	s := diffSummary{
		Total:            60,
		DimensionProduct: 60,
		Limit:            1000,
		AllowToDeny:      60,
	}
	for i := range 60 {
		s.Transitions = append(s.Transitions, transition{
			Kind:  "ALLOW\u2192DENY",
			Tuple: []string{fmt.Sprintf("v%02d", i)},
		})
	}
	var buf strings.Builder
	if err := printSummaryText(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "and 10 more") {
		t.Errorf("expected '... and 10 more' truncation line, got:\n%s", out)
	}
}

func TestRunDiffIncludeUnchanged(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld) // identical → all UNCHANGED
	out := captureStdout(t, func() {
		if code := runDiff([]string{oldP, newP, "--include-unchanged", "--format", "json"}); code != 0 {
			t.Errorf("identical configs should exit 0, got %d", code)
		}
	})
	var s diffSummary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if s.Unchanged == 0 {
		t.Errorf("expected unchanged > 0, got %d", s.Unchanged)
	}
	if len(s.Transitions) == 0 {
		t.Error("expected transitions in output with --include-unchanged")
	}
	for _, tr := range s.Transitions {
		if tr.Kind != "UNCHANGED" {
			t.Errorf("unexpected transition kind %q with --include-unchanged on identical configs", tr.Kind)
		}
	}
}

func TestRunDiffLimitZero(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld)
	if code := runDiff([]string{oldP, newP, "--limit=0"}); code != 2 {
		t.Errorf("--limit=0 should exit 2, got %d", code)
	}
}

func TestRunDiffLoadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	badP := writeJSON(t, dir, "bad.json", "not valid json{{{")
	goodP := writeJSON(t, dir, "good.json", diffOld)
	if code := runDiff([]string{badP, goodP}); code != 1 {
		t.Errorf("invalid JSON file should exit 1, got %d", code)
	}
}

func TestRunDiffTextDecisionChanges(t *testing.T) {
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "old.json", diffOld)
	newP := writeJSON(t, dir, "new.json", diffNew)
	out := captureStdout(t, func() {
		if code := runDiff([]string{oldP, newP}); code != 1 {
			t.Errorf("expected exit 1 (decision changes), got %d", code)
		}
	})
	if !strings.Contains(out, "first 50 changes") {
		t.Errorf("text output missing 'first 50 changes': %s", out)
	}
}

func TestRunDiffRuleChangedOnly(t *testing.T) {
	// Two configs: same decisions, but rule names swap → RULE_CHANGED, no allow/deny flips.
	const ruleChangedOld = `{
  "dimensions": [{"name":"role","values":["admin","user"]}],
  "rules": [
    {"action":"ALLOW","name":"r-admin","conditions":["admin"]},
    {"action":"ALLOW","name":"r-user","conditions":["user"]}
  ]
}`
	const ruleChangedNew = `{
  "dimensions": [{"name":"role","values":["admin","user"]}],
  "rules": [
    {"action":"ALLOW","name":"r-user","conditions":["user"]},
    {"action":"ALLOW","name":"r-admin","conditions":["admin"]}
  ]
}`
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "old.json", ruleChangedOld)
	newP := writeJSON(t, dir, "new.json", ruleChangedNew)
	out := captureStdout(t, func() {
		if code := runDiff([]string{oldP, newP, "--format", "json"}); code != 0 {
			t.Errorf("RULE_CHANGED only should exit 0, got %d", code)
		}
	})
	var s diffSummary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if s.RuleChanged == 0 {
		t.Error("expected RuleChanged > 0")
	}
	if s.AllowToDeny != 0 || s.DenyToAllow != 0 {
		t.Errorf("expected no decision flips, got allow_to_deny=%d deny_to_allow=%d", s.AllowToDeny, s.DenyToAllow)
	}
}

func TestProductEmptyDimension(t *testing.T) {
	t.Parallel()
	// A dimension with no values: product must return (0, false).
	dims := []gorege.Dimension{gorege.DimValues(), gorege.Dim("x", "a", "b")}
	p, exceeded := product(dims, 100)
	if p != 0 || exceeded {
		t.Errorf("product with empty dim: got (%d, %v), want (0, false)", p, exceeded)
	}
}

func TestWalkProductEmptyDims(t *testing.T) {
	t.Parallel()
	// With no dimensions, walkProduct must invoke fn exactly once with a nil tuple.
	calls := 0
	walkProduct(nil, func(_ []string) { calls++ })
	if calls != 1 {
		t.Errorf("walkProduct(nil, fn): fn called %d times, want 1", calls)
	}
}

func TestRunDiffSplitArgError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	oldP := writeJSON(t, dir, "a.json", diffOld)
	newP := writeJSON(t, dir, "b.json", diffOld)
	if code := runDiff([]string{oldP, newP, "--unknown-flag"}); code != 2 {
		t.Errorf("unknown flag should exit 2, got %d", code)
	}
}

func TestRunDiffNewLoadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goodP := writeJSON(t, dir, "good.json", diffOld)
	badP := writeJSON(t, dir, "bad.json", "not valid json{{{")
	if code := runDiff([]string{goodP, badP}); code != 1 {
		t.Errorf("invalid new JSON should exit 1, got %d", code)
	}
}

func TestPrintSummaryTextUnchangedSkipped(t *testing.T) {
	t.Parallel()
	// Ensure UNCHANGED transitions are skipped when printing the changes list.
	s := diffSummary{
		Total:            3,
		DimensionProduct: 3,
		Limit:            1000,
		AllowToDeny:      1,
		Unchanged:        2,
		Transitions: []transition{
			{Kind: "ALLOW\u2192DENY", Tuple: []string{"a"}},
			{Kind: "UNCHANGED", Tuple: []string{"b"}},
			{Kind: "UNCHANGED", Tuple: []string{"c"}},
		},
	}
	var buf strings.Builder
	if err := printSummaryText(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "UNCHANGED") {
		t.Errorf("UNCHANGED transitions should be skipped in changes list, got:\n%s", out)
	}
	if !strings.Contains(out, "ALLOW") {
		t.Errorf("expected ALLOW→DENY in output, got:\n%s", out)
	}
}

func TestSortTransitionsUnknownKind(t *testing.T) {
	t.Parallel()
	ts := []transition{
		{Kind: "UNCHANGED", Tuple: []string{"a"}},
		{Kind: "CUSTOM_KIND", Tuple: []string{"b"}},
	}
	sortTransitions(ts)
	// Unknown kind gets rank 4 (after UNCHANGED rank 3), so it sorts last.
	if ts[0].Kind != "UNCHANGED" || ts[1].Kind != "CUSTOM_KIND" {
		t.Errorf("unknown kind should sort after UNCHANGED: %v %v", ts[0].Kind, ts[1].Kind)
	}
}

func TestStringSliceEqualLengthMismatch(t *testing.T) {
	t.Parallel()
	a := []gorege.Dimension{gorege.Dim("x", "a", "b")}
	b := []gorege.Dimension{gorege.Dim("x", "a")} // same name, fewer values
	if err := dimensionsCompatible(a, b); err == nil {
		t.Fatal("expected error when dimension value count differs")
	}
}
