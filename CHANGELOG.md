# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.5.0] - 2026-04-25

### Performance

- **`ClosestIn` allocation reduction** — the working candidate slice is now
  managed via `curPool` (the same `sync.Pool` used by `searchSubset`),
  eliminating one heap allocation per call when `len(values) <= 16`.
  `Distance` in the returned `ClosestResult` is now assigned the constant `1`
  instead of calling `hammingDistance`; `ClosestIn` always changes exactly one
  dimension by construction, so the O(n) scan was redundant.
  `Closest` (D=1–3) is unaffected — no change to `searchSubset` or
  `searchSubsetDFS`.

  Measured on Apple M2 Pro, darwin/arm64, Go 1.26, `count=6` with `benchstat`,
  compared against v0.4.0:

  | Benchmark | v0.4.0 | v0.5.0 | Δ ns/op | Δ B/op | Δ allocs |
  |-----------|--------|--------|---------|--------|----------|
  | `ClosestIn` — by name  | 177.8 ns / 176 B / 3 allocs | 157.9 ns / 128 B / 2 allocs | −11.2% | −27% | −1 |
  | `ClosestIn` — by index | 175.7 ns / 176 B / 3 allocs | 154.5 ns / 128 B / 2 allocs | −12.0% | −27% | −1 |

## [0.4.0] - 2026-03-31

### Fixed

- **Trie wildcard semantics** — `Check` (and `Explain`) now correctly reject
  undeclared dimension values when a rule uses `Wildcard`. Previously, when the
  trie path was active, a `Wildcard` matcher at depth D would accept *any* input
  string at that position, including values not declared for that dimension.
  The linear scan was always correct (`matcher.matches` calls `dim.contains`);
  the trie `search` method lacked the equivalent guard. The fix passes `dims`
  through `search` and applies the same `dim.contains(input)` check before
  descending into the wildcard child.

  This bug was latent under v0.3.0 because the trie only activated for N > 150
  rules; tests and the fuzz target (`FuzzTrieVsLinear`) operated on small
  engines where the linear path was always taken. The always-trie change in
  this release made the bug reachable with any rule count and surfaced it
  immediately.

### Performance

- **Always-trie** — the `trieThreshold = 150` guard is removed. The Priority
  Multi-path Trie now activates for every engine that has at least one dimension
  and one rule. Benchmarks show the trie outperforms linear scan at all measured
  N (crossover is below N = 10); `New()` overhead at N = 3 is +183 ns —
  recovered after ~10 `Check()` calls. Zero-allocation hot path preserved.
  Overall geomean across the full benchmark suite: **−19.0%**.

- **`Closest` indirect speedup** — `Closest` D=2 −29%, D=3 −44% vs v0.3.0.
  `searchSubsetDFS` calls `Check()` internally; the always-trie speedup
  propagates through BFS candidate evaluation automatically.

- **Regressions** — `New` construction overhead increases for small engines
  previously below the threshold: `New_SkipAnalysis` (gym, 3 rules) +22%
  latency / +34% allocs; `New_WithAnalysis` +10% / +31% allocs. Expected cost
  of always-trie; recovered after ~10 `Check()` calls. High-dimension,
  low-rule-count engines (`Scale_Dims` D=8, D=12) show +5% due to trie depth
  traversal exceeding linear scan cost for N=1; typical production configs
  (multiple rules, 3–20 values per dimension) are unaffected.

## [0.3.0] - 2026-03-29

### Performance

`Check` hot path gains a **Priority Multi-path Trie** (`trie.go`) that activates
automatically when an engine has more than 150 rules and at least one declared
dimension. Below that threshold the existing linear scan is used unchanged. No
API or configuration changes are required.

Measured on Apple M2 Pro, darwin/arm64, Go 1.26, `count=6` with `benchstat`,
compared against v0.2.1.

#### Rule scaling (`Check` at N=10…1000 rules)

