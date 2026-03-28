package gorege

// Dimension describes one axis of the decision tuple. Values are the allowed
// strings for that axis when dimensions are declared.
type Dimension struct {
	name   string
	values []string
	idx    map[string]struct{}
}

// Dim creates a named dimension. The name enables ClosestIn(name, ...) and
// clearer warnings once those features are wired up.
func Dim(name string, values ...string) Dimension {
	if name == "" {
		panic("gorege: Dim name must be non-empty; use DimValues for anonymous dimensions")
	}
	return Dimension{name: name, values: append([]string(nil), values...), idx: indexValues(values)}
}

// DimValues creates an anonymous dimension, addressable only by index.
func DimValues(values ...string) Dimension {
	return Dimension{name: "", values: append([]string(nil), values...), idx: indexValues(values)}
}

// Name returns the dimension name, or empty if anonymous.
func (d Dimension) Name() string {
	return d.name
}

// Values returns a copy of declared values in declaration order.
func (d Dimension) Values() []string {
	return append([]string(nil), d.values...)
}

func indexValues(values []string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}

func (d Dimension) contains(value string) bool {
	if len(d.idx) == 0 {
		return false
	}
	_, ok := d.idx[value]
	return ok
}

func cloneDimensions(dims []Dimension) []Dimension {
	out := make([]Dimension, len(dims))
	for i := range dims {
		out[i] = Dimension{
			name:   dims[i].name,
			values: append([]string(nil), dims[i].values...),
			idx:    indexValues(dims[i].values),
		}
	}
	return out
}
