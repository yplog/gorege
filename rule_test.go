package gorege_test

import (
	"testing"

	"github.com/yplog/gorege"
)

func TestAllowInvalidMatcherPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	gorege.Allow(42)
}

func TestAnyOfNoMatch(t *testing.T) {
	t.Parallel()
	e, _, err := gorege.New(
		gorege.WithDimensions(gorege.DimValues("a", "b")),
		gorege.WithRules(gorege.Allow(gorege.AnyOf("a"))),
	)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Check("b")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
