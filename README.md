# gorege

A small Go library for **first-match rule evaluation** over a fixed tuple of dimensions: access control, feature flags, A/B cohorts, product availability, and similar decisions all map to the same pattern.

Design goals: idiomatic Go, immutable engines safe for concurrent use, explicit semantics (including `Explain` and dead/shadow rule warnings), and a true BFS-based `Closest` search for minimum Hamming distance. The API is influenced by [recht](https://github.com/dashersw/recht); gorege adds stronger guarantees and observability.

- **Go 1.26+**
- **Zero runtime dependencies** (standard library only)
- **JSON** configuration via `Load` / `LoadWithOptions` / `LoadFileWithOptions` (`.json` only)

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

`Check` requires exactly as many arguments as dimensions (`ErrArityMismatch` otherwise). `PartialCheck` allows a prefix tuple, including **zero** values (empty prefix: “could any full tuple still be allowed?”), with Recht-style trailing “unconstrained” behaviour. It returns `(bool, error)`; if you pass **more** values than dimensions you get `ErrArityMismatch` instead of a bare `false`, so overload is not mistaken for denial.

## JSON config

`LoadFileWithOptions`, `Load`, and `LoadWithOptions` decode the same schema (call `LoadFileWithOptions(path)` with no extra options for a plain file load). Extra options (for example `WithAnalysisLimit`) apply after the JSON-derived dimensions and rules. Example (see also `testdata/rules.json`):

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

On `New`, `Load`, `LoadWithOptions`, or `LoadFileWithOptions`, the engine reports **warnings** for rules that never match any tuple in the Cartesian product (“dead”) or never win first-match (“shadowed”), unless analysis is skipped (see below). Dead detection does not enumerate the product; shadow detection does, subject to a tuple cap. Each `Warning` includes `Kind` (`WarningKindDead`, `WarningKindShadowed`, or `WarningKindAnalysisLimitExceeded`) so callers need not parse `Message`.

> **Performance note:** Shadowed-rule analysis walks the Cartesian product of declared dimension values. With large dimension sets (e.g. 6 dimensions × 20 values = 64 000 000 tuples) this can be slow. The default cap is 100 000 tuples for that pass; use `WithAnalysisLimit(n)` with `New` or `LoadWithOptions` / `LoadFileWithOptions` to adjust, or pass a negative value to skip analysis entirely. When the cap is exceeded, dead rules are still reported.

## Bring Your Own Parser

`gorege.Config`, `gorege.DimensionConfig`, and `gorege.RuleConfig` are exported. To build an engine from YAML, TOML, or another format, decode into these types with your own parser, then call `NewFromConfig`:

```go
import "gopkg.in/yaml.v3" // in your project; not a gorege dependency

var cfg gorege.Config
if err := yaml.Unmarshal(data, &cfg); err != nil { ... }

e, warnings, err := gorege.NewFromConfig(cfg,
    gorege.WithAnalysisLimit(50_000),
)
```

`gorege` uses only `encoding/json` internally. The `yaml:"..."` struct tags let you unmarshal with any YAML library on your side while keeping gorege a zero third-party dependency as a library.

## API overview

| Area | Functions |
|------|-----------|
| Build | `New`, `NewFromConfig`, `WithDimensions`, `WithRules`, `WithTiebreak`, `WithAnalysisLimit` (shadow analysis tuple cap, default 100 000) |
| Inspect | `Dimensions`, `Rules` (defensive copies) |
| Evaluate | `Check`, `PartialCheck`, `Explain` |
| Nearest allow | `Closest` — BFS by Hamming distance from the input; **any** dimensions may change until an allowed tuple is found. `ClosestIn` — **only** the selected dimension changes (others fixed); `dim` is an index or dimension name. Tiebreak (`WithTiebreak`): leftmost / rightmost / declaration order affects `Closest` search and reporting. |
| Config | `LoadFileWithOptions`, `Load`, `LoadWithOptions` (`.json` only) |
| Types | `Dimension`, `Rule`, `Action`, `Explanation`, `ClosestResult`, `Warning`, `WarningKind`, `Config`, `DimensionConfig`, `RuleConfig` |

`Engine` is immutable and safe to share. For hot reload, load a new engine and swap a `sync/atomic.Pointer` holding `*gorege.Engine`.

## CLI

```bash
go install github.com/yplog/gorege/cmd/gorege@latest
# or: task build-cli  → ./bin/gorege

gorege check path/to/rules.json Guest Wed Sauna   # prints true/false; exit 1 if denied or error
gorege partial-check path/to/rules.json Guest     # prefix [Engine.PartialCheck]: 0..N values (N = #dims); true if some completion could still be allowed
gorege explain path/to/rules.json Guest Wed Sauna # which rule matched (debug); exit 1 on load/arity error only
gorege closest path/to/rules.json Guest Wed Sauna # nearest allowed tuple (BFS); exit 1 if none exists
gorege closest-in path/to/rules.json 2 Guest Wed Sauna   # same, varying only dim index 2
gorege closest-in path/to/rules.json facility Guest Wed Sauna # or dimension name
gorege lint path/to/rules.json                    # dead/shadow warnings (or "ok"); exit 1 if any warnings
```

**Where loader warnings go:** For `check`, `explain`, `partial-check`, `closest`, and `closest-in`, the main result is on **stdout** (for example `true`/`false` or `explain` fields), so engine load warnings (dead rules, shadowed rules, analysis limit, …) are printed to **stderr** as secondary output. **`lint`** is the opposite: those warnings *are* the intended output, so each message is printed to **stdout** (or `ok` when there are none), which keeps `lint` easy to pipe or scrape; load errors still go to **stderr**.

`explain` prints `matched`, `allowed`, `rule_index`, `rule_name`, and `action` (or a line for implicit deny when no rule matches). Exit code stays `0` when the explanation was computed successfully.

`closest` walks increasing Hamming distance and may change several dimensions at once (`Engine.Closest`). `closest-in` only tries alternate values on one axis (`Engine.ClosestIn`). Both print `found`, `conditions` (JSON array), `distance` (Hamming distance from the input tuple), `dim_index`, `dim_name`, and `value` for the reported pivot dimension. `found: false` uses exit code `1`. For `closest-in`, a numeric-only selector is treated as a dimension index; otherwise it is resolved as a name (same as the library).

## Development

This repo uses **[mise](https://mise.jdx.dev/)** for pinned Go (see `mise.toml`) and **[Task](https://taskfile.dev/)** for common commands:

| Task | Purpose |
|------|---------|
| `task` / `task test` | Unit tests |
| `task cover` | Coverage (profile + merged summary line) |
| `task build-cli` | Build `bin/gorege` |
| `task ci` | `gofmt`, `vet`, `test`, `build` |
| `task fuzz-load` / `task fuzz-check` | Go fuzz (default 5s; e.g. `task fuzz-load FUZZTIME=30s`) |

Fuzz targets live in `fuzz_test.go`. Normal `go test` runs each fuzz function once with its seed corpus; use `-fuzz=FuzzLoad` (etc.) for real fuzzing.

## Layout

```
gorege.go    Engine, New, options
rule.go      Rules, matchers, Allow/Deny
dimension.go Dimensions
check.go     Check, PartialCheck, Explain
closest.go   Closest, ClosestIn, tiebreak
conflict.go  Dead / shadow warnings
loader.go    Config types, NewFromConfig, JSON Load / LoadFileWithOptions
result.go    Explanation, ClosestResult, Action helpers
cmd/gorege   CLI
fuzz_test.go Go fuzz targets (Load, Check, …)
testdata/    Example JSON fixtures
```
