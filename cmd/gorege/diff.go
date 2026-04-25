package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/yplog/gorege"
)

type transitionKind int

const (
	transUnchanged transitionKind = iota
	transAllowToDeny
	transDenyToAllow
	transRuleChanged
)

func (k transitionKind) String() string {
	switch k {
	case transUnchanged:
		return "UNCHANGED"
	case transAllowToDeny:
		return "ALLOW→DENY"
	case transDenyToAllow:
		return "DENY→ALLOW"
	case transRuleChanged:
		return "RULE_CHANGED"
	default:
		return "?"
	}
}

type transition struct {
	Tuple    []string `json:"tuple"`
	Kind     string   `json:"kind"`
	OldRule  string   `json:"old_rule,omitempty"`
	NewRule  string   `json:"new_rule,omitempty"`
	OldIndex int      `json:"old_index"`
	NewIndex int      `json:"new_index"`
	OldOk    bool     `json:"old_ok"`
	NewOk    bool     `json:"new_ok"`
}

type diffSummary struct {
	Total            int          `json:"total"`
	Unchanged        int          `json:"unchanged"`
	AllowToDeny      int          `json:"allow_to_deny"`
	DenyToAllow      int          `json:"deny_to_allow"`
	RuleChanged      int          `json:"rule_changed"`
	DimensionProduct int64        `json:"dimension_product"`
	Limit            int          `json:"limit"`
	Transitions      []transition `json:"transitions"`
}

func classifyTransition(ox, nx gorege.Explanation) transitionKind {
	oldOK := ox.Matched && ox.Allowed
	newOK := nx.Matched && nx.Allowed
	switch {
	case oldOK && !newOK:
		return transAllowToDeny
	case !oldOK && newOK:
		return transDenyToAllow
	case oldOK == newOK && (ox.RuleIndex != nx.RuleIndex || ox.RuleName != nx.RuleName):
		return transRuleChanged
	default:
		return transUnchanged
	}
}

func dimensionsCompatible(a, b []gorege.Dimension) error {
	if len(a) != len(b) {
		return fmt.Errorf("dimension count mismatch: old=%d new=%d", len(a), len(b))
	}
	for i := range a {
		if a[i].Name() != b[i].Name() {
			return fmt.Errorf("dim[%d] name mismatch: old=%q new=%q", i, a[i].Name(), b[i].Name())
		}
		av, bv := a[i].Values(), b[i].Values()
		if !stringSliceEqual(av, bv) {
			return fmt.Errorf("dim[%d] (%q) values differ: old=%v new=%v", i, a[i].Name(), av, bv)
		}
	}
	return nil
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func product(dims []gorege.Dimension, limit int) (int64, bool) {
	var p int64 = 1
	for _, d := range dims {
		n := int64(len(d.Values()))
		if n == 0 {
			return 0, false
		}
		p *= n
		if p > int64(limit) {
			return p, true
		}
	}
	return p, false
}

func walkProduct(dims []gorege.Dimension, fn func(tuple []string)) {
	if len(dims) == 0 {
		fn(nil)
		return
	}
	valuesPerDim := make([][]string, len(dims))
	for i, d := range dims {
		valuesPerDim[i] = d.Values()
	}
	cur := make([]string, len(dims))
	var rec func(idx int)
	rec = func(idx int) {
		if idx == len(dims) {
			fn(cur)
			return
		}
		for _, v := range valuesPerDim[idx] {
			cur[idx] = v
			rec(idx + 1)
		}
	}
	rec(0)
}

func splitDiffArgs(args []string) (flagArgs, positional []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		switch {
		case strings.HasPrefix(arg, "--limit="), strings.HasPrefix(arg, "-limit="),
			strings.HasPrefix(arg, "--format="), strings.HasPrefix(arg, "-format="),
			arg == "--include-unchanged", arg == "-include-unchanged":
			flagArgs = append(flagArgs, arg)
		case arg == "--limit" || arg == "-limit" || arg == "--format" || arg == "-format":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %s requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		default:
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		}
	}
	return flagArgs, positional, nil
}

