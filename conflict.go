package gorege

import "strconv"

// WarningKind classifies a [Warning] from rule analysis.
type WarningKind int

const (
	// WarningKindDead means the rule never matches any tuple in the dimension
	// Cartesian product.
	WarningKindDead WarningKind = iota
	// WarningKindShadowed means the rule matches some tuple but never wins
	// first-match against earlier rules.
	WarningKindShadowed
	// WarningKindAnalysisLimitExceeded means shadowed-rule analysis (Cartesian
	// enumeration) was skipped because the dimension value product exceeded the
	// configured limit. Dead-rule detection still runs without this cap.
	WarningKindAnalysisLimitExceeded
)

// String implements [fmt.Stringer] for [WarningKind].
func (k WarningKind) String() string {
	switch k {
	case WarningKindDead:
		return "dead"
	case WarningKindShadowed:
		return "shadowed"
	case WarningKindAnalysisLimitExceeded:
		return "analysis_limit_exceeded"
	default:
		return "WarningKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// Warning describes a non-fatal issue detected at engine construction time.
type Warning struct {
	Kind    WarningKind
	Message string
}

// tupleCount computes the Cartesian product size of dimension value lists.
// If limit > 0, multiplication stops as soon as total exceeds limit (the
// full product is not computed). When over limit, the returned value is the
// running product at that step (greater than limit).
func tupleCount(dims []Dimension, limit int64) int64 {
	total := int64(1)
	for _, d := range dims {
		if len(d.values) == 0 {
			return 0
		}
		total *= int64(len(d.values))
		if limit > 0 && total > limit {
			return total
		}
	}
	return total
}

// effectiveValues returns dimension values a matcher can match against dim.
// It returns nil when the matcher can never match any declared value (empty set).
func effectiveValues(m matcher, dim Dimension) []string {
	switch m.kind {
	case mWildcard:
		return dim.values
	case mExact:
		if dim.contains(m.vals[0]) {
			return []string{m.vals[0]}
		}
		return nil
	case mAnyOf:
		out := make([]string, 0, len(m.vals))
		for _, v := range m.vals {
			if dim.contains(v) {
				out = append(out, v)
			}
		}
		return out
	default:
		return nil
	}
}

// isDeadRule reports whether r can never match any tuple in the dimension product.
// A rule is dead if some dimension has no declared values or that dimension's
// effective value set for the rule is empty.
func isDeadRule(r Rule, dims []Dimension) bool {
	for i, dim := range dims {
		var m matcher
		if i < len(r.m) {
			m = r.m[i]
		} else {
			m = matcher{kind: mWildcard}
		}
		if len(dim.values) == 0 {
			return true
		}
		if len(effectiveValues(m, dim)) == 0 {
			return true
		}
	}
	return false
}

func shadowWarnings(dims []Dimension, rules []Rule, deadMask []bool) []Warning {
	n := len(rules)
	if n == 0 {
		return nil
	}
	wins := make([]bool, n)
	d := len(dims)
	walkCartesian(dims, func(tup []string) {
		fm := -1
		for j, r := range rules {
			if deadMask[j] {
				continue
			}
			if ruleMatches(r, dims, d, tup, false) {
				if fm < 0 {
					fm = j
				}
			}
		}
		if fm >= 0 {
			wins[fm] = true
		}
	})
	var out []Warning
	for j := range rules {
		if deadMask[j] {
			continue
		}
		r := rules[j]
		label := ruleWarningLabel(j, r)
		if !wins[j] {
			out = append(out, Warning{
				Kind:    WarningKindShadowed,
				Message: "shadowed rule " + label + ": never wins first-match against earlier rules",
			})
		}
	}
	return out
}

func ruleWarningLabel(j int, r Rule) string {
	if r.Name != "" {
		return strconv.Itoa(j) + " (" + r.Name + ")"
	}
	return strconv.Itoa(j)
}

// walkCartesian calls fn for each tuple in the Cartesian product of dims' value
// lists. fn receives a reused buffer; callers must copy if they retain it.
// Empty value lists yield no calls (same as skipping shadow tuples). len(dims)==0
// invokes fn(nil) once.
func walkCartesian(dims []Dimension, fn func(tuple []string)) {
	for _, dim := range dims {
		if len(dim.values) == 0 {
			return
		}
	}
	if len(dims) == 0 {
		fn(nil)
		return
	}
	d := len(dims)
	indices := make([]int, d)
	buf := make([]string, d)
	for {
		for i, dim := range dims {
			buf[i] = dim.values[indices[i]]
		}
		fn(buf)
		carry := true
		for i := d - 1; i >= 0 && carry; i-- {
			indices[i]++
			if indices[i] < len(dims[i].values) {
				carry = false
			} else {
				indices[i] = 0
			}
		}
		if carry {
			return
		}
	}
}
