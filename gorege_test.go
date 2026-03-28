package gorege_test

import (
	"testing"

	"github.com/yplog/gorege"
)

func TestNewNilOptionSkipped(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(nil, gorege.WithRules(gorege.Allow()))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check()
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestWithTiebreakDeclOrder(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithTiebreak(gorege.TiebreakDeclOrder),
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
	if err != nil || res == nil || res.Distance != 1 {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

func TestEngineRulesCopy(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow("a"),
			gorege.Deny("b"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	rules := e.Rules()
	if len(rules) != 2 {
		t.Fatal(len(rules))
	}
	if rules[0].Action() != gorege.ActionAllow || rules[0].Name != "" {
		t.Fatalf("%+v", rules[0])
	}
	if rules[1].Action() != gorege.ActionDeny {
		t.Fatalf("%+v", rules[1])
	}
	rules[0] = gorege.Deny("b")
	rules2 := e.Rules()
	if rules2[0].Action() != gorege.ActionAllow {
		t.Fatal("mutating returned slice must not affect engine")
	}
}

func TestEngineDimensionsCopy(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	d := e.Dimensions()
	if len(d) != 1 {
		t.Fatal(len(d))
	}
	d[0] = gorege.DimValues("x")
	d2 := e.Dimensions()
	if len(d2[0].Values()) == 0 || d2[0].Values()[0] != "a" {
		t.Fatal("mutating copy should not affect engine")
	}
}

func TestAnalysisLimitExceeded(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(100),
		gorege.WithDimensions(
			gorege.DimValues("a", "b", "c", "d", "e"),
			gorege.DimValues("1", "2", "3", "4", "5"),
			gorege.DimValues("x", "y", "z", "w", "v"),
		),
		gorege.WithRules(gorege.Allow(gorege.Wildcard, gorege.Wildcard, gorege.Wildcard)),
	)
	if err != nil {
		t.Fatal(err)
	}
	var hasLimit bool
	for _, w := range warnings {
		if w.Kind == gorege.WarningKindAnalysisLimitExceeded {
			hasLimit = true
		}
	}
	if !hasLimit {
		t.Fatalf("expected AnalysisLimitExceeded warning, got %v", warnings)
	}
}

func TestDeadRuleDetectedEvenWhenLimitExceeded(t *testing.T) {
	t.Parallel()
	// Product 3×3 > limit 1; empty AnyOf validates but never matches (dead).
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(
			gorege.DimValues("a", "b", "c"),
			gorege.DimValues("x", "y", "z"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.AnyOf()),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	var hasDead, hasLimit bool
	for _, w := range warnings {
		switch w.Kind {
		case gorege.WarningKindDead:
			hasDead = true
		case gorege.WarningKindAnalysisLimitExceeded:
			hasLimit = true
		}
	}
	if !hasDead {
		t.Fatalf("expected dead warning even when limit exceeded, got %v", warnings)
	}
	if !hasLimit {
		t.Fatalf("expected limit warning, got %v", warnings)
	}
}

func TestAnalysisLimitNegativeSkipsAnalysis(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(-1),
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard),
			gorege.Deny("a"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestAnalysisLimitProductEqualToLimitStillAnalyzes(t *testing.T) {
	t.Parallel()
	axis := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(100),
		gorege.WithDimensions(
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
		),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("0", "0"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		if w.Kind == gorege.WarningKindAnalysisLimitExceeded {
			t.Fatalf("100 tuples should not exceed limit 100: %v", warnings)
		}
	}
	found := false
	for _, w := range warnings {
		if w.Kind == gorege.WarningKindShadowed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected shadowed warning, got %v", warnings)
	}
}

func TestAnalysisLimitZeroUsesDefault(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(0),
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard),
			gorege.Deny("a"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindShadowed {
		t.Fatalf("expected shadowed warning, got %v", warnings)
	}
}
