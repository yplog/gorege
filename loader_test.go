package gorege_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yplog/gorege"
)

func TestLoadFileJSONQuickStart(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rules.json")
	e, warnings, err := gorege.LoadFileWithOptions(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}
	ok, err := e.Check("Guest", "Mon", "Sauna")
	if err != nil || ok {
		t.Fatalf("Guest Mon Sauna: ok=%v err=%v", ok, err)
	}
	ok, err = e.Check("Guest", "Wed", "Sauna")
	if err != nil || !ok {
		t.Fatalf("Guest Wed Sauna: ok=%v err=%v", ok, err)
	}
}

func TestLoadFileUnsupportedExt(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := gorege.LoadFileWithOptions(path)
	if !errors.Is(err, gorege.ErrUnsupportedConfigFormat) {
		t.Fatalf("got %v", err)
	}
}

func TestLoadFileMissingPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nope.json")
	_, _, err := gorege.LoadFileWithOptions(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestLoadReadError(t *testing.T) {
	t.Parallel()
	_, _, err := gorege.Load(errReader{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadJSONSyntaxError(t *testing.T) {
	t.Parallel()
	_, _, err := gorege.Load(strings.NewReader("{"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadDimensionNoValues(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":[]}],"rules":[]}`
	_, _, err := gorege.Load(strings.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "dimension 0") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadBadAction(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"MAYBE","conditions":["a"]}]}`
	_, _, err := gorege.Load(strings.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "invalid action") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadBadConditionScalar(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","conditions":[1]}]}`
	_, _, err := gorege.Load(strings.NewReader(doc))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadAnyOfElementNotString(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","conditions":[[1]]}]}`
	_, _, err := gorege.Load(strings.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "anyOf element") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadJSONAnonymousDimension(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"values":["a","b"]}],"rules":[{"action":"ALLOW","conditions":["*"]}]}`
	e, _, err := gorege.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("a")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestLoadJSONWildcardTrimmed(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","conditions":["  *  "]}]}`
	e, _, err := gorege.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("a")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestLoadIgnoresSchemaField(t *testing.T) {
	t.Parallel()
	doc := `{
  "$schema": "https://example.com/schema.json",
  "dimensions": [{"name":"x","values":["a"]}],
  "rules": [{"action":"ALLOW","conditions":["a"]}]
}`
	e, _, err := gorege.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("a")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestLoadFileExampleFixtures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file     string
		check    []string
		want     bool
		wantWarn bool
	}{
		{"minimal.json", []string{"on"}, true, false},
		{"minimal.json", []string{"off"}, false, false},
		{"feature-toggle.json", []string{"prod", "legacy_only"}, true, false},
		{"feature-toggle.json", []string{"prod", "beta_search"}, false, false},
		{"feature-toggle.json", []string{"dev", "beta_search"}, true, false},
		{"ecom-availability.json", []string{"JP", "SKU-A"}, true, false},
		{"ecom-availability.json", []string{"JP", "SKU-B"}, false, false},
		{"ecom-availability.json", []string{"US", "SKU-B"}, false, false},
		{"with-shadow-warnings.json", []string{"acme"}, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.file+"/"+strings.Join(tc.check, ","), func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("testdata", tc.file)
			e, warnings, err := gorege.LoadFileWithOptions(path)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantWarn {
				if len(warnings) == 0 {
					t.Fatal("expected non-empty warnings")
				}
				if tc.file == "with-shadow-warnings.json" && warnings[0].Kind != gorege.WarningKindShadowed {
					t.Fatalf("want shadowed warning, got kind=%v", warnings[0].Kind)
				}
			} else if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %v", warnings)
			}
			ok, err := e.Check(tc.check...)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.want {
				t.Fatalf("Check(...) = %v, want %v", ok, tc.want)
			}
		})
	}
}

func TestLoadWithOptionsAnalysisLimitNegativeSkipsWarnings(t *testing.T) {
	t.Parallel()
	doc := `{
  "dimensions": [
    {"values": ["a", "b"]}
  ],
  "rules": [
    {"action": "ALLOW", "conditions": ["*"]},
    {"action": "DENY", "conditions": ["a"]}
  ]
}`
	_, warnings, err := gorege.LoadWithOptions(
		strings.NewReader(doc),
		gorege.WithAnalysisLimit(-1),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no analysis warnings, got %v", warnings)
	}
}

func TestLoadWithOptionsAnalysisLimitExceeded(t *testing.T) {
	t.Parallel()
	// 5×5×5 = 125 > limit 100
	doc := `{
  "dimensions": [
    {"values": ["a", "b", "c", "d", "e"]},
    {"values": ["1", "2", "3", "4", "5"]},
    {"values": ["x", "y", "z", "w", "v"]}
  ],
  "rules": [
    {"action": "ALLOW", "conditions": ["*", "*", "*"]},
    {"action": "DENY", "conditions": ["*", "*", "*"]}
  ]
}`
	_, warnings, err := gorege.LoadWithOptions(
		strings.NewReader(doc),
		gorege.WithAnalysisLimit(100),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 ||
		warnings[0].Kind != gorege.WarningKindAnalysisLimitExceeded ||
		warnings[1].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("got %v", warnings)
	}
	for i, want := range []string{"rule 0", "rule 1"} {
		if !strings.Contains(warnings[i].Message, want) {
			t.Fatalf("message %q should contain %q", warnings[i].Message, want)
		}
	}
}

func TestLoadFileWithOptionsPassesOptions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	doc := `{
  "dimensions": [{"values": ["a", "b"]}],
  "rules": [
    {"action": "ALLOW", "conditions": ["*"]},
    {"action": "DENY", "conditions": ["a"]}
  ]
}`
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	_, warnings, err := gorege.LoadFileWithOptions(path, gorege.WithAnalysisLimit(-1))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestNewFromConfigRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Name: "role", Values: []string{"admin", "user"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "ALLOW", Name: "allow-admin", Conditions: []any{"admin"}},
			{Action: "DENY", Name: "deny-rest", Conditions: []any{"*"}},
		},
	}
	e, warnings, err := gorege.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}
	ok, err := e.Check("admin")
	if err != nil || !ok {
		t.Fatalf("Check(admin): ok=%v err=%v", ok, err)
	}
	ok, err = e.Check("user")
	if err != nil || ok {
		t.Fatalf("Check(user): ok=%v err=%v", ok, err)
	}
}

