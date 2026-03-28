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
