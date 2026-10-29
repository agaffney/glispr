package types

import "slices"

// Vector is a one-dimensional array of objects.
type Vector struct {
	items []Expression
}

// NewVector returns a vector of items. It panics if any item is nil. The vector
// does not share storage with items.
func NewVector(items []Expression) *Vector {
	if slices.Contains(items, nil) {
		panic("types: NewVector called with a nil item")
	}
	// Never nil, so that equal vectors have equal contents however they were made
	return &Vector{
		items: append(make([]Expression, 0, len(items)), items...),
	}
}

func (Vector) Type() ExpressionType { return ExpressionTypeVector }

// Len returns the number of items in the vector.
func (v Vector) Len() int { return len(v.items) }

// At returns the item at index i. It panics if i is out of range.
func (v Vector) At(i int) Expression { return v.items[i] }

// Items returns a copy of the items in the vector.
func (v Vector) Items() []Expression { return slices.Clone(v.items) }

func (v Vector) String() string { return Format(&v, nil) }
