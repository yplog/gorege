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

// Config holds the raw configuration needed to build a gorege engine.
// Callers are responsible for parsing; this struct only carries data into the engine.
//
// To use YAML or TOML, import a parser in your own project and populate this struct, for example:
//
//	var cfg gorege.Config
//	if err := yaml.Unmarshal(data, &cfg); err != nil { ... }
//	e, _, err := gorege.NewFromConfig(cfg)
//
// The gorege module itself uses only encoding/json.
type Config struct {
	Dimensions []DimensionConfig `json:"dimensions" yaml:"dimensions"`
	Rules      []RuleConfig      `json:"rules"      yaml:"rules"`
}

// DimensionConfig describes one dimension axis.
// If Name is empty, the dimension is anonymous (DimValues semantics).
type DimensionConfig struct {
	Name   string   `json:"name"   yaml:"name"`
	Values []string `json:"values" yaml:"values"`
}

// RuleConfig describes a single rule. Each element of Conditions may be:
//   - string — exact match or "*" wildcard
//   - []any — AnyOf list (as produced by encoding/json)
//   - []string — AnyOf list (as some YAML decoders produce)
type RuleConfig struct {
	Action     string `json:"action"     yaml:"action"`
	Name       string `json:"name"       yaml:"name"`
	Conditions []any  `json:"conditions" yaml:"conditions"`
}

// NewFromConfig builds an engine from a populated Config.
// It runs the same validation and analysis as Load / LoadFileWithOptions.
// Callers are responsible for parsing.
//
// opts are applied after WithDimensions and WithRules derived from the config,
// so options such as WithAnalysisLimit and WithTiebreak can override those settings.
func NewFromConfig(cfg Config, opts ...Option) (*Engine, []Warning, error) {
	dims, err := dimensionsFromFile(cfg.Dimensions)
	if err != nil {
		return nil, nil, err
	}
	rules, err := rulesFromFile(cfg.Rules)
	if err != nil {
		return nil, nil, err
	}
	base := []Option{WithDimensions(dims...), WithRules(rules...)}
	return New(append(base, opts...)...)
}

// LoadFileWithOptions reads a JSON engine definition from path. The file
// extension must be .json. It decodes the file and calls [LoadWithOptions],
// appending opts after the JSON-derived [WithDimensions] and [WithRules].
// Pass no opts for the same behaviour as [LoadWithOptions] on the file bytes.
// Use opts to set [WithAnalysisLimit], [WithTiebreak], or other [Option]
// values when loading large configs.
//
// Hot reload: build a new engine with LoadFileWithOptions and swap a
// sync/atomic.Pointer value holding the active [*Engine] so readers always
// load through that pointer.
func LoadFileWithOptions(path string, opts ...Option) (*Engine, []Warning, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".json" {
		return nil, nil, fmt.Errorf("%w: %q", ErrUnsupportedConfigFormat, ext)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return LoadWithOptions(bytes.NewReader(b), opts...)
}

// Load decodes JSON from r into an engine. Equivalent to [LoadWithOptions]
// with no extra options.
func Load(r io.Reader) (*Engine, []Warning, error) {
	return LoadWithOptions(r)
}

// LoadWithOptions decodes JSON from r into a [Config] and calls [NewFromConfig].
// Later options override earlier ones for the same setting (e.g. a second
// [WithDimensions] replaces dimensions from JSON).
func LoadWithOptions(r io.Reader, opts ...Option) (*Engine, []Warning, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, nil, err
	}
	return NewFromConfig(cfg, opts...)
}

func dimensionsFromFile(in []DimensionConfig) ([]Dimension, error) {
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

func rulesFromFile(in []RuleConfig) ([]Rule, error) {
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
		return matcher{kind: mExact, vals: []string{x}}, nil
	case []any:
		vals := make([]string, 0, len(x))
		for j, e := range x {
			s, ok := e.(string)
			if !ok {
				return matcher{}, fmt.Errorf("anyOf element %d: want string, got %T", j, e)
			}
			vals = append(vals, s)
		}
		return matcher{kind: mAnyOf, vals: vals}, nil
	case []string:
		return matcher{kind: mAnyOf, vals: append([]string(nil), x...)}, nil
	default:
		return matcher{}, fmt.Errorf("want string, anyOf list, or *; got %T", v)
	}
}
