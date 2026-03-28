# gorege

A small Go library for **first-match rule evaluation** over a fixed tuple of dimensions: access control, feature flags, A/B cohorts, product availability, and similar decisions all map to the same pattern.

Design goals: idiomatic Go, immutable engines safe for concurrent use, explicit semantics (including `Explain` and dead/shadow rule warnings), and a true BFS-based `Closest` search for minimum Hamming distance. The API is influenced by [recht](https://github.com/dashersw/recht); gorege adds stronger guarantees and observability.

- **Go 1.21+**
- **Zero runtime dependencies** (standard library only)
- **JSON** configuration via `Load` / `LoadFile` (`.json` only)

## Install

```bash
go get github.com/yplog/gorege
```

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/yplog/gorege"
)

func main() {
	e, warnings, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("membership", "Gold member", "Regular member", "Guest"),
			gorege.Dim("day", "Mon", "Tue", "Wed", "Thu", "Fri"),
			gorege.Dim("facility", "Swimming pool", "Gym", "Sauna"),
		),
		gorege.WithRules(
			gorege.Allow("Gold member", gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("Guest", gorege.AnyOf("Mon", "Tue"), "Sauna"),
			gorege.Allow(gorege.AnyOf("Guest", "Regular member"), gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, w := range warnings {
		log.Printf("rule warning: %s", w)
	}

	ok, err := e.Check("Guest", "Mon", "Sauna")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ok) // false

	ok, err = e.Check("Guest", "Wed", "Sauna")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ok) // true
}
```

### Rule shape

Each rule is **ALLOW** or **DENY** plus one matcher per dimension (in order):

| Matcher in code | Meaning |
|-----------------|--------|
| `"exact"` string | Exact value |
| `gorege.AnyOf("a", "b")` | Any listed value |
| `gorege.Wildcard` | Any value **declared** for that dimension |

Evaluation is **first match wins**. If nothing matches, `Check` returns `false`. Shorter rules implicitly wildcard trailing dimensions.

`Check` requires exactly as many arguments as dimensions (`ErrArityMismatch` otherwise). `PartialCheck` allows a prefix tuple (Recht-style trailing “unconstrained” behaviour) and returns `(bool, error)`: if you pass **more** values than dimensions you get `ErrArityMismatch` instead of a bare `false`, so overload is not mistaken for denial.

## JSON config

`LoadFile("rules.json")` and `Load(io.Reader)` decode the same schema. Example (see also `testdata/rules.json`):

```json
{
  "dimensions": [
    { "name": "membership", "values": ["Gold member", "Regular member", "Guest"] },
    { "name": "day", "values": ["Mon", "Tue", "Wed", "Thu", "Fri"] },
    { "name": "facility", "values": ["Swimming pool", "Gym", "Sauna"] }
  ],
  "rules": [
    { "action": "ALLOW", "name": "allow-gold", "conditions": ["Gold member", "*", "*"] },
    { "action": "DENY", "name": "deny-guest-sauna-early-week", "conditions": ["Guest", ["Mon", "Tue"], "Sauna"] },
    { "action": "ALLOW", "name": "allow-rest", "conditions": [["Guest", "Regular member"], "*", "*"] }
  ]
}
```

- `"*"` in JSON is a wildcard (same as `gorege.Wildcard`).
- A JSON array of strings in a slot is `AnyOf`.
- Omit `name` on a dimension to get an anonymous axis (`DimValues`-style).

On `New` / `Load`, the engine reports **warnings** for rules that never match any tuple in the Cartesian product (“dead”) or never win first-match (“shadowed”).

## API overview

| Area | Functions |
|------|-----------|
| Build | `New`, `WithDimensions`, `WithRules`, `WithTiebreak` |
| Evaluate | `Check`, `PartialCheck`, `Explain` |
| Nearest allow | `Closest`, `ClosestIn` (tiebreak: leftmost / rightmost / decl order) |
| Config | `LoadFile` (`.json` only), `Load` |
| Types | `Dimension`, `Rule`, `Action`, `Explanation`, `ClosestResult`, `Warning` |

`Engine` is immutable and safe to share. For hot reload, load a new engine and swap a `sync/atomic.Pointer` holding `*gorege.Engine`.

## CLI

```bash
go install github.com/yplog/gorege/cmd/gorege@latest
# or: task build-cli  → ./bin/gorege

gorege check path/to/rules.json Guest Wed Sauna   # prints true/false; exit 1 if denied or error
gorege explain path/to/rules.json Guest Wed Sauna # which rule matched (debug); exit 1 on load/arity error only
gorege lint path/to/rules.json                    # prints warnings; exit 1 if any
```

`explain` prints `matched`, `allowed`, `rule_index`, `rule_name`, and `action` (or a line for implicit deny when no rule matches). Exit code stays `0` when the explanation was computed successfully.

## Development

This repo uses **[mise](https://mise.jdx.dev/)** for pinned Go (see `mise.toml`) and **[Task](https://taskfile.dev/)** for common commands:

| Task | Purpose |
|------|---------|
| `task` / `task test` | Unit tests |
| `task cover` | Coverage (profile + merged summary line) |
| `task build-cli` | Build `bin/gorege` |
| `task ci` | `gofmt`, `vet`, `test`, `build` |

## Layout

```
gorege.go    Engine, New, options
rule.go      Rules, matchers, Allow/Deny
dimension.go Dimensions
check.go     Check, PartialCheck, Explain
closest.go   Closest, ClosestIn, tiebreak
conflict.go  Dead / shadow warnings
loader.go    JSON Load / LoadFile
result.go    Explanation, ClosestResult, Action helpers
cmd/gorege   CLI
testdata/    Example JSON fixtures
```
