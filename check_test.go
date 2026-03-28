package gorege_test

import (
	"errors"
	"testing"

	"github.com/yplog/gorege"
)

func TestQuickStart(t *testing.T) {
	t.Parallel()
	e, warnings, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("membership", "Gold member", "Regular member", "Guest"),
			gorege.Dim("day", "Mon", "Tue", "Wed", "Thu", "Fri"),
			gorege.Dim("facility", "Swimming pool", "Gym", "Sauna"),
		),
		gorege.WithRules(
			gorege.Allow("Gold member", gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("Guest", gorege.AnyOf("Mon", "Tue"), "Sauna"),
			gorege.Allow(gorege.AnyOf("Guest", "Regular member"), gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	ok, err := e.Check("Guest", "Mon", "Sauna")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected deny for Guest Mon Sauna")
	}

	ok, err = e.Check("Guest", "Wed", "Sauna")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected allow for Guest Wed Sauna")
	}
}

func TestCheckArity(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Check()
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("expected ErrArityMismatch, got %v", err)
	}
	_, err = e.Check("a", "b")
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("expected ErrArityMismatch, got %v", err)
	}
}

func TestWildcardRejectsUnknownInput(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow(gorege.Wildcard)),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("zzz")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("wildcard should not match value outside dimension")
	}
}

func TestNoDimensionsCatchAllAllow(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithRules(gorege.Allow()),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected allow with empty tuple")
	}
}

func TestRuleTooWide(t *testing.T) {
	t.Parallel()
	_, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a", "b")),
	)
	if !errors.Is(err, gorege.ErrRuleTooWide) {
		t.Fatalf("expected ErrRuleTooWide, got %v", err)
	}
}

func TestShortRuleImplicitWildcardTail(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.DimValues("x", "y"),
			gorege.DimValues("1", "2"),
		),
		gorege.WithRules(
			gorege.Allow("x"),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("x", "2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("short rule should wildcard trailing dimension")
	}
}

func TestPartialCheckTrailingUnconstrained(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.DimValues("Gold", "Guest"),
			gorege.DimValues("Mon", "Tue"),
		),
		gorege.WithRules(
			gorege.Deny("Guest", "Mon"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Deny("Guest","Mon") does not match PartialCheck("Guest"): dim1 absent fails a non-wildcard DENY slot.
	// Next rule Allow(*,*) matches, so the result is true.
	ok, err := e.PartialCheck("Guest")
	if err != nil || !ok {
		t.Fatalf("Guest: ok=%v err=%v", ok, err)
	}
	ok, err = e.PartialCheck("Gold")
	if err != nil || !ok {
		t.Fatalf("Gold: ok=%v err=%v", ok, err)
	}

	e2, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.DimValues("Gold", "Guest"),
			gorege.DimValues("Mon", "Tue"),
		),
		gorege.WithRules(
			gorege.Deny("Guest", "Mon"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = e2.PartialCheck("Guest")
	if err != nil || ok {
		t.Fatalf("expected false when DENY cannot bind trailing dim and no other rule matches: ok=%v err=%v", ok, err)
	}
}

func TestUnknownValueInRuleRejectedAtNew(t *testing.T) {
	t.Parallel()
	_, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow("c")),
	)
	if !errors.Is(err, gorege.ErrUnknownDimensionValue) {
		t.Fatalf("expected ErrUnknownDimensionValue, got %v", err)
	}
}

func TestPartialCheckTooManyValues(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.PartialCheck("a", "b")
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("expected ErrArityMismatch, got ok=%v err=%v", ok, err)
	}
	if ok {
		t.Fatal("expected ok false with arity error")
	}
}

func TestPartialCheckZeroPrefix(t *testing.T) {
	t.Parallel()

	// Engine has ALLOW rules → zero-prefix should return true
	e1, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e1.PartialCheck()
	if err != nil || !ok {
		t.Fatalf("zero prefix with ALLOW rule: ok=%v err=%v", ok, err)
	}

	// Engine has only DENY rules → zero-prefix returns false
	e2, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Deny("a"), gorege.Deny("b")),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = e2.PartialCheck()
	if err != nil || ok {
		t.Fatalf("zero prefix with only DENY rules: ok=%v err=%v", ok, err)
	}
}
