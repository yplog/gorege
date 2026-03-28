package gorege

import "fmt"

// Engine evaluates a frozen rule set. It is safe for concurrent use.
type Engine struct {
	dims     []Dimension
	rules    []Rule
	tiebreak TiebreakStrategy
}

type engineConfig struct {
	dims  []Dimension
	rules []Rule
	tb    TiebreakStrategy
	hasTB bool
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

// New builds an immutable engine. It validates matchers against dimensions and
// returns warnings for dead or shadowed rules (finite Cartesian product over
// declared dimension values).
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
		dims:     cloneDimensions(cfg.dims),
		rules:    cloneRules(cfg.rules),
		tiebreak: tb,
	}
	return e, ruleWarnings(cfg.dims, cfg.rules), nil
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
			return fmt.Errorf("%w: rule %d dim %d exact %q with no dimension declared", ErrUnknownDimensionValue, ruleIdx, dimIdx, m.exact)
		}
		for _, v := range m.anyof {
			_ = v
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
		if !dim.contains(m.exact) {
			return fmt.Errorf("%w: rule %d dim %d exact %q", ErrUnknownDimensionValue, ruleIdx, dimIdx, m.exact)
		}
	case mAnyOf:
		for _, v := range m.anyof {
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
