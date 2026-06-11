package gorege

import "testing"

func firstMatchLinear(dims []Dimension, rules []Rule, values []string) int {
	d := len(dims)
	for i := range rules {
		if ruleMatches(rules[i], dims, d, values, false) {
			return i
		}
	}
	return noMatch
}

func TestTrieMinRuleIdxOnAncestors(t *testing.T) {
	t.Parallel()
	dims := []Dimension{DimValues("a", "b")}
	rules := []Rule{
		Allow("b"),
		Allow("a"),
	}
	root := buildTrie(dims, rules)
	if root.minRuleIdx != 0 {
		t.Fatalf("root minRuleIdx=%d want 0", root.minRuleIdx)
	}
	var childA *ruleTrieNode
	for i := range root.children {
		if root.children[i].key == "a" {
			childA = root.children[i].node
			break
		}
	}
	if childA == nil {
		t.Fatal("missing child a")
	}
	if childA.minRuleIdx != 1 {
		t.Fatalf("child a minRuleIdx=%d want 1", childA.minRuleIdx)
	}
}

func TestTrieSearchMatchesLinear(t *testing.T) {
	t.Parallel()
	dims := []Dimension{
		DimValues("Gold", "Guest", "Regular"),
		DimValues("Mon", "Tue", "Wed"),
		DimValues("Pool", "Gym", "Sauna"),
	}
	rules := []Rule{
		Allow("Gold", Wildcard, Wildcard),
		Deny("Guest", AnyOf("Mon", "Tue"), "Sauna"),
		Allow(AnyOf("Guest", "Regular"), Wildcard, Wildcard),
		Deny(Wildcard, Wildcard, Wildcard),
	}
	root := buildTrie(dims, rules)
	queries := [][]string{
		{"Gold", "Mon", "Sauna"},
		{"Guest", "Wed", "Sauna"},
		{"Guest", "Mon", "Sauna"},
		{"Regular", "Tue", "Gym"},
		{"Guest", "Wed", "Pool"},
	}
	for _, q := range queries {
		want := firstMatchLinear(dims, rules, q)
		got := root.search(q, dims, 0)
		if got != want {
			t.Fatalf("input=%v trie=%d linear=%d", q, got, want)
		}
	}
}

func TestTrieImplicitTrailingWildcard(t *testing.T) {
	t.Parallel()
	dims := []Dimension{
		DimValues("x", "y"),
		DimValues("1", "2"),
	}
	rules := []Rule{
		Allow("x"),
		Deny(Wildcard, Wildcard),
	}
	root := buildTrie(dims, rules)
	q := []string{"x", "1"}
	want := firstMatchLinear(dims, rules, q)
	got := root.search(q, dims, 0)
	if got != want {
		t.Fatalf("trie=%d linear=%d", got, want)
	}
}

func TestTrieWildcardBranchAndExactOrder(t *testing.T) {
	t.Parallel()
	dims := []Dimension{DimValues("a", "b"), DimValues("p", "q")}
	rules := []Rule{
		Deny("a", "p"),
		Allow(Wildcard, Wildcard),
	}
	root := buildTrie(dims, rules)
	q := []string{"a", "p"}
	want := firstMatchLinear(dims, rules, q)
	got := root.search(q, dims, 0)
	if got != want {
		t.Fatalf("trie=%d linear=%d", got, want)
	}
}

func TestTrieNoEarlyExitWhenMinIdxZeroButSuffixMismatch(t *testing.T) {
	t.Parallel()
	dims := []Dimension{
		DimValues("a", "b"),
		DimValues("x", "y"),
	}
	rules := []Rule{
		Allow("a", "x"),
		Allow("a", "y"),
	}
	root := buildTrie(dims, rules)
	q := []string{"a", "y"}
	want := firstMatchLinear(dims, rules, q)
	got := root.search(q, dims, 0)
	if got != want {
		t.Fatalf("trie=%d linear=%d (want rule 1 for input a,y)", got, want)
	}
}

func TestTrieAnyOfFanOut(t *testing.T) {
	t.Parallel()
	dims := []Dimension{DimValues("a", "b", "c")}
	rules := []Rule{
		Deny("c"),
		Allow(AnyOf("a", "b")),
	}
	root := buildTrie(dims, rules)
	for _, q := range [][]string{{"a"}, {"b"}, {"c"}} {
		want := firstMatchLinear(dims, rules, q)
		got := root.search(q, dims, 0)
		if got != want {
			t.Fatalf("input=%v trie=%d linear=%d", q, got, want)
		}
	}
}

// TestTrieMapModeUpgrade verifies that inserting more than trieChildThreshold (16)
// distinct exact keys at one node promotes the children slice to a map, and that
// all three map-path branches of getOrCreateChild are covered:
//
//  1. The slice-to-map upgrade (17th distinct key triggers it).
//  2. A new key inserted while already in map mode (18th distinct key).
//  3. An existing key looked up in map mode (duplicate key after map exists).
func TestTrieMapModeUpgrade(t *testing.T) {
	t.Parallel()
	// 18 distinct values: upgrade fires at the 17th, and the 18th exercises the
	// "insert new child into an already-map-mode node" path.
	vals := []string{
		"e01", "e02", "e03", "e04", "e05", "e06", "e07", "e08",
		"e09", "e10", "e11", "e12", "e13", "e14", "e15", "e16", "e17", "e18",
	}
	dims := []Dimension{DimValues(vals...)}
	rules := make([]Rule, len(vals))
	for i, v := range vals {
		rules[i] = Allow(v)
	}
	root := buildTrie(dims, rules)

	if root.childrenMap == nil {
		t.Fatal("expected map-mode trie after inserting >trieChildThreshold children")
	}
	if root.children != nil {
		t.Fatal("slice children should be nil after map promotion")
	}

	// Map-mode search must agree with the linear oracle for every known value.
	for i, v := range vals {
		want := firstMatchLinear(dims, rules, []string{v})
		got := root.search([]string{v}, dims, 0)
		if got != want {
			t.Errorf("v=%q: trie=%d linear=%d (expected rule %d)", v, got, want, i)
		}
	}

	// A value absent from the map should return noMatch.
	if got := root.search([]string{"missing"}, dims, 0); got != noMatch {
		t.Errorf("missing key: expected noMatch(-1), got %d", got)
	}

	// Add an extra rule that reuses an already-mapped key ("e01") to exercise
	// the getOrCreateChild "map returns existing child" (ok == true) branch.
	rulesWithDup := append(append([]Rule{}, rules...), Allow("e01"))
	root2 := buildTrie(dims, rulesWithDup)
	if root2.childrenMap == nil {
		t.Fatal("root2: expected map-mode trie")
	}
	// The original Allow("e01") at index 0 wins under first-match.
	want0 := firstMatchLinear(dims, rulesWithDup, []string{"e01"})
	got0 := root2.search([]string{"e01"}, dims, 0)
	if got0 != want0 {
		t.Errorf("dup e01: trie=%d linear=%d", got0, want0)
	}
}
