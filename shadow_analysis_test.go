package gorege

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

// shadowWarningsLinearOracle deliberately reimplements matching and Cartesian
// enumeration without production shadow-analysis helpers.
func shadowWarningsLinearOracle(dims []Dimension, rules []Rule) []Warning {
	dead := make([]bool, len(rules))
	wins := make([]bool, len(rules))

	for j, r := range rules {
		for i, dim := range dims {
			if len(dim.values) == 0 {
				dead[j] = true
				break
			}
			if i >= len(r.m) || r.m[i].kind == mWildcard {
				continue
			}
			matchesDeclared := false
			for _, declared := range dim.values {
				if oracleMatcherMatches(r.m[i], declared) {
					matchesDeclared = true
					break
				}
			}
			if !matchesDeclared {
				dead[j] = true
				break
			}
		}
	}

	var visit func(int, []string)
	visit = func(depth int, tuple []string) {
		if depth == len(dims) {
			for j, r := range rules {
				if dead[j] || !oracleRuleMatches(r, dims, tuple) {
					continue
				}
				wins[j] = true
				break
			}
			return
		}
		for _, value := range dims[depth].values {
			tuple[depth] = value
			visit(depth+1, tuple)
		}
	}
	if len(dims) == 0 {
		visit(0, nil)
	} else {
		visit(0, make([]string, len(dims)))
	}

	var out []Warning
	for j, r := range rules {
		if dead[j] {
			label := ruleWarningLabel(j, r)
			out = append(out, Warning{
				Kind:    WarningKindDead,
				Message: "dead rule " + label + ": never matches any tuple in the dimension product",
			})
		}
	}
	for j, r := range rules {
		if !dead[j] && !wins[j] {
			label := ruleWarningLabel(j, r)
			out = append(out, Warning{
				Kind:    WarningKindShadowed,
				Message: "shadowed rule " + label + ": never wins first-match against earlier rules",
			})
		}
	}
	return out
}

func oracleMatcherMatches(m matcher, value string) bool {
	switch m.kind {
	case mExact:
		return len(m.vals) == 1 && m.vals[0] == value
	case mAnyOf:
		for _, candidate := range m.vals {
			if candidate == value {
				return true
			}
		}
		return false
	case mWildcard:
		return true
	default:
		return false
	}
}

func oracleRuleMatches(r Rule, dims []Dimension, tuple []string) bool {
	for i := range dims {
		if i < len(r.m) && !oracleMatcherMatches(r.m[i], tuple[i]) {
			return false
		}
	}
	return true
}

func TestShadowWarningsCartesianMatchLinearOracle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		dims  []Dimension
		rules []Rule
	}{
		{
			name: "duplicates and trailing wildcard",
			dims: []Dimension{
				DimValues("a", "b"),
				DimValues("x", "y"),
			},
			rules: []Rule{
				Allow(AnyOf("a", "a"), "x"),
				Deny("a"),
				Allow(Wildcard, Wildcard),
			},
		},
		{
			name:  "zero dimensions",
			rules: []Rule{Allow(), Deny()},
		},
		{
			name: "zero rules",
			dims: []Dimension{DimValues("a")},
		},
		{
			name: "empty dimension values",
			dims: []Dimension{DimValues()},
			rules: []Rule{
				Allow(Wildcard),
				Deny(AnyOf()),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, got, err := New(
				WithDimensions(tc.dims...),
				WithRules(tc.rules...),
				WithAnalysisLimit(10_000),
			)
			if err != nil {
				t.Fatal(err)
			}
			want := shadowWarningsLinearOracle(tc.dims, tc.rules)
			if !slices.Equal(got, want) {
				t.Fatalf("warnings mismatch\ngot  %#v\nwant %#v", got, want)
			}
		})
	}
}

func TestBudgetedLargeProductMatchesLinearOracle(t *testing.T) {
	t.Parallel()
	axis := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	dims := []Dimension{
		DimValues(axis...),
		DimValues(axis...),
		DimValues(axis...),
		DimValues(axis...),
		DimValues(axis...),
		DimValues(axis...),
	}
	rules := []Rule{
		Allow("0", "0", "0", "0", "0", "0"),
		Deny(AnyOf("1", "1", "2"), "3", "4", "5", "6", "7"),
		Allow("9", "9", "9", "9", "9", "9"),
	}
	_, got, err := New(
		WithDimensions(dims...),
		WithRules(rules...),
		WithAnalysisLimit(4),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := shadowWarningsLinearOracle(dims, rules)
	if !slices.Equal(got, want) {
		t.Fatalf("warnings mismatch\ngot  %#v\nwant %#v", got, want)
	}
}

func TestDeadRulesDoNotChangeTrieFirstMatchOnDeclaredProduct(t *testing.T) {
	t.Parallel()
	dims := []Dimension{
		DimValues("a", "b"),
		DimValues("x", "y"),
	}
	rules := []Rule{
		Allow(AnyOf(), Wildcard),
		Deny("a", "x"),
		Allow(Wildcard, Wildcard),
	}
	all := buildTrie(dims, rules)
	live := buildTrie(dims, rules[1:])

	for _, a := range dims[0].values {
		for _, b := range dims[1].values {
			got := all.search([]string{a, b}, dims, 0)
			want := live.search([]string{a, b}, dims, 0)
			if want != noMatch {
				want++
			}
			if got != want {
				t.Fatalf("tuple %q,%q: all=%d live=%d", a, b, got, want)
			}
		}
	}
}

func FuzzShadowWarningsTrieVsLinearOracle(f *testing.F) {
	f.Add(uint8(2), uint8(3), uint8(8), int64(1))
	f.Add(uint8(0), uint8(2), uint8(3), int64(2))
	f.Add(uint8(3), uint8(1), uint8(12), int64(3))

	f.Fuzz(func(t *testing.T, dimByte, valueByte, ruleByte uint8, seed int64) {
		numDims := int(dimByte % 4)
		numValues := int(valueByte % 4)
		numRules := int(ruleByte % 20)
		rng := rand.New(rand.NewSource(seed))

		dims := make([]Dimension, numDims)
		for i := range dims {
			values := make([]string, numValues)
			for j := range values {
				values[j] = fmt.Sprintf("d%dv%d", i, j)
			}
			dims[i] = DimValues(values...)
		}

		rules := make([]Rule, numRules)
		for j := range rules {
			width := 0
			if numDims > 0 {
				width = rng.Intn(numDims + 1)
			}
			parts := make([]any, width)
			for i := range parts {
				switch rng.Intn(3) {
				case 0:
					parts[i] = Wildcard
				case 1:
					if numValues == 0 {
						parts[i] = AnyOf()
					} else {
						parts[i] = dims[i].values[rng.Intn(numValues)]
					}
				default:
					if numValues == 0 {
						parts[i] = AnyOf()
					} else {
						value := dims[i].values[rng.Intn(numValues)]
						parts[i] = AnyOf(value, value)
					}
				}
			}
			rules[j] = Allow(parts...)
		}

		_, got, err := New(
			WithDimensions(dims...),
			WithRules(rules...),
			WithAnalysisLimit(10_000),
		)
		if err != nil {
			t.Fatal(err)
		}
		want := shadowWarningsLinearOracle(dims, rules)
		if !slices.Equal(got, want) {
			t.Fatalf("warnings mismatch\ngot  %#v\nwant %#v", got, want)
		}
	})
}
