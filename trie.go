package gorege

const noMatch = -1

// trieChildThreshold: above this many distinct exact child keys, slice is upgraded to a map.
const trieChildThreshold = 16

// ruleTrieNode is one node in a multi-path priority trie over dimensions.
type ruleTrieNode struct {
	children []trieEntry
	// childrenMap is used after the slice grows past trieChildThreshold; nil means slice path.
	childrenMap map[string]*ruleTrieNode
	wildcard    *ruleTrieNode
	// minRuleIdx is the smallest rule index in this subtree; -1 if unset.
	minRuleIdx int
}

type trieEntry struct {
	key  string
	node *ruleTrieNode
}

func newTrieNode() *ruleTrieNode {
	return &ruleTrieNode{minRuleIdx: -1}
}

func (n *ruleTrieNode) getOrCreateChild(key string) *ruleTrieNode {
	if n.childrenMap != nil {
		if child, ok := n.childrenMap[key]; ok {
			return child
		}
		child := newTrieNode()
		n.childrenMap[key] = child
		return child
	}

	for i := range n.children {
		if n.children[i].key == key {
			return n.children[i].node
		}
	}
	child := newTrieNode()
	n.children = append(n.children, trieEntry{key: key, node: child})

	if len(n.children) > trieChildThreshold {
		n.childrenMap = make(map[string]*ruleTrieNode, len(n.children))
		for _, e := range n.children {
			n.childrenMap[e.key] = e.node
		}
		n.children = nil
	}
	return child
}

// insert adds rule ruleIdx (first-match order) at depth into the trie.
func (n *ruleTrieNode) insert(dims []Dimension, rule Rule, ruleIdx, depth int) {
	if ruleIdx < n.minRuleIdx || n.minRuleIdx == -1 {
		n.minRuleIdx = ruleIdx
	}

	d := len(dims)
	if depth == d {
		return
	}

	var m matcher
	if depth < len(rule.m) {
		m = rule.m[depth]
	} else {
		m = matcher{kind: mWildcard}
	}

	switch m.kind {
	case mWildcard:
		if n.wildcard == nil {
			n.wildcard = newTrieNode()
		}
		n.wildcard.insert(dims, rule, ruleIdx, depth+1)
	case mExact:
		child := n.getOrCreateChild(m.vals[0])
		child.insert(dims, rule, ruleIdx, depth+1)
	case mAnyOf:
		for _, v := range m.vals {
			child := n.getOrCreateChild(v)
			child.insert(dims, rule, ruleIdx, depth+1)
		}
	}
}

// search returns the first-match rule index for values, or noMatch.
func (n *ruleTrieNode) search(values []string, dims []Dimension, depth int) int {
	d := len(values)
	if depth == d {
		return n.minRuleIdx
	}

	best := noMatch

	input := values[depth]

	var exactChild *ruleTrieNode
	if n.childrenMap != nil {
		exactChild = n.childrenMap[input]
	} else {
		for i := range n.children {
			if n.children[i].key == input {
				exactChild = n.children[i].node
				break
			}
		}
	}

	if exactChild != nil &&
		(best == noMatch || exactChild.minRuleIdx < best) &&
		exactChild.minRuleIdx != noMatch {
		res := exactChild.search(values, dims, depth+1)
		if res != noMatch && (best == noMatch || res < best) {
			best = res
			if best == 0 {
				return 0
			}
		}
	}

	if n.wildcard != nil {
		dim := dims[depth]
		if len(dim.values) == 0 || dim.contains(input) {
			wc := n.wildcard
			if wc.minRuleIdx != noMatch && (best == noMatch || wc.minRuleIdx < best) {
				res := wc.search(values, dims, depth+1)
				if res != noMatch && (best == noMatch || res < best) {
					best = res
				}
			}
		}
	}

	return best
}

func buildTrie(dims []Dimension, rules []Rule) *ruleTrieNode {
	root := newTrieNode()
	for i := range rules {
		root.insert(dims, rules[i], i, 0)
	}
	return root
}
