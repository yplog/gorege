package main

import (
	"fmt"
	"os"

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
	case "lint":
		return runLint(args[2:])
	default:
		usage()
		return 2
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gorege check <config.json> <dim values...>")
	fmt.Fprintln(os.Stderr, "       gorege lint <config.json>")
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
		fmt.Fprintln(os.Stderr, w.Message)
	}
	return 1
}
