package gorege

import "strconv"

// TiebreakStrategy orders equally-distant candidates in [Engine.Closest].
type TiebreakStrategy int

const (
	// TiebreakLeftmostDim prefers the smallest index among changed dimensions
	// when reporting [ClosestResult], and tries subset combinations in
	// increasing lexicographic index order.
	TiebreakLeftmostDim TiebreakStrategy = iota
	// TiebreakRightmostDim prefers the largest changed dimension index and tries
	// subsets with larger indices first.
	TiebreakRightmostDim
	// TiebreakDeclOrder matches [TiebreakLeftmostDim] for this implementation
	// (dimensions are already in declaration order).
	TiebreakDeclOrder
)

// ClosestResult is returned by [Engine.Closest] and [Engine.ClosestIn] when an
// allowed tuple exists. DimIndex names the primary dimension reported for that
// hit (see tiebreak strategy). If no allowed combination exists, both methods
// return a nil pointer.
type ClosestResult struct {
	Conditions []string
	// Distance is the Hamming distance from the input tuple to Conditions (number
	// of dimensions whose value differs).
	Distance int
	DimIndex int
	DimName  string
	Value    string
}

// Explanation is the outcome of [Engine.Explain] for a full input tuple.
type Explanation struct {
	Allowed   bool
	RuleIndex int
	RuleName  string
	Action    Action
	Matched   bool
}

// String implements [fmt.Stringer] for [Action].
func (a Action) String() string {
	if a == ActionAllow {
		return "ALLOW"
	}
	return "DENY"
}

// Action returns the rule's action (ALLOW or DENY).
func (r Rule) Action() Action {
	return r.act
}

// String returns a stable textual form for [Warning].
func (w Warning) String() string {
	return w.Message
}

// GoString satisfies [fmt.GoStringer] for [TiebreakStrategy].
func (t TiebreakStrategy) GoString() string {
	switch t {
	case TiebreakLeftmostDim:
		return "TiebreakLeftmostDim"
	case TiebreakRightmostDim:
		return "TiebreakRightmostDim"
	case TiebreakDeclOrder:
		return "TiebreakDeclOrder"
	default:
		return "TiebreakStrategy(" + strconv.Itoa(int(t)) + ")"
	}
}