| N | v0.2.1 | v0.3.0 | Δ | Path |
|---|--------|--------|---|------|
| 10 | 92.8 ns | 92.9 ns | ~ | linear |
| 100 | 761.5 ns | 762.6 ns | ~ | linear |
| 150 | 1140 ns | 1133 ns | ~ | linear (threshold boundary) |
| 200 | 1532 ns | 11.8 ns | −99.2% | trie |
| 500 | 3723 ns | 11.7 ns | −99.7% | trie |
| 1000 | 7342 ns | 11.7 ns | −99.8% | trie |

Zero allocations on all rows preserved.

> **Benchmark note:** the rule-scale fixture uses a single dimension with N
> distinct exact values — a pathological case where every rule occupies a
> unique leaf. In typical gorege configurations each dimension has 3–20 declared
> values; the trie children per node are bounded by that count, not by the rule
> count, so the O(D) characteristic holds in practice.

#### Hot path and other methods

All hot-path benchmarks (`Check`, `Explain`, `PartialCheck`, `Closest`,
`ClosestIn`) are within noise of v0.2.1. Zero-allocation guarantees are
unchanged. Overall geomean across the full benchmark suite: **−22.3%**.

`New` (skip analysis) shows +1.2% due to trie construction; `New` with analysis
is within noise.

### Changed (internals)

- **Priority Multi-path Trie** (`trie.go`) — `ruleTrieNode` stores per-node
  `minRuleIdx` (the smallest reachable rule index in that subtree) enabling
  subtree pruning during DFS search. `AnyOf` matchers expand into multiple child
  paths at insert time. Children are stored as a `[]trieEntry` slice up to 16
  entries (cache-friendly linear scan); beyond that the slice is promoted to a
  `map[string]*ruleTrieNode` in-place, keeping lookup O(1) for large dimension
  value sets.

- **`eval` routing** — `Check` and `Explain` route through the trie when
  `trieRoot != nil`. `PartialCheck` always uses the linear scan: its
  trailing-dimension `unconstrainedMatch` semantics (DENY rules return `false`
  for unconstrained positions) cannot be represented in the trie without
  significant added complexity, and `PartialCheck` is not a hot path.

- **Differential fuzz coverage** — `FuzzTrieVsLinear` in
  `gorege_internal_test.go` verifies that trie search and linear scan produce
  identical results across randomly generated rule sets with mixed `Exact`,
  `Wildcard`, and `AnyOf` matchers (113 interesting inputs found, zero
  mismatches in 60 s runs).

---

## [0.2.1] - 2026-03-29

### Performance

- **`Closest` allocations** — k-combination subsets are iterated in-place (`nextCombo`
  for left/declaration tiebreak; recursive `trySubsetsRightmost` for rightmost,
  matching the former sort order). `searchSubset` uses a package-level DFS with
  explicit `sync.Pool` release (no closures). `buildClosestResult` uses stack
  buffers when `len(input) ≤ maxDims`, with a heap fallback for larger arity.
  `BenchmarkClosest` (2-dim deny/allow fixture, darwin/arm64, Go 1.26): ~344 B /
  11 allocs/op → ~112 B / 2 allocs/op; latency ~344 ns → ~109 ns in local runs.

---

## [0.2.0] - 2026-03-29

### Performance

This release is focused entirely on performance. The public API is unchanged;
all existing code compiles and behaves identically.

Measured on Apple M2 Pro, darwin/arm64, Go 1.26, `count=6` with `benchstat`.

#### Hot path (`Check`, `Explain`, `PartialCheck`)

