package gorege

import (
	"errors"
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
	if m.kind != mAnyOf || len(m.anyof) != 2 || m.anyof[0] != "a" || m.anyof[1] != "b" {
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
	m := matcher{kind: mExact, exact: "x"}
	dim := DimValues("a", "b")
	if m.matches("x", dim, true) {
		t.Fatal("exact value not in dimension should not match when dimKnown")
	}
}

func TestMatcherMatchesDefensiveAnyOfUnknownValue(t *testing.T) {
	t.Parallel()
	m := matcher{kind: mAnyOf, anyof: []string{"x"}}
	dim := DimValues("a", "b")
	if m.matches("x", dim, true) {
		t.Fatal("anyOf value not in dimension should not match when dimKnown")
	}
}

func TestMatcherMatchesInvalidKind(t *testing.T) {
	t.Parallel()
	m := matcher{kind: matcherKind(255)}
	dim := DimValues("a")
	if m.matches("a", dim, true) {
		t.Fatal("invalid kind should not match")
	}
}

func TestMatcherWildcardWhenDimNotKnownForMatch(t *testing.T) {
	t.Parallel()
	m := matcher{kind: mWildcard}
	var dim Dimension
	if !m.matches("any-input", dim, false) {
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

func TestCombinationsZeroZeroRightmostUsesMaxOrZero(t *testing.T) {
	t.Parallel()
	out := combinations(0, 0, TiebreakRightmostDim)
	if len(out) != 1 || len(out[0]) != 0 {
		t.Fatalf("got %#v", out)
	}
}

func TestCombinationsInvalidKReturnsNil(t *testing.T) {
	t.Parallel()
	if combinations(2, 3, TiebreakLeftmostDim) != nil {
		t.Fatal("k>n should return nil")
	}
	if combinations(2, -1, TiebreakLeftmostDim) != nil {
		t.Fatal("k<0 should return nil")
	}
}

func TestCombinationsRightmostNonEmptySubsetsHitMaxOrZero(t *testing.T) {
	t.Parallel()
	out := combinations(3, 1, TiebreakRightmostDim)
	if len(out) != 3 {
		t.Fatalf("got %d subsets", len(out))
	}
}
