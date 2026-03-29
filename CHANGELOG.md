# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.2.0]: https://github.com/yplog/gorege/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/yplog/gorege/releases/tag/v0.1.1