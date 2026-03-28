package gorege

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type fileDoc struct {
	Dimensions []fileDimension `json:"dimensions"`
	Rules      []fileRule      `json:"rules"`
}

type fileDimension struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type fileRule struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	Conditions []any  `json:"conditions"`
}

// LoadFile reads a JSON engine definition. The file extension must be .json.
//
// Hot reload: build a new engine with LoadFile and swap a
// sync/atomic.Pointer value holding the active [*Engine] so readers always
// load through that pointer.
func LoadFile(path string) (*Engine, []Warning, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".json" {
		return nil, nil, fmt.Errorf("%w: %q", ErrUnsupportedConfigFormat, ext)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return Load(bytes.NewReader(b))
}

// Load decodes JSON from r into an engine.
func Load(r io.Reader) (*Engine, []Warning, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var doc fileDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, err
	}
	dims, err := dimensionsFromFile(doc.Dimensions)
	if err != nil {
		return nil, nil, err
	}
	rules, err := rulesFromFile(doc.Rules)
	if err != nil {
		return nil, nil, err
	}
	return New(WithDimensions(dims...), WithRules(rules...))
}

func dimensionsFromFile(in []fileDimension) ([]Dimension, error) {
	out := make([]Dimension, 0, len(in))
	for i, d := range in {
		if len(d.Values) == 0 {
			return nil, fmt.Errorf("gorege: dimension %d has no values", i)
		}
		name := strings.TrimSpace(d.Name)
		if name == "" {
			out = append(out, DimValues(d.Values...))
			continue
		}
		out = append(out, Dim(name, d.Values...))
	}
	return out, nil
}

func rulesFromFile(in []fileRule) ([]Rule, error) {
	out := make([]Rule, 0, len(in))
	for i, r := range in {
		act, err := parseAction(r.Action)
		if err != nil {
			return nil, fmt.Errorf("gorege: rule %d: %w", i, err)
		}
		ms, err := matchersFromConditions(r.Conditions)
		if err != nil {
			return nil, fmt.Errorf("gorege: rule %d: %w", i, err)
		}
		rule := Rule{Name: r.Name, act: act, m: ms}
		out = append(out, rule)
	}
	return out, nil
}

func parseAction(s string) (Action, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ALLOW":
		return ActionAllow, nil
	case "DENY":
		return ActionDeny, nil
	default:
		return ActionDeny, fmt.Errorf("invalid action %q (want ALLOW or DENY)", s)
	}
}

func matchersFromConditions(slots []any) ([]matcher, error) {
	out := make([]matcher, 0, len(slots))
	for i, v := range slots {
		m, err := matcherFromSlot(v)
		if err != nil {
			return nil, fmt.Errorf("condition %d: %w", i, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func matcherFromSlot(v any) (matcher, error) {
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) == "*" {
			return matcher{kind: mWildcard}, nil
		}
		return matcher{kind: mExact, exact: x}, nil
	case []any:
		vals := make([]string, 0, len(x))
		for j, e := range x {
			s, ok := e.(string)
			if !ok {
				return matcher{}, fmt.Errorf("anyOf element %d: want string, got %T", j, e)
			}
			vals = append(vals, s)
		}
		return matcher{kind: mAnyOf, anyof: vals}, nil
	case []string:
		return matcher{kind: mAnyOf, anyof: append([]string(nil), x...)}, nil
	default:
		return matcher{}, fmt.Errorf("want string, anyOf list, or *; got %T", v)
	}
}