| Benchmark | v0.1.1 | v0.2.0 | Δ |
|-----------|--------|--------|---|
| `Check` — allow | 71.2 ns | 69.3 ns | −2.6% |
| `Check` — deny | 51.5 ns | 45.2 ns | −12.3% |
| `Check` — parallel | 10.8 ns | 9.6 ns | −10.8% |
| `Check` — best case | 38.4 ns | 35.8 ns | −6.8% |
| `Check` — worst case | 22.5 ns | 17.4 ns | −22.5% |
| `Check` — mixed load | 11.4 ns | 10.3 ns | −9.4% |
| `Explain` — allow | 75.7 ns | 68.6 ns | −9.5% |
| `Explain` — deny | 54.4 ns | 44.4 ns | −18.4% |
| `PartialCheck` — empty prefix | 13.3 ns | 6.3 ns | −53.0% |
| `PartialCheck` — 1-value prefix | 48.6 ns | 37.4 ns | −23.0% |
| `PartialCheck` — 2-value prefix | 71.1 ns | 63.3 ns | −11.0% |

Zero allocations on all hot-path methods preserved.

#### Rule scaling (`Check` at N=10…1000 rules)

| N | v0.1.1 | v0.2.0 | Δ |
|---|--------|--------|---|
| 10 | 117.6 ns | 92.4 ns | −21.4% |
| 100 | 1045 ns | 755.9 ns | −27.7% |
| 500 | 5104 ns | 3717 ns | −27.2% |
| 1000 | 10120 ns | 7341 ns | −27.5% |

The linear scaling characteristic is preserved; only the per-rule constant improved.

#### Dimension scaling (`Check` at D=1…12 dimensions)

| D | v0.1.1 | v0.2.0 | Δ |
|---|--------|--------|---|
| 1 | 16.2 ns | 14.4 ns | −11.2% |
| 3 | 37.5 ns | 33.2 ns | −11.6% |
| 8 | 93.8 ns | 86.4 ns | −7.9% |
| 12 | 136.5 ns | 123.2 ns | −9.8% |

#### `Closest` / `ClosestIn`

| Benchmark | v0.1.1 | v0.2.0 | Δ |
|-----------|--------|--------|---|
| `Closest` D=1 | 362.6 ns / 392 B / 12 allocs | 343.6 ns / 344 B / 11 allocs | −5.2% / −12% / −1 alloc |
| `Closest` — already allowed | 371.2 ns / 392 B / 12 allocs | 343.7 ns / 344 B / 11 allocs | −7.4% / −12% / −1 alloc |
| `ClosestIn` — by name | 252.5 ns | 224.5 ns | −11.1% |
| `ClosestIn` — by index | 245.9 ns | 220.2 ns | −10.5% |

#### Engine construction (`New`)

| Benchmark | v0.1.1 | v0.2.0 | Δ |
|-----------|--------|--------|---|
| `New` — with analysis | 11885 ns / 14672 B / 194 allocs | 6522 ns / 5024 B / 67 allocs | −45.1% / −65.8% / −65.5% |
| `New` — skip analysis | 2861 ns | 2772 ns | −3.1% |

### Changed (internals)

- **`matcher` struct memory layout** — `exact string` and `anyof []string` fields
  merged into a single `vals []string`, reducing struct size from ~48 to ~32 bytes
  per matcher. This improves cache utilization in the `ruleMatches` hot loop.

- **`ruleMatches` hot path simplified** — removed the redundant `dimKnown` branch.
  `validateEngine` already guarantees `len(r.m) <= len(dims)`, so the guard was
  evaluated but never triggered. Removing it allows the compiler to generate
  tighter code for the inner loop.

- **`cartesianProduct` replaced with `walkCartesian`** — shadow analysis no longer
  materializes all tuples as `[][]string` upfront. A callback-based odometer
  walker reuses a single `[]string` buffer, reducing construction-time allocations
  dramatically for large dimension sets (e.g. 5 dims × 10 values: ~40 MB → ~500 B).

- **`Closest` / `ClosestIn` allocation reduction** — `searchSubset` now manages
  the `cur` working slice via `sync.Pool`, and `ClosestIn` reuses a single
  candidate slice across the inner loop instead of allocating per candidate.

---

## [0.1.1] - 2025-03-28

Initial public release.

[0.5.0]: https://github.com/yplog/gorege/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/yplog/gorege/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/yplog/gorege/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/yplog/gorege/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/yplog/gorege/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/yplog/gorege/releases/tag/v0.1.1