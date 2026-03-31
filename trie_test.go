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
