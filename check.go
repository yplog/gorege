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

// PartialCheck allows a shorter input prefix (including an empty prefix).
// Trailing dimensions are unconstrained: a matcher at those positions is
// treated as satisfied for ALLOW rules and as failed for DENY rules
// (Recht-style behaviour). The empty prefix means “no values fixed yet”:
// it is not an arity error (unlike [Engine.Check], which requires a full tuple).
// Semantically it answers whether any completion could still be allowed—for
// example, after PartialCheck("Guest") asks whether Guest can access for some
// day, PartialCheck() asks whether anyone can access for some full tuple.
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
	if !partial && e.trieRoot != nil {
		idx := e.trieRoot.search(values, e.dims, 0)
		if idx == noMatch {
			return false, false
		}
		return e.rules[idx].act == ActionAllow, true
	}
	d := len(e.dims)
	for _, r := range e.rules {
		if ruleMatches(r, e.dims, d, values, partial) {
			return r.act == ActionAllow, true
		}
	}
	return false, false
}

func ruleMatches(r Rule, dims []Dimension, dimCount int, values []string, partial bool) bool {
	mlen := len(r.m)
	for i := range dimCount {
		if partial && i >= len(values) {
			if i < mlen {
				if !r.m[i].unconstrainedMatch(r.act) {
					return false
				}
			}
			continue
		}
		dim := dims[i]
		input := values[i]
		if i >= mlen {
			if len(dim.values) > 0 && !dim.contains(input) {
				return false
			}
			continue
		}
		if !r.m[i].matches(input, dim) {
			return false
		}
	}
	return true
}
