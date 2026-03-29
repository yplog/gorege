package gorege

import "fmt"

// Action is ALLOW or DENY.
type Action bool

const (
	ActionAllow Action = true
	ActionDeny  Action = false
)

// Rule is a single first-match rule. Use Allow / Deny constructors.
type Rule struct {
	Name string
	act  Action
	m    []matcher
}

// Allow builds an ALLOW rule. Each part may be:
//   - string — exact match
//   - WildcardType — match any declared value in that dimension (or any string if the engine has no dimensions)
//   - anyOf — match any of the listed values (from AnyOf)
func Allow(parts ...any) Rule {
	return Rule{act: ActionAllow, m: parseMatchers(parts)}
}

// Deny builds a DENY rule. Arguments follow the same rules as [Allow].
func Deny(parts ...any) Rule {
	return Rule{act: ActionDeny, m: parseMatchers(parts)}
}

// WildcardType marks a dimension slot as matching any declared value (when
// dimensions exist) or any input (when the engine has zero dimensions).
type WildcardType struct{}

// Wildcard matches any value in the corresponding dimension. If the engine
// has no dimensions, it matches any input at that position.
var Wildcard WildcardType

type anyOf []string

// AnyOf matches if the input equals any of vals.
func AnyOf(vals ...string) anyOf {
	return append(anyOf(nil), vals...)
}

type matcherKind uint8

const (
	mExact matcherKind = iota
	mAnyOf
	mWildcard
)

type matcher struct {
	kind matcherKind
	vals []string // nil: wildcard; len==1: exact; len>=2: anyOf
}

func parseMatchers(parts []any) []matcher {
	out := make([]matcher, 0, len(parts))
	for _, p := range parts {
		out = append(out, mustMatcher(p))
	}
	return out
}

func mustMatcher(p any) matcher {
	switch x := p.(type) {
	case string:
		return matcher{kind: mExact, vals: []string{x}}
	case WildcardType:
		return matcher{kind: mWildcard}
	case anyOf:
		return matcher{kind: mAnyOf, vals: append([]string(nil), x...)}
	default:
		panic(fmt.Sprintf("gorege: invalid matcher type %T (use string, Wildcard, or AnyOf)", p))
	}
}

func (m matcher) matches(input string, dim Dimension) bool {
	declared := len(dim.values) > 0
	switch m.kind {
	case mWildcard:
		if !declared {
			return true
		}
		return dim.contains(input)
	case mExact:
		if m.vals[0] != input {
			return false
		}
		if declared && !dim.contains(input) {
			// Validated at New; defensive for tests without full engine.
			return false
		}
		return true
	case mAnyOf:
		for _, v := range m.vals {
			if v == input {
				if declared && !dim.contains(input) {
					return false
				}
				return true
			}
		}
		return false
	default:
		return false
	}
}

// unconstrainedMatch reports whether a matcher is satisfied when the input
// tuple has no value at this dimension (PartialCheck trailing dimensions).
func (m matcher) unconstrainedMatch(act Action) bool {
	if m.kind == mWildcard {
		return true
	}
	if act == ActionAllow {
		return true
	}
	return false
}

func cloneRules(rules []Rule) []Rule {
	out := make([]Rule, len(rules))
	for i := range rules {
		out[i] = Rule{
			Name: rules[i].Name,
			act:  rules[i].act,
			m:    cloneMatchers(rules[i].m),
		}
	}
	return out
}

func cloneMatchers(m []matcher) []matcher {
	out := make([]matcher, len(m))
	for i, v := range m {
		out[i] = matcher{kind: v.kind}
		if v.vals != nil {
			out[i].vals = append([]string(nil), v.vals...)
		}
	}
	return out
}
