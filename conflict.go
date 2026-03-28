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
	// WarningKindAnalysisLimitExceeded means dead/shadowed analysis was skipped
	// because the dimension value product exceeded the configured limit.
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

func ruleWarnings(dims []Dimension, rules []Rule) []Warning {
	tuples := cartesianProduct(dims)
	n := len(rules)
	if n == 0 {
		return nil
	}
	wins := make([]bool, n)
	matches := make([]bool, n)
	d := len(dims)
	for _, tup := range tuples {
		fm := -1
		for j, r := range rules {
			if ruleMatches(r, dims, d, tup, false) {
				matches[j] = true
				if fm < 0 {
					fm = j
				}
			}
		}
		if fm >= 0 {
			wins[fm] = true
		}
	}
	var out []Warning
	for j := range rules {
		r := rules[j]
		label := ruleWarningLabel(j, r)
		switch {
		case !matches[j]:
			out = append(out, Warning{
				Kind:    WarningKindDead,
				Message: "dead rule " + label + ": never matches any tuple in the dimension product",
			})
		case !wins[j]:
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

func cartesianProduct(dims []Dimension) [][]string {
	if len(dims) == 0 {
		return [][]string{{}}
	}
	out := [][]string{{}}
	for _, dim := range dims {
		if len(dim.values) == 0 {
			return nil
		}
		var next [][]string
		for _, prefix := range out {
			for _, v := range dim.values {
				t := append(append([]string(nil), prefix...), v)
				next = append(next, t)
			}
		}
		out = next
	}
	return out
}
