package gorege_test

import (
	"errors"
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
	if res == nil || res.DimIndex != 1 || res.Value != "1" {
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
		if err != nil || res == nil || res.Value != "1" {
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
