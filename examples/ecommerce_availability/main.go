package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/yplog/gorege"
)

var activeEngine atomic.Pointer[gorege.Engine]

func loadEngine(path string) (*gorege.Engine, error) {
	e, warnings, err := gorege.LoadFileWithOptions(path)
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		log.Printf("WARNING [%s]: %s", w.Kind, w.Message)
	}
	return e, nil
}

func main() {
	path := "rules.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	e, err := loadEngine(path)
	if err != nil {
		log.Fatalf("initial load failed: %v", err)
	}
	activeEngine.Store(e)

	type combo struct {
		region   string
		tier     string
		channel  string
		category string
	}

	cases := []combo{
		{"tr", "standard", "mobile", "electronics"},
		{"tr", "standard", "mobile", "clothing"},
		{"eu", "premium", "api", "food"},
		{"eu", "vip", "api", "food"},
		{"us", "standard", "api", "clothing"},
		{"us", "premium", "api", "clothing"},
	}

	for _, c := range cases {
		ok, err := activeEngine.Load().Check(c.region, c.tier, c.channel, c.category)
		if err != nil {
			log.Fatal(err)
		}
		mark := "OK"
		if !ok {
			mark = "NO"
		}
		fmt.Printf("  %s  %s/%s/%s/%s\n", mark, c.region, c.tier, c.channel, c.category)
	}

	x, err := activeEngine.Load().Explain("tr", "standard", "mobile", "electronics")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nmatched:    %v\n", x.Matched)
	fmt.Printf("allowed:    %v\n", x.Allowed)
	fmt.Printf("rule_index: %d\n", x.RuleIndex)
	fmt.Printf("rule_name:  %s\n", x.RuleName)
	fmt.Printf("action:     %s\n", x.Action.String())

	reachable, err := activeEngine.Load().PartialCheck("tr", "standard", "api")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\ntr/standard/api/* purchasable: %v\n", reachable)

	reachable, err = activeEngine.Load().PartialCheck("tr", "premium", "api")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tr/premium/api/*  purchasable: %v\n", reachable)

	res, err := activeEngine.Load().Closest("tr", "standard", "mobile", "electronics")
	if err != nil {
		log.Fatal(err)
	}
	if res != nil {
		fmt.Printf("\nfound:      %v\n", true)
		fmt.Printf("conditions: %v\n", res.Conditions)
		fmt.Printf("distance:   %d (Hamming)\n", res.Distance)
		fmt.Printf("pivot:      dim[%d]=%s -> %q\n", res.DimIndex, res.DimName, res.Value)
	}

	tierRes, err := activeEngine.Load().ClosestIn("tier", "tr", "standard", "mobile", "electronics")
	if err != nil {
		log.Fatal(err)
	}
	if tierRes != nil {
		fmt.Printf("\nUpgrade tier to %q -> purchase unlocked (distance: %d)\n", tierRes.Value, tierRes.Distance)
	}

	newEngine, err := loadEngine(path)
	if err != nil {
		log.Printf("reload failed, keeping current engine: %v", err)
	} else {
		activeEngine.Store(newEngine)
		fmt.Println("\nengine swapped (zero downtime)")
	}

	ok, err := activeEngine.Load().Check("us", "premium", "api", "clothing")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("post-reload check: %v\n", ok)
}
