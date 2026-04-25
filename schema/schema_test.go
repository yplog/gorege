package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaIsValidJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("gorege-config.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	for _, key := range []string{"$schema", "$id", "type", "$defs"} {
		if _, ok := v[key]; !ok {
			t.Errorf("missing required top-level key: %s", key)
		}
	}
}

func TestTestdataFixturesAreValidJSON(t *testing.T) {
	t.Parallel()

	matches, err := filepath.Glob("../testdata/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range matches {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var v any
			if err := json.Unmarshal(b, &v); err != nil {
				t.Errorf("invalid JSON: %v", err)
			}
		})
	}
}
