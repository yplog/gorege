package gorege

import (
	"slices"
	"sync"
)

// maxDims is the largest tuple length for which Closest reuses pooled scratch
// slices; larger arity allocates per searchSubset call.
const maxDims = 16

var curPool = sync.Pool{
	New: func() any {
		s := make([]string, maxDims)
		return &s
	},
}

// nextCombo advances combo[:k] to the next k-combination of {0..n-1} in
// lexicographic (ascending index) order. Returns false when combo is already
// the last combination [n-k, n-k+1, ..., n-1].
// combo must be initialised to [0, 1, ..., k-1] before the first call.
func nextCombo(combo []int, n int) bool {
	k := len(combo)
	if k == 0 {
		return false
	}
	i := k - 1
	for i >= 0 && combo[i] == n-k+i {
		i--
	}
	if i < 0 {
		return false
	}
	combo[i]++
	for j := i + 1; j < k; j++ {
		combo[j] = combo[j-1] + 1
	}
	return true
}

// Closest searches for a nearest allowed tuple using breadth-first Hamming
// distance: distance 1, then 2, … until a candidate passes [Engine.Check].
// Tiebreak controls subset and reporting order; the returned [ClosestResult]
// highlights one primary changed dimension consistent with that strategy.
func (e *Engine) Closest(values ...string) (*ClosestResult, error) {
	if len(values) != len(e.dims) {
		return nil, ErrArityMismatch
	}
	if len(e.dims) == 0 {
		return nil, nil
	}
	n := len(e.dims)
	var comboStack [maxDims]int
	var combo []int
	if n <= maxDims {
		combo = comboStack[:n]
	} else {
		combo = make([]int, n)
	}
	for k := 1; k <= n; k++ {
		if e.tiebreak == TiebreakRightmostDim {
			if res := e.trySubsetsRightmost(values, n, k, 0, k, combo); res != nil {
				return res, nil
			}
		} else {
			for i := range k {
				combo[i] = i
			}
			for {
				if res := e.searchSubset(values, combo[:k]); res != nil {
					return res, nil
				}
				if !nextCombo(combo[:k], n) {
					break
				}
			}
		}
	}
	return nil, nil
}

// trySubsetsRightmost visits k-subsets of {0..universe-1} into buf[start:start+k],
// then tests buf[:kFull] in the same order as the former
// combinations(universe, kFull, TiebreakRightmostDim). start and kFull are fixed
// across recursion so the leaf always calls searchSubset with the full kFull-length prefix.
func (e *Engine) trySubsetsRightmost(values []string, universe, k, start, kFull int, buf []int) *ClosestResult {
	if k == 1 {
		for v := universe - 1; v >= 0; v-- {
			buf[start] = v
			if res := e.searchSubset(values, buf[:kFull]); res != nil {
				return res
			}
		}
		return nil
	}
	for last := universe - 1; last >= k-1; last-- {
		buf[start+k-1] = last
		if res := e.trySubsetsRightmost(values, last, k-1, start, kFull, buf); res != nil {
			return res
		}
	}
	return nil
}

// ClosestIn restricts the search to a single dimension. dim may be a
// dimension index (int, int32, int64, uint, uint32, uint64) or a non-empty
// dimension name (string). Returns nil when no value in that dimension yields
// an allowed tuple.
func (e *Engine) ClosestIn(dim any, values ...string) (*ClosestResult, error) {
	if len(values) != len(e.dims) {
		return nil, ErrArityMismatch
	}
	di, err := e.resolveDim(dim)
	if err != nil {
		return nil, err
	}
	if len(e.dims) == 0 {
		return nil, nil
	}
	input := values
	d := e.dims[di]
	cand := append([]string(nil), input...)
	for _, v := range d.values {
		if v == input[di] {
			continue
		}
		cand[di] = v
		ok, err := e.Check(cand...)
		if err != nil {
			return nil, err
		}
		if !ok {
			cand[di] = input[di]
			continue
		}
		return &ClosestResult{
			Conditions: append([]string(nil), cand...),
			Distance:   hammingDistance(input, cand),
			DimIndex:   di,
			DimName:    d.name,
			Value:      v,
		}, nil
	}
	return nil, nil
}

