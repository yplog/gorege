package gorege_test

import (
	"testing"

	"github.com/yplog/gorege"
)

func TestActionString(t *testing.T) {
	t.Parallel()
	if gorege.ActionAllow.String() != "ALLOW" {
		t.Fatal()
	}
	if gorege.ActionDeny.String() != "DENY" {
		t.Fatal()
	}
}

func TestRuleAction(t *testing.T) {
	t.Parallel()
	r := gorege.Allow("a")
	if r.Action() != gorege.ActionAllow {
		t.Fatal()
	}
	r2 := gorege.Deny("a")
	if r2.Action() != gorege.ActionDeny {
		t.Fatal()
	}
}

func TestWarningString(t *testing.T) {
	t.Parallel()
	w := gorege.Warning{Kind: gorege.WarningKindDead, Message: "x"}
	if w.String() != "x" {
		t.Fatal()
	}
}

func TestWarningKindString(t *testing.T) {
	t.Parallel()
	if gorege.WarningKindDead.String() != "dead" {
		t.Fatal()
	}
	if gorege.WarningKindShadowed.String() != "shadowed" {
		t.Fatal()
	}
	if gorege.WarningKindAnalysisLimitExceeded.String() != "analysis_limit_exceeded" {
		t.Fatal()
	}
	if s := gorege.WarningKind(42).String(); s != "WarningKind(42)" {
		t.Fatalf("unexpected string: %q", s)
	}
}

func TestTiebreakGoString(t *testing.T) {
	t.Parallel()
	if s := gorege.TiebreakLeftmostDim.GoString(); s != "TiebreakLeftmostDim" {
		t.Fatal(s)
	}
	if s := gorege.TiebreakRightmostDim.GoString(); s != "TiebreakRightmostDim" {
		t.Fatal(s)
	}
	if s := gorege.TiebreakDeclOrder.GoString(); s != "TiebreakDeclOrder" {
		t.Fatal(s)
	}
	if s := gorege.TiebreakStrategy(99).GoString(); s != "TiebreakStrategy(99)" {
		t.Fatal(s)
	}
}
