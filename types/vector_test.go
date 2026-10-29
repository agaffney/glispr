package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestVector(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")
	v := types.NewVector([]types.Expression{foo, bar})

	assert.Equal(t, types.ExpressionTypeVector, v.Type())
	assert.Equal(t, 2, v.Len())
	assert.Same(t, foo, v.At(0))
	assert.Same(t, bar, v.At(1))
	assert.Equal(t, []types.Expression{foo, bar}, v.Items())
	assert.Panics(t, func() { v.At(2) })
	assert.Panics(t, func() { v.At(-1) })

	empty := types.NewVector(nil)
	assert.Equal(t, 0, empty.Len())
	assert.Empty(t, empty.Items())
}

func TestVectorItemsAreCopied(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")
	input := []types.Expression{foo}
	v := types.NewVector(input)

	// Changing the slice passed in must not change the vector
	input[0] = bar
	assert.Equal(t, []types.Expression{foo}, v.Items())

	// Changing the slice returned must not change it either
	v.Items()[0] = bar
	assert.Equal(t, []types.Expression{foo}, v.Items())
}

func TestEmptyVectorsAreEqual(t *testing.T) {
	assert.Equal(t, types.NewVector(nil), types.NewVector([]types.Expression{}))
}

func TestVectorNilItem(t *testing.T) {
	assert.Panics(t, func() { types.NewVector([]types.Expression{types.NewSymbol("a"), nil}) })
}

func TestVectorString(t *testing.T) {
	testDefs := []struct {
		vec  *types.Vector
		want string
	}{
		{types.NewVector(nil), `#()`},
		{types.NewVector([]types.Expression{types.NewSymbol("foo")}), `#(#:|foo|)`},
		{
			types.NewVector([]types.Expression{
				types.NewSymbol("foo"),
				types.NewString("bar"),
				types.NewNumberInteger(big.NewInt(1)),
				types.NewCharacter('a'),
				types.NewList([]types.Expression{types.NewNumberRatio(big.NewRat(1, 2))}),
				types.Nil,
			}),
			`#(#:|foo| "bar" 1 #\a (1/2) ())`,
		},
		{
			types.NewVector([]types.Expression{
				types.NewVector([]types.Expression{types.NewSymbol("a")}),
				types.NewVector(nil),
			}),
			`#(#(#:|a|) #())`,
		},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, testDef.vec.String())
	}
}
