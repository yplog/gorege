package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yplog/gorege"
)

func main() {
	path := "rules.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	e, warnings, err := gorege.LoadFileWithOptions(path)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	for _, w := range warnings {
		log.Printf("WARNING [%s]: %s", w.Kind, w.Message)
	}

	cases := []struct {
		plan   string
		region string
	}{
		{"free", "eu"},
		{"free", "us"},
		{"free", "apac"},
		{"pro", "eu"},
		{"pro", "us"},
		{"enterprise", "apac"},
	}

	for _, c := range cases {
		ok, err := e.Check(c.plan, c.region)
		if err != nil {
			log.Fatal(err)
		}
		mark := "OK"
		if !ok {
			mark = "NO"
		}
		fmt.Printf("  %s  plan=%-10s  region=%s\n", mark, c.plan, c.region)
	}

	reachable, err := e.PartialCheck("free")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nfree/* reachable: %v\n", reachable)

	reachable, err = e.PartialCheck("pro")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pro/*  reachable: %v\n", reachable)

	res, err := e.ClosestIn("plan", "free", "eu")
	if err != nil {
		log.Fatal(err)
	}
	if res != nil {
		fmt.Printf("\nUpgrade suggestion: plan=%q (distance: %d)\n", res.Value, res.Distance)
	}
}
