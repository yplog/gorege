package gorege_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/yplog/gorege"
)

func TestClosestHammingOne(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("role", "u", "v"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Deny("u", "0"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest("u", "0")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected closest")
	}
	if res.Distance != 1 {
		t.Fatalf("Distance=%d want 1", res.Distance)
	}
	if ok, _ := e.Check(res.Conditions...); !ok {
		t.Fatalf("closest tuple not allowed: %#v", res.Conditions)
	}
	if res.Conditions[0] == "u" && res.Conditions[1] == "0" {
		t.Fatal("expected a change from input")
	}
}

func TestClosestInNamed(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("role", "u", "v"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Deny("u", "0"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.ClosestIn("flag", "u", "0")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Value != "1" {
		t.Fatalf("got %#v err=%v", res, err)
	}
}

func TestClosestNoDimensionsNil(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(gorege.WithRules(gorege.Allow()))
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest()
	if err != nil || res != nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

func TestClosestArityMismatch(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Closest()
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestClosestTiebreakRightmost(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithTiebreak(gorege.TiebreakRightmostDim),
		gorege.WithDimensions(
			gorege.Dim("role", "u", "v"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Deny("u", "0"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest("u", "0")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.DimIndex != 1 || res.Value != "1" || res.Distance != 1 {
		t.Fatalf("got %#v err=%v", res, err)
	}
}

func TestClosestInNumericIndices(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("role", "u", "v"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Deny("u", "0"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, dim := range []any{
		int32(1), int64(1), uint(1), uint32(1), uint64(1),
	} {
		res, err := e.ClosestIn(dim, "u", "0")
		if err != nil || res == nil || res.Value != "1" || res.Distance != 1 {
			t.Fatalf("%T(1): res=%v err=%v", dim, res, err)
		}
	}
}

func TestClosestInInvalidSelector(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.Dim("x", "a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.ClosestIn("missing", "a")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = e.ClosestIn("", "a")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = e.ClosestIn(struct{}{}, "a")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = e.ClosestIn(-1, "a")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = e.ClosestIn(uint(1), "a")
	if err == nil {
		t.Fatal("expected error for uint index out of range")
	}
}

func TestClosestNilWhenOnlyCurrentTupleAllowed(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow("a"), gorege.Deny(gorege.Wildcard)),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest("a")
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("expected no closer allowed tuple, got %#v", res)
	}
}

func TestClosestInArityMismatch(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.ClosestIn(0)
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestClosestInZeroDimensionsResolveDimFails(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(gorege.WithRules(gorege.Allow()))
	if err != nil {
		t.Fatal(err)
	}
	// No dimensions: any numeric index is out of range (0 >= len).
	_, err = e.ClosestIn(0)
	if !errors.Is(err, gorege.ErrInvalidDimension) {
		t.Fatalf("got %v", err)
	}
}

func TestClosestInIntOutOfRange(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.ClosestIn(1, "a")
	if !errors.Is(err, gorege.ErrInvalidDimension) {
		t.Fatalf("got %v", err)
	}
}

func TestClosestInNilWhenOnlyCurrentTupleAllowed(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow("a"), gorege.Deny(gorege.Wildcard)),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.ClosestIn(0, "a")
	if err != nil || res != nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

func TestClosestInNilWhenNoAlternativeWorks(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("x", "a", "b"),
		),
		gorege.WithRules(
			gorege.Deny("a"),
			gorege.Deny("b"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.ClosestIn(0, "a")
	if err != nil || res != nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

func TestClosestDistance2(t *testing.T) {
	t.Parallel()
	// Allow only ("b","1"); deny everything else.
	// From ("a","0"), distance-1 candidates ("b","0") and ("a","1") are both
	// denied. The first allowed tuple is ("b","1") at Hamming distance 2.
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("role", "a", "b"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Allow("b", "1"),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest("a", "0")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected a closest result at distance 2")
	}
	if res.Distance != 2 {
		t.Fatalf("Distance=%d, want 2", res.Distance)
	}
	ok, err := e.Check(res.Conditions...)
	if err != nil || !ok {
		t.Fatalf("closest conditions not allowed: %v err=%v", res.Conditions, err)
	}
}

// TestClosestResultDynamic exercises the len(input) > maxDims (16) code path
// in both searchSubset (non-pool allocation) and buildClosestResultDynamic.
func TestClosestResultDynamic(t *testing.T) {
	t.Parallel()
	const n = 17 // one above the pool threshold (maxDims = 16)
	dims := make([]gorege.Dimension, n)
	for i := range dims {
		dims[i] = gorege.Dim(fmt.Sprintf("d%d", i), "a", "b")
	}
	// Allow when d0=="b", deny everything else.
	allowParts := make([]any, n)
	allowParts[0] = "b"
	for i := 1; i < n; i++ {
		allowParts[i] = gorege.Wildcard
	}
	denyParts := make([]any, n)
	for i := range denyParts {
		denyParts[i] = gorege.Wildcard
	}
	e, _, err := gorege.New(
		gorege.WithDimensions(dims...),
		gorege.WithRules(
			gorege.Allow(allowParts...),
			gorege.Deny(denyParts...),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := make([]string, n)
	for i := range input {
		input[i] = "a"
	}
	res, err := e.Closest(input...)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected closest result for 17-dim engine")
	}
	if res.Distance != 1 {
		t.Fatalf("Distance=%d, want 1", res.Distance)
	}
	if res.DimIndex != 0 || res.Value != "b" {
		t.Fatalf("DimIndex=%d Value=%q, want DimIndex=0 Value=b", res.DimIndex, res.Value)
	}
	ok, err := e.Check(res.Conditions...)
	if err != nil || !ok {
		t.Fatalf("closest conditions not allowed: %v err=%v", res.Conditions, err)
	}
}

// TestClosestInLargeDims exercises the non-pool allocation path (nd > maxDims)
// inside ClosestIn.
func TestClosestInLargeDims(t *testing.T) {
	t.Parallel()
	const n = 17
	dims := make([]gorege.Dimension, n)
	for i := range dims {
		dims[i] = gorege.Dim(fmt.Sprintf("d%d", i), "a", "b")
	}
	allowParts := make([]any, n)
	allowParts[0] = "b"
	for i := 1; i < n; i++ {
		allowParts[i] = gorege.Wildcard
	}
	denyParts := make([]any, n)
	for i := range denyParts {
		denyParts[i] = gorege.Wildcard
	}
	e, _, err := gorege.New(
		gorege.WithDimensions(dims...),
		gorege.WithRules(
			gorege.Allow(allowParts...),
			gorege.Deny(denyParts...),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	input := make([]string, n)
	for i := range input {
		input[i] = "a"
	}
	res, err := e.ClosestIn(0, input...)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected ClosestIn result for 17-dim engine")
	}
	if res.Distance != 1 || res.Value != "b" {
		t.Fatalf("Distance=%d Value=%q, want 1 and b", res.Distance, res.Value)
	}
}

// TestClosestTiebreakRightmostDistance2 exercises the k>1 recursive path of
// trySubsetsRightmost by requiring a Hamming distance of 2 with the rightmost
// tiebreak strategy.
func TestClosestTiebreakRightmostDistance2(t *testing.T) {
	t.Parallel()
	// Allow only ("b","1"); all other tuples are denied.
	// From ("a","0"), no distance-1 candidate is allowed, so Closest must
	// recurse to k=2 inside trySubsetsRightmost.
	e, _, err := gorege.New(
		gorege.WithTiebreak(gorege.TiebreakRightmostDim),
		gorege.WithDimensions(
			gorege.Dim("role", "a", "b"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Allow("b", "1"),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Closest("a", "0")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected closest result at distance 2 with rightmost tiebreak")
	}
	if res.Distance != 2 {
		t.Fatalf("Distance=%d, want 2", res.Distance)
	}
	// With TiebreakRightmostDim, the primary reported dimension should be the
	// rightmost changed position (index 1 = "flag").
	if res.DimIndex != 1 {
		t.Fatalf("DimIndex=%d, want 1 (rightmost changed)", res.DimIndex)
	}
	ok, err := e.Check(res.Conditions...)
	if err != nil || !ok {
		t.Fatalf("closest conditions not allowed: %v err=%v", res.Conditions, err)
	}
}

// TestResolveDimNegativeTypedInts checks that signed negative int32/int64 and
// out-of-range unsigned types return ErrInvalidDimension.
func TestResolveDimNegativeTypedInts(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.Dim("x", "a", "b")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	invalid := []any{int32(-1), int64(-1), uint32(999), uint64(999)}
	for _, dim := range invalid {
		_, err = e.ClosestIn(dim, "a")
		if !errors.Is(err, gorege.ErrInvalidDimension) {
			t.Errorf("%T(%v): expected ErrInvalidDimension, got %v", dim, dim, err)
		}
	}
}