func runDiff(args []string) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", gorege.DefaultAnalysisLimit, "Cartesian product cap")
	format := fs.String("format", "text", "output format: text|json")
	includeUnchanged := fs.Bool("include-unchanged", false, "include unchanged tuples in JSON output")

	flagArgs, positional, splitErr := splitDiffArgs(args)
	if splitErr != nil {
		fmt.Fprintln(os.Stderr, "gorege diff:", splitErr)
		return 2
	}
	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintln(os.Stderr, "gorege diff:", err)
		return 2
	}
	if *limit <= 0 {
		fmt.Fprintln(os.Stderr, "gorege diff: --limit must be > 0")
		return 2
	}
	if len(positional) != 2 {
		fmt.Fprintln(os.Stderr, "gorege diff: need <old.json> <new.json>")
		return 2
	}
	oldPath, newPath := positional[0], positional[1]

	oldEng, oldWarn, err := gorege.LoadFileWithOptions(oldPath, gorege.WithAnalysisLimit(-1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege diff:", err)
		return 1
	}
	newEng, newWarn, err := gorege.LoadFileWithOptions(newPath, gorege.WithAnalysisLimit(-1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege diff:", err)
		return 1
	}
	for _, w := range append(oldWarn, newWarn...) {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}

	if err := dimensionsCompatible(oldEng.Dimensions(), newEng.Dimensions()); err != nil {
		fmt.Fprintln(os.Stderr, "gorege diff:", err)
		return 1
	}

	dims := oldEng.Dimensions()
	p, exceeded := product(dims, *limit)
	if exceeded {
		fmt.Fprintf(os.Stderr, "gorege diff: dimension product (~%d) exceeds limit (%d); use --limit to raise\n", p, *limit)
		return 1
	}
	if p == 0 {
		fmt.Fprintln(os.Stderr, "gorege diff: cannot enumerate (a dimension has empty value list)")
		return 1
	}

	summary := diffSummary{
		Total:            int(p),
		DimensionProduct: p,
		Limit:            *limit,
	}

	walkProduct(dims, func(t []string) {
		ox, errO := oldEng.Explain(t...)
		nx, errN := newEng.Explain(t...)
		if errO != nil || errN != nil {
			return
		}
		kind := classifyTransition(ox, nx)
		switch kind {
		case transUnchanged:
			summary.Unchanged++
			if !*includeUnchanged {
				return
			}
		case transAllowToDeny:
			summary.AllowToDeny++
		case transDenyToAllow:
			summary.DenyToAllow++
		case transRuleChanged:
			summary.RuleChanged++
		}

		tCopy := append([]string(nil), t...)
		summary.Transitions = append(summary.Transitions, transition{
			Tuple:    tCopy,
			Kind:     kind.String(),
			OldRule:  ox.RuleName,
			NewRule:  nx.RuleName,
			OldIndex: ox.RuleIndex,
			NewIndex: nx.RuleIndex,
			OldOk:    ox.Matched && ox.Allowed,
			NewOk:    nx.Matched && nx.Allowed,
		})
	})

	sortTransitions(summary.Transitions)

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(summary); err != nil {
			fmt.Fprintln(os.Stderr, "gorege diff:", err)
			return 1
		}
	case "text":
		printSummaryText(os.Stdout, summary)
	default:
		fmt.Fprintf(os.Stderr, "gorege diff: unknown format %q\n", *format)
		return 2
	}

	if summary.AllowToDeny+summary.DenyToAllow > 0 {
		return 1
	}
	return 0
}

func sortTransitions(ts []transition) {
	rank := func(k string) int {
		switch k {
		case "ALLOW→DENY":
			return 0
		case "DENY→ALLOW":
			return 1
		case "RULE_CHANGED":
			return 2
		case "UNCHANGED":
			return 3
		}
		return 4
	}
	sort.SliceStable(ts, func(i, j int) bool {
		ri, rj := rank(ts[i].Kind), rank(ts[j].Kind)
		if ri != rj {
			return ri < rj
		}
		return strings.Join(ts[i].Tuple, "|") < strings.Join(ts[j].Tuple, "|")
	})
}

func printSummaryText(w io.Writer, s diffSummary) {
	fmt.Fprintf(w, "tuples examined: %d (product=%d, limit=%d)\n", s.Total, s.DimensionProduct, s.Limit)
	fmt.Fprintf(w, "  ALLOW→DENY:   %d\n", s.AllowToDeny)
	fmt.Fprintf(w, "  DENY→ALLOW:   %d\n", s.DenyToAllow)
	fmt.Fprintf(w, "  rule changed: %d\n", s.RuleChanged)
	fmt.Fprintf(w, "  unchanged:    %d\n", s.Unchanged)

	decisionChanges := s.AllowToDeny + s.DenyToAllow
	if decisionChanges == 0 {
		fmt.Fprintln(w, "\nno decision changes")
		return
	}

	fmt.Fprintln(w, "\nfirst 50 changes:")
	n := 50
	if len(s.Transitions) < n {
		n = len(s.Transitions)
	}
	printed := 0
	for _, t := range s.Transitions {
		if t.Kind == "UNCHANGED" {
			continue
		}
		fmt.Fprintf(w, "  %-12s %v  old=%q (idx=%d)  new=%q (idx=%d)\n", t.Kind, t.Tuple, t.OldRule, t.OldIndex, t.NewRule, t.NewIndex)
		printed++
		if printed >= n {
			break
		}
	}
	if len(s.Transitions) > n {
		fmt.Fprintf(w, "  ... and %d more (use --format json for full list)\n", len(s.Transitions)-n)
	}
}
