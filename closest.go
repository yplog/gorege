package gorege

import "slices"

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
	for k := 1; k <= n; k++ {
		subs := combinations(n, k, e.tiebreak)
		for _, subset := range subs {
			if res := e.searchSubset(values, subset); res != nil {
				return res, nil
			}
		}
	}
	return nil, nil
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
	for _, v := range d.values {
		if v == input[di] {
			continue
		}
		cand := append([]string(nil), input...)
		cand[di] = v
		ok, err := e.Check(cand...)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		return &ClosestResult{
			Conditions: cand,
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

func (e *Engine) searchSubset(input []string, subset []int) *ClosestResult {
	cur := append([]string(nil), input...)
	var dfs func(pos int) *ClosestResult
	dfs = func(pos int) *ClosestResult {
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
			prev := cur[di]
			cur[di] = v
			if res := dfs(pos + 1); res != nil {
				return res
			}
			cur[di] = prev
		}
		return nil
	}
	return dfs(0)
}

func buildClosestResult(e *Engine, input, candidate []string, subset []int) *ClosestResult {
	var diffs []int
	seen := make([]bool, len(input))
	for _, i := range subset {
		seen[i] = true
	}
	for i := range input {
		if !seen[i] {
			continue
		}
		if candidate[i] != input[i] {
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
		DimIndex:   primary,
		DimName:    d.name,
		Value:      candidate[primary],
	}
}

func pickPrimaryDim(diffs []int, tb TiebreakStrategy) int {
	switch tb {
	case TiebreakRightmostDim:
		return slices.Max(diffs)
	default:
		return slices.Min(diffs)
	}
}

func combinations(n, k int, tb TiebreakStrategy) [][]int {
	if k < 0 || k > n {
		return nil
	}
	var out [][]int
	var gen func(start, left int, cur []int)
	gen = func(start, left int, cur []int) {
		if left == 0 {
			cp := append([]int(nil), cur...)
			out = append(out, cp)
			return
		}
		for i := start; i <= n-left; i++ {
			gen(i+1, left-1, append(cur, i))
		}
	}
	gen(0, k, nil)
	switch tb {
	case TiebreakRightmostDim:
		slices.SortFunc(out, func(a, b []int) int {
			ma, mb := maxOrZero(a), maxOrZero(b)
			if ma != mb {
				return mb - ma
			}
			for i := len(a) - 1; i >= 0; i-- {
				if a[i] != b[i] {
					return b[i] - a[i]
				}
			}
			return 0
		})
	default:
		// TiebreakLeftmostDim and TiebreakDeclOrder: lexicographic on indices.
		slices.SortFunc(out, func(a, b []int) int {
			for i := 0; i < len(a) && i < len(b); i++ {
				if a[i] != b[i] {
					return a[i] - b[i]
				}
			}
			return len(a) - len(b)
		})
	}
	return out
}

// maxOrZero uses [slices.Max] for non-empty slices; empty returns 0 so sort
// comparators never panic (e.g. k==0 combinations).
func maxOrZero(s []int) int {
	if len(s) == 0 {
		return 0
	}
	return slices.Max(s)
}
