package gorege

// Explain returns which rule matched first, if any. Arity follows [Engine.Check].
// When no rule matches, Matched is false, Allowed is false, and RuleIndex is -1.
func (e *Engine) Explain(values ...string) (Explanation, error) {
	if len(values) != len(e.dims) {
		return Explanation{}, ErrArityMismatch
	}
	d := len(e.dims)
	for i, r := range e.rules {
		if ruleMatches(r, e.dims, d, values, false) {
			return Explanation{
				Allowed:   r.act == ActionAllow,
				RuleIndex: i,
				RuleName:  r.Name,
				Action:    r.act,
				Matched:   true,
			}, nil
		}
	}
	return Explanation{
		Allowed:   false,
		RuleIndex: -1,
		Matched:   false,
	}, nil
}

// Check evaluates the input tuple with strict arity: len(values) must equal
// the number of dimensions. First matching rule wins; if none match, false.
func (e *Engine) Check(values ...string) (bool, error) {
	if len(values) != len(e.dims) {
		return false, ErrArityMismatch
	}
	if ok, matched := e.eval(values, false); matched {
		return ok, nil
	}
	return false, nil
}

// PartialCheck allows a shorter input prefix. Trailing dimensions are
// unconstrained: a matcher at those positions is treated as satisfied for
// ALLOW rules and as failed for DENY rules (Recht-style behaviour).
//
// If len(values) is greater than the number of dimensions, it returns
// [ErrArityMismatch] so misuse is not conflated with an implicit deny (false, nil).
func (e *Engine) PartialCheck(values ...string) (bool, error) {
	if len(values) > len(e.dims) {
		return false, ErrArityMismatch
	}
	if ok, matched := e.eval(values, true); matched {
		return ok, nil
	}
	return false, nil
}

func (e *Engine) eval(values []string, partial bool) (allowed bool, matched bool) {
	d := len(e.dims)
	for _, r := range e.rules {
		if ruleMatches(r, e.dims, d, values, partial) {
			return r.act == ActionAllow, true
		}
	}
	return false, false
}

func ruleMatches(r Rule, dims []Dimension, dimCount int, values []string, partial bool) bool {
	for i := range dimCount {
		var m matcher
		if i < len(r.m) {
			m = r.m[i]
		} else {
			m = matcher{kind: mWildcard}
		}
		if partial && i >= len(values) {
			if !m.unconstrainedMatch(r.act) {
				return false
			}
			continue
		}
		dimKnown := i < dimCount
		var dim Dimension
		if dimKnown {
			dim = dims[i]
		}
		input := values[i]
		if !m.matches(input, dim, dimKnown && len(dim.values) > 0) {
			return false
		}
	}
	return true
}
