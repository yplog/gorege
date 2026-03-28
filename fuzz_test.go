package gorege_test

import (
	"bytes"
	"testing"

	"github.com/yplog/gorege"
)

// fuzzTwoDim is a fixed engine for Check / PartialCheck / Explain / Closest fuzzing.
var fuzzTwoDim *gorege.Engine

func init() {
	var err error
	fuzzTwoDim, _, err = gorege.New(
		gorege.WithDimensions(
			gorege.Dim("d0", "a", "b", "c", "d"),
			gorege.Dim("d1", "x", "y", "z", "w"),
		),
		gorege.WithRules(
			gorege.Allow("a", gorege.Wildcard),
			gorege.Deny(gorege.Wildcard, "y"),
			gorege.Allow(gorege.AnyOf("b", "c"), gorege.AnyOf("x", "z")),
			gorege.Deny(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		panic("fuzzTwoDim init: " + err.Error())
	}
}

// FuzzLoad feeds arbitrary bytes into [gorege.Load] to exercise JSON decoding,
// dimension/rule loading, and condition slot parsing (including []any shapes
// produced by encoding/json). Must not panic.
func FuzzLoad(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"dimensions":[],"rules":[]}`))
	f.Add([]byte(`{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","conditions":["a"]}]}`))
	f.Add([]byte(`{"dimensions":[{"values":["a","b"]}],"rules":[{"action":"ALLOW","conditions":["*"]}]}`))
	f.Add([]byte(`{"dimensions":[{"name":"x","values":["a"]}],"rules":[{"action":"ALLOW","conditions":[[1]]}]}`))
	f.Add([]byte(`{"rules":[{"action":"ALLOW","conditions":["x"]}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Helper()
		_, _, _ = gorege.Load(bytes.NewReader(data))
	})
}

// FuzzCheckTwoDims exercises [gorege.Engine.Check] on a fixed two-dimensional
// engine. Must not panic.
func FuzzCheckTwoDims(f *testing.F) {
	f.Add([]byte("a"), []byte("x"))
	f.Add([]byte(""), []byte("y"))
	f.Add([]byte("c"), []byte("z"))
	f.Add([]byte{0xff, 0xfe}, []byte("x"))

	f.Fuzz(func(t *testing.T, v0, v1 []byte) {
		t.Helper()
		_, _ = fuzzTwoDim.Check(string(v0), string(v1))
	})
}

// FuzzPartialCheckTwoDims exercises [gorege.Engine.PartialCheck] with 0..N
// segments split on NUL. Must not panic.
func FuzzPartialCheckTwoDims(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("a"))
	f.Add([]byte("a\x00"))
	f.Add([]byte("b\x00x"))

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Helper()
		parts := bytes.Split(data, []byte{0})
		if len(parts) > 16 {
			parts = parts[:16]
		}
		vals := make([]string, len(parts))
		for i, p := range parts {
			vals[i] = string(p)
		}
		_, _ = fuzzTwoDim.PartialCheck(vals...)
	})
}

// FuzzExplainTwoDims exercises [gorege.Engine.Explain] on the fixed engine.
func FuzzExplainTwoDims(f *testing.F) {
	f.Add([]byte("b"), []byte("x"))
	f.Fuzz(func(t *testing.T, v0, v1 []byte) {
		t.Helper()
		_, _ = fuzzTwoDim.Explain(string(v0), string(v1))
	})
}

// FuzzClosestTwoDims exercises [gorege.Engine.Closest] and [gorege.Engine.ClosestIn].
func FuzzClosestTwoDims(f *testing.F) {
	f.Add([]byte("a"), []byte("y"))
	f.Fuzz(func(t *testing.T, v0, v1 []byte) {
		t.Helper()
		s0, s1 := string(v0), string(v1)
		_, _ = fuzzTwoDim.Closest(s0, s1)
		_, _ = fuzzTwoDim.ClosestIn(0, s0, s1)
		_, _ = fuzzTwoDim.ClosestIn(1, s0, s1)
		_, _ = fuzzTwoDim.ClosestIn("d0", s0, s1)
	})
}