func (e *Engine) resolveDim(dim any) (int, error) {
	switch x := dim.(type) {
	case int:
		if x < 0 || x >= len(e.dims) {
			return 0, ErrInvalidDimension
		}
		return x, nil
	case int32:
		if int(x) < 0 || int(x) >= len(e.dims) {
			return 0, ErrInvalidDimension
		}
		return int(x), nil
	case int64:
		if int(x) < 0 || int(x) >= len(e.dims) {
			return 0, ErrInvalidDimension
		}
		return int(x), nil
	case uint:
		if uint64(x) >= uint64(len(e.dims)) {
			return 0, ErrInvalidDimension
		}
		return int(x), nil
	case uint32:
		if uint64(x) >= uint64(len(e.dims)) {
			return 0, ErrInvalidDimension
		}
		return int(x), nil
	case uint64:
		if x >= uint64(len(e.dims)) {
			return 0, ErrInvalidDimension
		}
		return int(x), nil
	case string:
		if x == "" {
			return 0, ErrInvalidDimension
		}
		for i, d := range e.dims {
			if d.name == x {
				return i, nil
			}
		}
		return 0, ErrInvalidDimension
	default:
		return 0, ErrInvalidDimension
	}
}

// searchSubsetDFS traverses the subset by DFS, mutating cur in-place and
// restoring each slot on backtrack. Engine is read-only; cur is private to the
// calling searchSubset frame.
func searchSubsetDFS(e *Engine, input, cur []string, subset []int, pos int) *ClosestResult {
	if pos == len(subset) {
		ok, err := e.Check(cur...)
		if err != nil || !ok {
			return nil
		}
		return buildClosestResult(e, input, cur, subset)
	}
	di := subset[pos]
	dim := e.dims[di]
	for _, v := range dim.values {
		if v == input[di] {
			continue
		}
		cur[di] = v
		if res := searchSubsetDFS(e, input, cur, subset, pos+1); res != nil {
			return res
		}
		cur[di] = input[di]
	}
	return nil
}

func (e *Engine) searchSubset(input []string, subset []int) *ClosestResult {
	d := len(input)
	var ptr *[]string
	var cur []string
	if d <= maxDims {
		ptr = curPool.Get().(*[]string)
		cur = (*ptr)[:d]
		copy(cur, input)
	} else {
		cur = append([]string(nil), input...)
	}
	res := searchSubsetDFS(e, input, cur, subset, 0)
	if ptr != nil {
		*ptr = (*ptr)[:maxDims]
		curPool.Put(ptr)
	}
	return res
}

func buildClosestResult(e *Engine, input, candidate []string, subset []int) *ClosestResult {
	if len(input) > maxDims {
		return buildClosestResultDynamic(e, input, candidate, subset)
	}
	var seen [maxDims]bool
	var diffs [maxDims]int
	ndiffs := 0
	for _, i := range subset {
		seen[i] = true
	}
	for i := range input {
		if seen[i] && candidate[i] != input[i] {
			diffs[ndiffs] = i
			ndiffs++
		}
	}
	if ndiffs == 0 {
		return nil
	}
	primary := pickPrimaryDim(diffs[:ndiffs], e.tiebreak)
	d := e.dims[primary]
	return &ClosestResult{
		Conditions: append([]string(nil), candidate...),
		// DFS assigns cur[di]=v where v!=input[di] for every di in subset,
		// so all subset positions differ — distance equals ndiffs=len(subset).
		Distance: ndiffs,
		DimIndex: primary,
		DimName:  d.name,
		Value:    candidate[primary],
	}
}

func buildClosestResultDynamic(e *Engine, input, candidate []string, subset []int) *ClosestResult {
	seen := make([]bool, len(input))
	var diffs []int
	for _, i := range subset {
		seen[i] = true
	}
	for i := range input {
		if seen[i] && candidate[i] != input[i] {
			diffs = append(diffs, i)
		}
	}
	if len(diffs) == 0 {
		return nil
	}
	primary := pickPrimaryDim(diffs, e.tiebreak)
	d := e.dims[primary]
	return &ClosestResult{
		Conditions: append([]string(nil), candidate...),
		Distance:   len(diffs),
		DimIndex:   primary,
		DimName:    d.name,
		Value:      candidate[primary],
	}
}

func hammingDistance(a, b []string) int {
	if len(a) != len(b) {
		panic("gorege: hammingDistance: length mismatch")
	}
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

func pickPrimaryDim(diffs []int, tb TiebreakStrategy) int {
	switch tb {
	case TiebreakRightmostDim:
		return slices.Max(diffs)
	default:
		return slices.Min(diffs)
	}
}
