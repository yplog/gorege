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
