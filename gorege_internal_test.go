package gorege

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestNewPropagatesOptionError(t *testing.T) {
	t.Parallel()
	bad := Option(func(*engineConfig) error {
		return errors.New("opt failed")
	})
	_, _, err := New(bad)
	if err == nil || err.Error() != "opt failed" {
		t.Fatalf("got %v", err)
	}
}

func TestMatcherFromSlotStringSlice(t *testing.T) {
	t.Parallel()
	m, err := matcherFromSlot([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if m.kind != mAnyOf || len(m.vals) != 2 || m.vals[0] != "a" || m.vals[1] != "b" {
		t.Fatalf("%+v", m)
	}
}

func TestMatcherFromSlotInvalidType(t *testing.T) {
	t.Parallel()
	_, err := matcherFromSlot(3.14)
	if err == nil || !strings.Contains(err.Error(), "want string") {
		t.Fatalf("got %v", err)
	}
}

func TestMatcherMatchesDefensiveExactUnknownValue(t *testing.T) {
	t.Parallel()
	m := matcher{kind: mExact, vals: []string{"x"}}
	dim := DimValues("a", "b")
	if m.matches("x", dim) {
		t.Fatal("exact value not in dimension should not match when dimKnown")
	}
}

func TestMatcherMatchesDefensiveAnyOfUnknownValue(t *testing.T) {
	t.Parallel()
	m := matcher{kind: mAnyOf, vals: []string{"x"}}
	dim := DimValues("a", "b")
	if m.matches("x", dim) {
		t.Fatal("anyOf value not in dimension should not match when dimKnown")
	}
}

func TestMatcherMatchesInvalidKind(t *testing.T) {
	t.Parallel()
	m := matcher{kind: matcherKind(255)}
	dim := DimValues("a")
	if m.matches("a", dim) {
		t.Fatal("invalid kind should not match")
	}
}

func TestMatcherWildcardWhenDimNotKnownForMatch(t *testing.T) {
	t.Parallel()
	m := matcher{kind: mWildcard}
	var dim Dimension
	if !m.matches("any-input", dim) {
		t.Fatal("wildcard with dimKnown false should match any input")
	}
}

func TestTupleCountEmptyDimensionValues(t *testing.T) {
	t.Parallel()
	if n := tupleCount([]Dimension{DimValues()}, 100); n != 0 {
		t.Fatalf("got %d", n)
	}
}

func TestDimensionContainsEmptyIndex(t *testing.T) {
	t.Parallel()
	d := DimValues()
	if d.contains("x") {
		t.Fatal("empty value list ⇒ empty index; contains must be false")
	}
}

func TestTupleCountEarlyExitOverLimit(t *testing.T) {
	t.Parallel()
	dims := []Dimension{
		DimValues("a", "b", "c", "d", "e"),
		DimValues("1", "2", "3", "4", "5"),
		DimValues("x", "y", "z", "w", "v"),
	}
	n := tupleCount(dims, 100)
	if n != 125 {
		t.Fatalf("expected product 125 before stop, got %d", n)
	}
}

func TestTupleCountNoLimitFullProduct(t *testing.T) {
	t.Parallel()
	n := tupleCount([]Dimension{
		DimValues("a", "b"),
		DimValues("1", "2", "3"),
	}, 0)
	if n != 6 {
		t.Fatalf("got %d", n)
	}
}

// combinationsLexOracle returns all k-combinations of {0..n-1} in ascending
// lexicographic index order (matches former combinations(..., TiebreakLeftmostDim)).
func combinationsLexOracle(n, k int) [][]int {
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
	slices.SortFunc(out, func(a, b []int) int {
		for i := 0; i < len(a) && i < len(b); i++ {
			if a[i] != b[i] {
				return a[i] - b[i]
			}
		}
		return len(a) - len(b)
	})
	return out
}

// prevCombo is the lexicographic predecessor of k-combinations; used only in tests
// (TiebreakRightmostDim uses [Engine.trySubsetsRightmost] in production).
func prevCombo(combo []int, n int) bool {
	k := len(combo)
	if k == 0 {
		return false
	}
	i := k - 1
	for i >= 0 {
		floor := -1
		if i > 0 {
			floor = combo[i-1]
		}
		if combo[i] > floor+1 {
			break
		}
		i--
	}
	if i < 0 {
		return false
	}
	combo[i]--
	for j := i + 1; j < k; j++ {
		combo[j] = n - k + j
	}
	return true
}

func maxOrZero(s []int) int {
	if len(s) == 0 {
		return 0
	}
	return slices.Max(s)
}

// combinationsRightmostOracle matches former combinations(..., TiebreakRightmostDim).
func combinationsRightmostOracle(n, k int) [][]int {
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
	return out
}

// rightmostWalkOrder mirrors [Engine.trySubsetsRightmost] subset order.
func rightmostWalkOrder(n, k int) [][]int {
	var out [][]int
	buf := make([]int, k)
	var walk func(universe, kk, start, kFull int)
	walk = func(universe, kk, start, kFull int) {
		if kk == 1 {
			for v := universe - 1; v >= 0; v-- {
				buf[start] = v
				out = append(out, append([]int(nil), buf[:kFull]...))
			}
			return
		}
		for last := universe - 1; last >= kk-1; last-- {
			buf[start+kk-1] = last
			walk(last, kk-1, start, kFull)
		}
	}
	walk(n, k, 0, k)
	return out
}

func TestRightmostWalkMatchesOracle(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 6; n++ {
		for k := 1; k <= n; k++ {
			got := rightmostWalkOrder(n, k)
			want := combinationsRightmostOracle(n, k)
			if !slices.EqualFunc(got, want, slices.Equal) {
				t.Errorf("n=%d k=%d:\ngot  %v\nwant %v", n, k, got, want)
			}
		}
	}
}

func TestCombinationsLexOracleInvalidK(t *testing.T) {
	t.Parallel()
	if combinationsLexOracle(2, 3) != nil {
		t.Fatal("k>n should return nil")
	}
	if combinationsLexOracle(2, -1) != nil {
		t.Fatal("k<0 should return nil")
	}
}

func TestCombinationsLexOracleZeroZero(t *testing.T) {
	t.Parallel()
	out := combinationsLexOracle(0, 0)
	if len(out) != 1 || len(out[0]) != 0 {
		t.Fatalf("got %#v", out)
	}
}

func TestNextComboEnumeratesAll(t *testing.T) {
	t.Parallel()
	for n := range 6 {
		for k := range n + 1 {
			if k == 0 {
				if nextCombo([]int{}, n) {
					t.Errorf("n=%d k=0: expected false", n)
				}
				continue
			}
			combo := make([]int, k)
			for i := range k {
				combo[i] = i
			}
			var got [][]int
			for {
				got = append(got, append([]int(nil), combo...))
				if !nextCombo(combo, n) {
					break
				}
			}
			want := combinationsLexOracle(n, k)
			if !slices.EqualFunc(got, want, slices.Equal) {
				t.Errorf("n=%d k=%d:\ngot  %v\nwant %v", n, k, got, want)
			}
		}
	}
}

func TestPrevComboReverseOfNext(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 6; n++ {
		for k := 1; k <= n; k++ {
			combo := make([]int, k)
			for i := range k {
				combo[i] = i
			}
			var fwd [][]int
			for {
				fwd = append(fwd, append([]int(nil), combo...))
				if !nextCombo(combo, n) {
					break
				}
			}
			for i := range k {
				combo[i] = n - k + i
			}
			var rev [][]int
			for {
				rev = append(rev, append([]int(nil), combo...))
				if !prevCombo(combo, n) {
					break
				}
			}
			if len(fwd) != len(rev) {
				t.Errorf("n=%d k=%d: len fwd=%d rev=%d", n, k, len(fwd), len(rev))
				continue
			}
			for i, j := 0, len(fwd)-1; i < len(fwd); i, j = i+1, j-1 {
				if !slices.Equal(fwd[i], rev[j]) {
					t.Errorf("n=%d k=%d pos %d: fwd=%v rev=%v", n, k, i, fwd[i], rev[j])
				}
			}
		}
	}
}

func TestPrevComboKZero(t *testing.T) {
	t.Parallel()
	if prevCombo([]int{}, 3) {
		t.Error("k=0: expected false")
	}
}
