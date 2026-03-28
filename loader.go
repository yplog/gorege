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

// Config, gorege motorunu ayağa kaldırmak için gereken ham konfigürasyonu
// temsil eder. Parsing sorumluluğu çağırana aittir; bu struct yalnızca
// motora veri taşır.
//
// YAML veya TOML kullanmak isteyen kullanıcılar kendi projesinde istediği
// parser kütüphanesini import eder ve veriyi bu struct'a doldurur:
//
//	var cfg gorege.Config
//	yaml.Unmarshal(data, &cfg)          // kendi projenizde
//	e, _, _ := gorege.NewFromConfig(cfg)
//
// gorege kendisi yalnızca encoding/json kullanır.
type Config struct {
	Dimensions []DimensionConfig `json:"dimensions" yaml:"dimensions"`
	Rules      []RuleConfig      `json:"rules"      yaml:"rules"`
}

// DimensionConfig, bir dimension eksenini tanımlar.
// Name boş bırakılırsa anonim dimension (DimValues semantiği) oluşturulur.
type DimensionConfig struct {
	Name   string   `json:"name"   yaml:"name"`
	Values []string `json:"values" yaml:"values"`
}

// RuleConfig, tek bir kuralı tanımlar. Conditions içindeki her slot şunlardan
// biri olabilir:
//   - string       → exact eşleşme veya "*" wildcard
//   - []any        → AnyOf listesi (encoding/json bu tipi üretir)
//   - []string     → AnyOf listesi (YAML parser'lar bu tipi üretebilir)
type RuleConfig struct {
	Action     string `json:"action"     yaml:"action"`
	Name       string `json:"name"       yaml:"name"`
	Conditions []any  `json:"conditions" yaml:"conditions"`
}

// NewFromConfig, önceden doldurulmuş bir Config'den engine oluşturur.
// Load / LoadFile ile aynı validation ve analiz adımlarını çalıştırır.
// Parsing sorumluluğu çağırana aittir.
//
// opts, JSON'dan türetilen WithDimensions ve WithRules'tan sonra uygulanır;
// dolayısıyla WithAnalysisLimit, WithTiebreak gibi seçenekler geçersiz kılınabilir.
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

// LoadFile reads a JSON engine definition. The file extension must be .json.
// It is equivalent to [LoadFileWithOptions] with no extra options.
//
// Hot reload: build a new engine with LoadFile and swap a
// sync/atomic.Pointer value holding the active [*Engine] so readers always
// load through that pointer.
func LoadFile(path string) (*Engine, []Warning, error) {
	return LoadFileWithOptions(path)
}

// LoadFileWithOptions reads a JSON engine like [LoadFile], then builds the
// engine with [New], appending opts after the JSON-derived [WithDimensions]
// and [WithRules]. Use this to pass [WithAnalysisLimit], [WithTiebreak], or
// other [Option] values when loading large configs.
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
