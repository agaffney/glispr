package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestPlaceholder(t *testing.T) {
	ph := types.NewPlaceholder(3)
	assert.Equal(t, types.ExpressionTypePlaceholder, ph.Type())
	assert.Equal(t, "#3#", ph.String())
	assert.NotSame(t, types.NewPlaceholder(3), ph)
}

func TestSubstitute(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.User().Intern("A")
	ph := types.NewPlaceholder(1)

	// (a . #1#) with #1# the list itself
	list := types.NewCons(a, ph)
	types.Substitute(list, ph, list)
	assert.Same(t, list, list.Cdr())

	// #1# in the car, in a vector, and in a nested list
	inner := types.NewCons(ph, types.Nil)
	vec := types.NewVector([]types.Expression{a, ph, inner})
	root := types.NewCons(vec, types.NewCons(ph, types.Nil))
	types.Substitute(root, ph, a)
	assert.Same(t, a, vec.At(1))
	assert.Same(t, a, inner.Car())
	assert.Same(t, a, root.Cdr().(*types.Cons).Car())

	// Each cell is visited once, so shared ones and cycles are fine
	ph = types.NewPlaceholder(2)
	shared := types.NewCons(ph, types.Nil)
	both := types.NewCons(shared, shared)
	cycle := types.NewCons(both, ph)
	types.Substitute(cycle, ph, cycle)
	assert.Same(t, cycle, shared.Car())
	assert.Same(t, cycle, cycle.Cdr())

	// Other placeholders are left, and an atom or Nil root is nothing to do
	other := types.NewPlaceholder(5)
	kept := types.NewCons(other, types.Nil)
	types.Substitute(kept, ph, a)
	assert.Same(t, other, kept.Car())
	types.Substitute(a, ph, a)
	types.Substitute(types.Nil, ph, a)

	assert.Panics(t, func() { types.Substitute(kept, nil, a) })
	assert.Panics(t, func() { types.Substitute(kept, ph, nil) })
	assert.Panics(t, func() { types.Substitute(kept, ph, (*types.Cons)(nil)) })
}

func TestListItemsCircular(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.User().Intern("A")
	b, _ := ps.User().Intern("B")

	for length := 1; length <= 6; length++ {
		// A list of the given length whose last cdr is its first cell, or its
		// second, or a cell in the middle
		for target := 0; target < length; target++ {
			cells := make([]*types.Cons, length)
			ph := types.NewPlaceholder(1)
			for k := length - 1; k >= 0; k-- {
				var next types.Expression = ph
				if k+1 < length {
					next = cells[k+1]
				}
				cells[k] = types.NewCons(a, next)
			}
			types.Substitute(cells[0], ph, cells[target])
			items, ok := types.ListItems(cells[0])
			assert.False(t, ok, "length %d, back to %d", length, target)
			assert.Nil(t, items)
		}
	}

	// A list that shares a tail with another one is proper
	tail := types.NewList([]types.Expression{b})
	items, ok := types.ListItems(types.NewCons(a, types.NewCons(a, tail)))
	assert.True(t, ok)
	assert.Len(t, items, 3)
	// And so is a long one
	long := make([]types.Expression, 100000)
	for k := range long {
		long[k] = a
	}
	items, ok = types.ListItems(types.NewList(long))
	assert.True(t, ok)
	assert.Len(t, items, len(long))
}
