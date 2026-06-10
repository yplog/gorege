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
	// WarningKindAnalysisLimitExceeded means a rule's effective product
	// exceeded the remaining global tuple budget, so its shadow status was
	// left unchecked.
	// Dead-rule detection still runs without this cap.
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
// If limit > 0, multiplication stops as soon as total exceeds limit. The
// returned value is greater than limit in that case, or -1 if the product
// overflows int64.
func tupleCount(dims []Dimension, limit int64) int64 {
	total := int64(1)
	for _, d := range dims {
		if len(d.values) == 0 {
			return 0
		}
		var exceeded bool
		total, exceeded = multiplyCount(total, len(d.values), limit)
		if exceeded {
			return total
		}
	}
	return total
}

func multiplyCount(total int64, factor int, limit int64) (int64, bool) {
	f := int64(factor)
	const maxInt64 = int64(^uint64(0) >> 1)
	if total > maxInt64/f {
		return -1, true
	}
	product := total * f
	if limit > 0 && product > limit {
		return product, true
	}
	return product, false
}

func multiplyWithinBudget(total int64, factor int, remaining int64) (int64, bool) {
	f := int64(factor)
	if total > remaining/f {
		return total, false
	}
	return total * f, true
}

// matcherHasEffectiveValue reports whether m can match a declared dim value.
func matcherHasEffectiveValue(m matcher, dim Dimension) bool {
	switch m.kind {
	case mWildcard:
		return len(dim.values) > 0
	case mExact:
		return len(m.vals) == 1 && dim.contains(m.vals[0])
	case mAnyOf:
		for _, v := range m.vals {
			if dim.contains(v) {
				return true
			}
		}
		return false
	default:
		return false
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
		if !matcherHasEffectiveValue(m, dim) {
			return true
		}
	}
	return false
}

func shadowWarningsCartesian(root *ruleTrieNode, dims []Dimension, rules []Rule, deadMask []bool) []Warning {
	n := len(rules)
	if n == 0 {
		return nil
	}
	wins := make([]bool, n)
	liveRemaining := 0
	for j := range rules {
		if !deadMask[j] {
			liveRemaining++
		}
	}
	if liveRemaining == 0 {
		return nil
	}

	walkCartesian(dims, func(tup []string) bool {
		fm := root.search(tup, dims, 0)
		if fm >= 0 {
			if !wins[fm] {
				wins[fm] = true
				liveRemaining--
			}
		}
		return liveRemaining > 0
	})

	return shadowWarningsForKnownRules(rules, deadMask, wins, nil)
}

func shadowWarningsBudgeted(
	root *ruleTrieNode,
	dims []Dimension,
	rules []Rule,
	deadMask []bool,
	limit int,
) ([]Warning, []bool) {
	wins := make([]bool, len(rules))
	checked := make([]bool, len(rules))
	unchecked := make([]bool, len(rules))
	remaining := limit
	effective := make([][]string, len(dims))
	anyOfBuffers := make([][]string, len(dims))
	indices := make([]int, len(dims))
	tuple := make([]string, len(dims))

	for j, r := range rules {
		if deadMask[j] {
			continue
		}
		product := int64(1)
		feasible := true
		for i, dim := range dims {
			m := matcher{kind: mWildcard}
			if i < len(r.m) {
				m = r.m[i]
			}
			switch m.kind {
			case mWildcard:
				effective[i] = dim.values
			case mExact:
				effective[i] = m.vals
			case mAnyOf:
				values := anyOfBuffers[i][:0]
				for _, value := range m.vals {
					if !dim.contains(value) || containsString(values, value) {
						continue
					}
					values = append(values, value)
				}
				anyOfBuffers[i] = values
				effective[i] = values
			}
			var withinBudget bool
			product, withinBudget = multiplyWithinBudget(product, len(effective[i]), int64(remaining))
			if !withinBudget {
				feasible = false
				break
			}
		}
		if !feasible {
			unchecked[j] = true
			continue
		}

		won := false
		completed := walkValueProduct(effective, indices, tuple, func(tup []string) bool {
			remaining--
			if root.search(tup, dims, 0) == j {
				wins[j] = true
				won = true
				return false
			}
			return true
		})
		if won || completed {
			checked[j] = true
		}
	}

	return shadowWarningsForKnownRules(rules, deadMask, wins, checked), unchecked
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func shadowWarningsForKnownRules(rules []Rule, deadMask, wins, checked []bool) []Warning {
	var out []Warning
	for j := range rules {
		if deadMask[j] {
			continue
		}
		if checked != nil && !checked[j] {
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
// lists until the product is exhausted or fn returns false. fn receives a
// reused buffer; callers must copy if they retain it. The return value reports
// whether the full product was exhausted.
// Empty value lists yield no calls (same as skipping shadow tuples). len(dims)==0
// invokes fn(nil) once.
func walkCartesian(dims []Dimension, fn func(tuple []string) bool) bool {
	for _, dim := range dims {
		if len(dim.values) == 0 {
			return true
		}
	}
	if len(dims) == 0 {
		return fn(nil)
	}
	d := len(dims)
	indices := make([]int, d)
	buf := make([]string, d)
	for {
		for i, dim := range dims {
			buf[i] = dim.values[indices[i]]
		}
		if !fn(buf) {
			return false
		}
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
			return true
		}
	}
}

func walkValueProduct(values [][]string, indices []int, tuple []string, fn func([]string) bool) bool {
	for _, dimValues := range values {
		if len(dimValues) == 0 {
			return true
		}
	}
	if len(values) == 0 {
		return fn(nil)
	}
	clear(indices)
	for {
		for i := range values {
			tuple[i] = values[i][indices[i]]
		}
		if !fn(tuple) {
			return false
		}
		carry := true
		for i := len(values) - 1; i >= 0 && carry; i-- {
			indices[i]++
			if indices[i] < len(values[i]) {
				carry = false
			} else {
				indices[i] = 0
			}
		}
		if carry {
			return true
		}
	}
}
