package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/yplog/gorege"
)

func main() {
	os.Exit(realMain(os.Args))
}

func realMain(args []string) int {
	if len(args) < 2 {
		usage()
		return 2
	}
	switch args[1] {
	case "check":
		return runCheck(args[2:])
	case "explain":
		return runExplain(args[2:])
	case "closest":
		return runClosest(args[2:])
	case "closest-in":
		return runClosestIn(args[2:])
	case "lint":
		return runLint(args[2:])
	case "partial-check":
		return runPartialCheck(args[2:])
	default:
		usage()
		return 2
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gorege check <config.json> <dim values...>")
	fmt.Fprintln(os.Stderr, "       gorege explain <config.json> <dim values...>")
	fmt.Fprintln(os.Stderr, "       gorege closest <config.json> <dim values...>")
	fmt.Fprintln(os.Stderr, "       gorege closest-in <config.json> <dim-index-or-name> <dim values...>")
	fmt.Fprintln(os.Stderr, "       gorege lint <config.json>")
	fmt.Fprintln(os.Stderr, "       gorege partial-check <config.json> [<dim values...>]")
}

func runCheck(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "gorege check: missing config path")
		return 2
	}
	path := args[0]
	vals := args[1:]
	e, warnings, err := gorege.LoadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege check:", err)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}
	ok, err := e.Check(vals...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege check:", err)
		return 1
	}
	fmt.Println(ok)
	if !ok {
		return 1
	}
	return 0
}

func runExplain(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "gorege explain: missing config path")
		return 2
	}
	path := args[0]
	vals := args[1:]
	e, warnings, err := gorege.LoadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege explain:", err)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}
	x, err := e.Explain(vals...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege explain:", err)
		return 1
	}
	fmt.Printf("matched: %v\n", x.Matched)
	fmt.Printf("allowed: %v\n", x.Allowed)
	fmt.Printf("rule_index: %d\n", x.RuleIndex)
	fmt.Printf("rule_name: %s\n", x.RuleName)
	if x.Matched {
		fmt.Printf("action: %s\n", x.Action.String())
	} else {
		fmt.Println("action: (implicit deny, no rule matched)")
	}
	return 0
}

func runPartialCheck(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "gorege partial-check: missing config path")
		return 2
	}
	path := args[0]
	vals := args[1:]
	e, warnings, err := gorege.LoadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege partial-check:", err)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}
	if len(vals) > len(e.Dimensions()) {
		fmt.Fprintf(os.Stderr, "gorege partial-check: at most %d dimension values allowed, got %d\n", len(e.Dimensions()), len(vals))
		return 1
	}
	ok, err := e.PartialCheck(vals...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege partial-check:", err)
		return 1
	}
	fmt.Println(ok)
	if !ok {
		return 1
	}
	return 0
}

func runClosest(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "gorege closest: missing config path")
		return 2
	}
	path := args[0]
	vals := args[1:]
	e, warnings, err := gorege.LoadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege closest:", err)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}
	want := len(e.Dimensions())
	if len(vals) != want {
		fmt.Fprintf(os.Stderr, "gorege closest: need %d dimension values, got %d\n", want, len(vals))
		return 2
	}
	res, err := e.Closest(vals...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege closest:", err)
		return 1
	}
	if res == nil {
		fmt.Println("found: false")
		return 1
	}
	printClosestResult(res)
	return 0
}

func runClosestIn(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gorege closest-in: need config path and dimension (index or name)")
		return 2
	}
	path := args[0]
	dimSel := args[1]
	vals := args[2:]
	e, warnings, err := gorege.LoadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege closest-in:", err)
		return 1
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w.Message)
	}
	want := len(e.Dimensions())
	if len(vals) != want {
		fmt.Fprintf(os.Stderr, "gorege closest-in: need %d dimension values, got %d\n", want, len(vals))
		return 2
	}
	dim := any(dimSel)
	if i, err := strconv.Atoi(dimSel); err == nil {
		dim = i
	}
	res, err := e.ClosestIn(dim, vals...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege closest-in:", err)
		return 1
	}
	if res == nil {
		fmt.Println("found: false")
		return 1
	}
	printClosestResult(res)
	return 0
}

func printClosestResult(res *gorege.ClosestResult) {
	_, _ = fmt.Println("found: true")
	condJSON, err := json.Marshal(res.Conditions)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "conditions: %q\n", res.Conditions)
	} else {
		_, _ = fmt.Printf("conditions: %s\n", condJSON)
	}
	_, _ = fmt.Printf("distance: %d\n", res.Distance)
	_, _ = fmt.Printf("dim_index: %d\n", res.DimIndex)
	_, _ = fmt.Printf("dim_name: %s\n", res.DimName)
	_, _ = fmt.Printf("value: %s\n", res.Value)
}

func runLint(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "gorege lint: need exactly one config path")
		return 2
	}
	_, warnings, err := gorege.LoadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "gorege lint:", err)
		return 1
	}
	if len(warnings) == 0 {
		fmt.Println("ok")
		return 0
	}
	for _, w := range warnings {
		fmt.Println(w.Message)
	}
	return 1
}