func TestNewFromConfigAnyOfAsStringSlice(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Values: []string{"a", "b", "c"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "ALLOW", Conditions: []any{[]string{"a", "b"}}},
			{Action: "DENY", Conditions: []any{"*"}},
		},
	}
	e, _, err := gorege.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("a")
	if err != nil || !ok {
		t.Fatalf("Check(a): ok=%v err=%v", ok, err)
	}
	ok, err = e.Check("c")
	if err != nil || ok {
		t.Fatalf("Check(c): ok=%v err=%v", ok, err)
	}
}

func TestNewFromConfigWithOptions(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Values: []string{"a", "b"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "ALLOW", Conditions: []any{"*"}},
			{Action: "DENY", Conditions: []any{"a"}},
		},
	}
	_, warnings, err := gorege.NewFromConfig(cfg, gorege.WithAnalysisLimit(-1))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no analysis warnings, got %v", warnings)
	}
}

func TestNewFromConfigInvalidAction(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Name: "x", Values: []string{"a"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "MAYBE", Conditions: []any{"*"}},
		},
	}
	_, _, err := gorege.NewFromConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid action") {
		t.Fatalf("got %v", err)
	}
}

func TestNewFromConfigUnknownDimensionValue(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Name: "role", Values: []string{"admin", "user"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "ALLOW", Conditions: []any{"superuser"}},
		},
	}
	_, _, err := gorege.NewFromConfig(cfg)
	if err == nil || !errors.Is(err, gorege.ErrUnknownDimensionValue) {
		t.Fatalf("got %v", err)
	}
}

func TestNewFromConfigDimensionNoValues(t *testing.T) {
	t.Parallel()
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Name: "x", Values: nil},
		},
		Rules: []gorege.RuleConfig{},
	}
	_, _, err := gorege.NewFromConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "dimension 0") {
		t.Fatalf("got %v", err)
	}
}

// TestLoadDimensionWhitespaceOnlyName verifies that a dimension whose JSON
// "name" field contains only whitespace is trimmed to an empty string and
// treated as an anonymous dimension (DimValues semantics).
func TestLoadDimensionWhitespaceOnlyName(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"  ","values":["a","b"]}],"rules":[{"action":"ALLOW","conditions":["*"]}]}`
	e, _, err := gorege.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	dims := e.Dimensions()
	if len(dims) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(dims))
	}
	if dims[0].Name() != "" {
		t.Fatalf("expected anonymous dim (name=%q), got name=%q", "", dims[0].Name())
	}
}

// TestLoadActionCaseInsensitive verifies that parseAction normalises the action
// string via strings.ToUpper and strings.TrimSpace, so lowercase and padded
// variants of "allow" and "deny" are accepted.
func TestLoadActionCaseInsensitive(t *testing.T) {
	t.Parallel()
	for _, act := range []string{"allow", "Allow", " ALLOW ", "deny", "Deny", " DENY "} {
		doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"` + act + `","conditions":["a"]}]}`
		_, _, err := gorege.Load(strings.NewReader(doc))
		if err != nil {
			t.Errorf("action %q: unexpected error %v", act, err)
		}
	}
}
