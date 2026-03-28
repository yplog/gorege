package gorege

import "errors"

// DefaultAnalysisLimit is the default upper bound on the number of dimension
// tuples enumerated for shadowed-rule analysis in [New]. Dead-rule detection
// does not use this cap. Use [WithAnalysisLimit] to change the threshold.
const DefaultAnalysisLimit = 100_000

var (
	// ErrArityMismatch is returned by [Engine.Check] when the number of
	// arguments does not equal the number of dimensions.
	ErrArityMismatch = errors.New("gorege: argument count must match dimension count")

	// ErrRuleTooWide is returned by [New] when a rule has more matchers than dimensions.
	ErrRuleTooWide = errors.New("gorege: rule has more matchers than dimensions")

	// ErrUnknownDimensionValue is returned by [New] when a matcher references a
	// value not present in the corresponding dimension declaration.
	ErrUnknownDimensionValue = errors.New("gorege: matcher references unknown dimension value")

	// ErrInvalidDimension is returned by [Engine.ClosestIn] when dim is not a
	// valid index (int-sized signed/unsigned types) or a known dimension name.
	ErrInvalidDimension = errors.New("gorege: invalid dimension selector")

	// ErrUnsupportedConfigFormat is returned by [LoadFileWithOptions] when the
	// path does not end in .json.
	ErrUnsupportedConfigFormat = errors.New("gorege: unsupported config format (use .json)")
)
