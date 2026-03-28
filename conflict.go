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
)

// String implements [fmt.Stringer] for [WarningKind].
func (k WarningKind) String() string {
	switch k {
	case WarningKindDead:
		return "dead"
	case WarningKindShadowed:
		return "shadowed"
	default:
		return "WarningKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// Warning describes a non-fatal issue detected at engine construction time.
type Warning struct {
	Kind    WarningKind
	Message string
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
