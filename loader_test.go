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
	e, warnings, err := gorege.LoadFile(path)
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
	_, _, err := gorege.LoadFile(path)
	if !errors.Is(err, gorege.ErrUnsupportedConfigFormat) {
		t.Fatalf("got %v", err)
	}
}

func TestLoadFileMissingPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nope.json")
	_, _, err := gorege.LoadFile(path)
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
			e, warnings, err := gorege.LoadFile(path)
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
    {"action": "ALLOW", "conditions": ["*", "*", "*"]}
  ]
}`
	_, warnings, err := gorege.LoadWithOptions(
		strings.NewReader(doc),
		gorege.WithAnalysisLimit(100),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("got %v", warnings)
	}
	if want := "~125 tuples"; !strings.Contains(warnings[0].Message, want) {
		t.Fatalf("message %q should contain %q", warnings[0].Message, want)
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
