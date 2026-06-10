package gorege_test

import (
	"strings"
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

func TestNewCopiesOptionInputs(t *testing.T) {
	t.Parallel()
	dims := []gorege.Dimension{gorege.DimValues("a", "b")}
	rules := []gorege.Rule{gorege.Allow("a")}
	e, _, err := gorege.New(
		gorege.WithDimensions(dims...),
		gorege.WithRules(rules...),
	)
	if err != nil {
		t.Fatal(err)
	}

	dims[0] = gorege.DimValues("x")
	rules[0] = gorege.Deny("a")

	ok, err := e.Check("a")
	if err != nil || !ok {
		t.Fatalf("caller mutation affected engine: ok=%v err=%v", ok, err)
	}
	got := e.Dimensions()
	if values := got[0].Values(); len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("caller mutation affected dimensions: %v", values)
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
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.Wildcard, gorege.Wildcard),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard, gorege.Wildcard),
		),
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
	if got := warnings[len(warnings)-1].Message; !strings.Contains(got, "rule 1") ||
		!strings.Contains(got, "effective product exceeds the remaining tuple budget") {
		t.Fatalf("unexpected limit warning: %q", got)
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
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
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

func TestAnalysisBudgetExactlyEnough(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow("a", "x"),
			gorege.Deny("b", "y"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("exactly sufficient budget produced warnings: %v", warnings)
	}
}

func TestAnalysisBudgetLastTupleCompletesShadowDecision(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow("a", "x"),
			gorege.Deny("a", "x"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindShadowed {
		t.Fatalf("final budget tuple should complete the shadow decision: %v", warnings)
	}
}

func TestAnalysisBudgetOneShortLabelsUncheckedRule(t *testing.T) {
	t.Parallel()
	second := gorege.Deny("b", "y")
	second.Name = "second"
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow("a", "x"),
			second,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("got %v", warnings)
	}
	if got := warnings[0].Message; !strings.Contains(got, "rule 1 (second)") {
		t.Fatalf("limit warning does not identify cutoff rule: %q", got)
	}
}

func TestLargeProductExactRulesCompleteWithinBudget(t *testing.T) {
	t.Parallel()
	axis := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
			gorege.DimValues(axis...),
		),
		gorege.WithRules(
			gorege.Allow("0", "0", "0", "0", "0", "0"),
			gorege.Deny("9", "9", "9", "9", "9", "9"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("large exact-heavy product should complete in two tuples: %v", warnings)
	}
}

func TestWildcardShadowConsumesGlobalBudget(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(3),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected one unchecked warning per wildcard rule, got %v", warnings)
	}
	for i, warning := range warnings {
		if warning.Kind != gorege.WarningKindAnalysisLimitExceeded ||
			!strings.Contains(warning.Message, []string{"rule 0", "rule 1"}[i]) {
			t.Fatalf("warning %d does not identify its unchecked rule: %v", i, warning)
		}
	}
}

func TestInfeasibleRuleDoesNotConsumeBudgetAndLaterExactRuleIsAnalyzed(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("a", "x"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 ||
		warnings[0].Kind != gorege.WarningKindShadowed ||
		warnings[1].Kind != gorege.WarningKindAnalysisLimitExceeded ||
		!strings.Contains(warnings[1].Message, "rule 0") {
		t.Fatalf("later exact rule was not analyzed after infeasible rule: %v", warnings)
	}
}

func TestMultipleInfeasibleRulesProduceLabeledWarnings(t *testing.T) {
	t.Parallel()
	first := gorege.Allow(gorege.Wildcard, gorege.Wildcard)
	first.Name = "first"
	second := gorege.Deny(gorege.AnyOf("a", "b"), gorege.Wildcard)
	second.Name = "second"
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(first, second),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 ||
		!strings.Contains(warnings[0].Message, "rule 0 (first)") ||
		!strings.Contains(warnings[1].Message, "rule 1 (second)") {
		t.Fatalf("unchecked warnings are not labeled in rule order: %v", warnings)
	}
}

func TestEffectiveProductFeasibilityBoundary(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(
			gorege.DimValues("a", "b", "c"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny(gorege.AnyOf("a", "b"), "x"),
			gorege.Deny(gorege.AnyOf("a", "b", "c"), "y"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 3 ||
		warnings[0].Kind != gorege.WarningKindShadowed ||
		!strings.Contains(warnings[0].Message, "rule 1") ||
		warnings[1].Kind != gorege.WarningKindAnalysisLimitExceeded ||
		!strings.Contains(warnings[1].Message, "rule 0") ||
		warnings[2].Kind != gorege.WarningKindAnalysisLimitExceeded ||
		!strings.Contains(warnings[2].Message, "rule 2") {
		t.Fatalf("product == remaining should run and remaining+1 should skip: %v", warnings)
	}
}

func TestOverflowingEffectiveProductIsUnchecked(t *testing.T) {
	t.Parallel()
	dims := make([]gorege.Dimension, 64)
	parts := make([]any, 64)
	for i := range dims {
		dims[i] = gorege.DimValues("0", "1")
		parts[i] = gorege.Wildcard
	}
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(dims...),
		gorege.WithRules(gorege.Allow(parts...)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("overflowing effective product should be unchecked: %v", warnings)
	}
}

func TestBudgetedAnyOfDeduplicatesValues(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(3),
		gorege.WithDimensions(
			gorege.DimValues("a", "b", "c"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.AnyOf("a", "b"), "x"),
			gorege.Deny(gorege.AnyOf("a", "a", "b"), "x"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || warnings[0].Kind != gorege.WarningKindShadowed {
		t.Fatalf("deduplicated AnyOf should complete within budget: %v", warnings)
	}
}

func TestKnownShadowWarningsPrecedeLimitWarning(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(gorege.DimValues("a", "b", "c")),
		gorege.WithRules(
			gorege.Allow("a"),
			gorege.Deny("a"),
			gorege.Allow("b"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 ||
		warnings[0].Kind != gorege.WarningKindShadowed ||
		warnings[1].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("warning order mismatch: %v", warnings)
	}
}

func TestWarningOrderDeadShadowedThenUnchecked(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(1),
		gorege.WithDimensions(
			gorege.DimValues("a", "b"),
			gorege.DimValues("x", "y"),
		),
		gorege.WithRules(
			gorege.Allow(gorege.AnyOf(), gorege.Wildcard),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("a", "x"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 3 ||
		warnings[0].Kind != gorege.WarningKindDead ||
		warnings[1].Kind != gorege.WarningKindShadowed ||
		warnings[2].Kind != gorege.WarningKindAnalysisLimitExceeded {
		t.Fatalf("warning order mismatch: %v", warnings)
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

func TestGlobalProductWithinLimitUsesCartesianAnalysis(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithAnalysisLimit(2),
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard),
			gorege.Deny(gorege.Wildcard),
			gorege.Allow(gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 ||
		warnings[0].Kind != gorege.WarningKindShadowed ||
		warnings[1].Kind != gorege.WarningKindShadowed {
		t.Fatalf("global product <= limit must use complete Cartesian analysis: %v", warnings)
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
