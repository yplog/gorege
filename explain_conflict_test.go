package gorege_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/yplog/gorege"
)

func TestExplainMatchesFirstRule(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow("a"),
			gorege.Deny(gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	x, err := e.Explain("a")
	if err != nil {
		t.Fatal(err)
	}
	if !x.Matched || x.RuleIndex != 0 || !x.Allowed || x.Action != gorege.ActionAllow {
		t.Fatalf("%+v", x)
	}
}

func TestExplainImplicitDeny(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Deny("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	x, err := e.Explain("b")
	if err != nil {
		t.Fatal(err)
	}
	if x.Matched || x.Allowed || x.RuleIndex != -1 {
		t.Fatalf("%+v", x)
	}
}

func TestExplainArity(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a")),
		gorege.WithRules(gorege.Allow("a")),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Explain()
	if !errors.Is(err, gorege.ErrArityMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestExplainRuleNameFromJSON(t *testing.T) {
	t.Parallel()
	doc := `{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","name":"gold","conditions":["a"]}]}`
	e, _, err := gorege.Load(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	x, err := e.Explain("a")
	if err != nil {
		t.Fatal(err)
	}
	if x.RuleName != "gold" || !x.Matched || !x.Allowed {
		t.Fatalf("%+v", x)
	}
}

func TestShadowedRuleWarning(t *testing.T) {
	t.Parallel()
	_, warnings, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(
			gorege.Allow(gorege.Wildcard),
			gorege.Deny("a"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
}

func TestDeadRuleWarning(t *testing.T) {
	t.Parallel()
	// Empty dimension ⇒ empty Cartesian product ⇒ no rule can match any tuple.
	_, warnings, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues()),
		gorege.WithRules(
			gorege.Allow(),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range warnings {
		if len(w.Message) >= 4 && w.Message[:4] == "dead" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dead rule warning, got %#v", warnings)
	}
}
