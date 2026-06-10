package gorege

import "fmt"

// Engine evaluates a frozen rule set. It is safe for concurrent use.
type Engine struct {
	dims     []Dimension
	rules    []Rule
	tiebreak TiebreakStrategy
	trieRoot *ruleTrieNode // nil only when rules is empty
}

type engineConfig struct {
	dims          []Dimension
	rules         []Rule
	tb            TiebreakStrategy
	hasTB         bool
	analysisLimit int // 0 = use DefaultAnalysisLimit; negative = skip analysis
}

// Option configures [New].
type Option func(*engineConfig) error

// WithDimensions sets the ordered dimension tuple. May be empty.
func WithDimensions(dims ...Dimension) Option {
	return func(c *engineConfig) error {
		c.dims = cloneDimensions(dims)
		return nil
	}
}

// WithRules sets rules in first-match order.
func WithRules(rules ...Rule) Option {
	return func(c *engineConfig) error {
		c.rules = cloneRules(rules)
		return nil
	}
}

// WithTiebreak sets the [TiebreakStrategy] used by [Engine.Closest]. The zero
// value selects [TiebreakLeftmostDim].
func WithTiebreak(s TiebreakStrategy) Option {
	return func(c *engineConfig) error {
		c.tb = s
		c.hasTB = true
		return nil
	}
}

// WithAnalysisLimit sets the global number of tuples that shadowed-rule
// analysis may enumerate in [New]. Dead-rule detection does not use this cap.
//
//   - n == 0: use [DefaultAnalysisLimit].
//   - n < 0: skip analysis entirely (no warnings).
//   - n > 0: enumerate at most n tuples. If the full dimension product is at
//     most n it is scanned globally. Otherwise each rule is scanned only when
//     its effective product fits the remaining shared budget. Infeasible rules
//     are left unchecked without consuming budget, and analysis continues.
func WithAnalysisLimit(n int) Option {
	return func(c *engineConfig) error {
		c.analysisLimit = n
		return nil
	}
}

// New builds an immutable engine. It validates matchers against dimensions and
// returns warnings for dead or shadowed rules.
//
// Dead rules are detected without enumerating the Cartesian product. Shadowed
// rules are detected with a global tuple budget. Small products are scanned
// directly; large products are analyzed rule by rule over only the values each
// rule can match. Use [WithAnalysisLimit] to raise, lower, or disable (negative
// value) analysis. Rules whose effective products exceed the remaining budget
// produce [WarningKindAnalysisLimitExceeded]; later feasible rules are still
// analyzed.
func New(opts ...Option) (*Engine, []Warning, error) {
	var cfg engineConfig
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&cfg); err != nil {
			return nil, nil, err
		}
	}
	if err := validateEngine(cfg.dims, cfg.rules); err != nil {
		return nil, nil, err
	}
	tb := TiebreakLeftmostDim
	if cfg.hasTB {
		tb = cfg.tb
	}
	e := &Engine{
		dims:     cfg.dims,
		rules:    cfg.rules,
		tiebreak: tb,
	}
	if len(e.rules) > 0 {
		e.trieRoot = buildTrie(e.dims, e.rules)
	}
	return e, buildWarnings(e, cfg.analysisLimit), nil
}

func buildWarnings(e *Engine, configuredLimit int) []Warning {
	limit := configuredLimit
	if limit < 0 {
		return nil
	}
	if limit == 0 {
		limit = DefaultAnalysisLimit
	}

	var out []Warning
	n := len(e.rules)
	deadMask := make([]bool, n)
	for j, r := range e.rules {
		if isDeadRule(r, e.dims) {
			deadMask[j] = true
			label := ruleWarningLabel(j, r)
			out = append(out, Warning{
				Kind:    WarningKindDead,
				Message: "dead rule " + label + ": never matches any tuple in the dimension product",
			})
		}
	}

	if len(e.rules) == 0 {
		return out
	}

	count := tupleCount(e.dims, int64(limit))
	if count >= 0 && count <= int64(limit) {
		out = append(out, shadowWarningsCartesian(e.trieRoot, e.dims, e.rules, deadMask)...)
		return out
	}

	shadowed, unchecked := shadowWarningsBudgeted(e.trieRoot, e.dims, e.rules, deadMask, limit)
	out = append(out, shadowed...)
	for j, isUnchecked := range unchecked {
		if !isUnchecked {
			continue
		}
		label := ruleWarningLabel(j, e.rules[j])
		out = append(out, Warning{
			Kind: WarningKindAnalysisLimitExceeded,
			Message: fmt.Sprintf(
				"shadow analysis skipped rule %s: its effective product exceeds the remaining tuple budget",
				label,
			),
		})
	}
	return out
}

func validateEngine(dims []Dimension, rules []Rule) error {
	d := len(dims)
	for ri, r := range rules {
		if len(r.m) > d {
			return fmt.Errorf("%w (rule index %d)", ErrRuleTooWide, ri)
		}
		for i := range r.m {
			dimKnown := i < d
			var dim Dimension
			if dimKnown {
				dim = dims[i]
			}
			if err := validateMatcher(r.m[i], dim, dimKnown, ri, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateMatcher(m matcher, dim Dimension, dimKnown bool, ruleIdx, dimIdx int) error {
	if !dimKnown {
		if m.kind == mWildcard {
			return nil
		}
		if m.kind == mExact {
			v := ""
			if len(m.vals) > 0 {
				v = m.vals[0]
			}
			return fmt.Errorf("%w: rule %d dim %d exact %q with no dimension declared", ErrUnknownDimensionValue, ruleIdx, dimIdx, v)
		}
		if m.kind == mAnyOf {
			return fmt.Errorf("%w: rule %d dim %d anyOf references values with no dimension declared", ErrUnknownDimensionValue, ruleIdx, dimIdx)
		}
	}
	if m.kind == mWildcard {
		return nil
	}
	if len(dim.values) == 0 {
		// Dimension slot exists but allows any string (empty value list).
		return nil
	}
	switch m.kind {
	case mExact:
		if !dim.contains(m.vals[0]) {
			return fmt.Errorf("%w: rule %d dim %d exact %q", ErrUnknownDimensionValue, ruleIdx, dimIdx, m.vals[0])
		}
	case mAnyOf:
		for _, v := range m.vals {
			if !dim.contains(v) {
				return fmt.Errorf("%w: rule %d dim %d anyOf value %q", ErrUnknownDimensionValue, ruleIdx, dimIdx, v)
			}
		}
	}
	return nil
}

// Dimensions returns the engine dimensions in order (defensive copy).
func (e *Engine) Dimensions() []Dimension {
	return cloneDimensions(e.dims)
}

// Rules returns the rules in first-match order (defensive copy). Matchers are
// not exported; use [Rule.Name] and [Rule.Action] for inspection, or rebuild
// logic via [Engine.Check] / [Engine.Explain].
func (e *Engine) Rules() []Rule {
	return cloneRules(e.rules)
}
