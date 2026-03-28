package gorege_test

import (
	"testing"

	"github.com/yplog/gorege"
)

func TestDimEmptyNamePanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	gorege.Dim("")
}

func TestDimensionName(t *testing.T) {
	t.Parallel()
	if gorege.Dim("region", "eu", "us").Name() != "region" {
		t.Fatal()
	}
	if gorege.DimValues("a", "b").Name() != "" {
		t.Fatal("anonymous dimension name should be empty")
	}
}
