package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestBitVector(t *testing.T) {
	v := types.NewBitVector([]byte{1, 0, 1, 1})
	assert.Equal(t, types.ExpressionTypeBitVector, v.Type())
	assert.Equal(t, 4, v.Len())
	assert.Equal(t, byte(1), v.At(0))
	assert.Equal(t, byte(0), v.At(1))
	assert.Equal(t, []byte{1, 0, 1, 1}, v.Bits())
	assert.Equal(t, "#*1011", v.String())
	assert.Panics(t, func() { v.At(4) })
	assert.Panics(t, func() { v.At(-1) })

	empty := types.NewBitVector(nil)
	assert.Equal(t, 0, empty.Len())
	assert.Empty(t, empty.Bits())
	assert.Equal(t, "#*", empty.String())
	// However it is made, an empty vector is equal to another one
	assert.Equal(t, empty, types.NewBitVector([]byte{}))

	assert.Equal(t, types.NewBitVector([]byte{0, 1}), types.NewBitVector([]byte{0, 1}))
	assert.NotEqual(t, types.NewBitVector([]byte{0, 1}), types.NewBitVector([]byte{1, 0}))

	for _, bit := range []byte{2, 3, '0', '1', 255} {
		assert.Panics(t, func() { types.NewBitVector([]byte{0, bit}) }, "bit %d", bit)
	}
}

func TestBitVectorBitsAreCopied(t *testing.T) {
	input := []byte{1, 0}
	v := types.NewBitVector(input)
	input[0] = 0
	assert.Equal(t, []byte{1, 0}, v.Bits())

	out := v.Bits()
	out[1] = 1
	assert.Equal(t, []byte{1, 0}, v.Bits())
}

func TestFormatBitVector(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.User().Intern("A")
	v := types.NewBitVector([]byte{1, 0})
	assert.Equal(t, "(A #*10 #*)", types.Format(types.NewList([]types.Expression{a, v, types.NewBitVector(nil)}), ps.User()))
	assert.Equal(t, "#(#*10)", types.Format(types.NewVector([]types.Expression{v}), ps.User()))
	// It is not labeled when it is shared, as it has nothing to refer to
	assert.Equal(t, "(#*10 #*10)", types.Format(types.NewList([]types.Expression{v, v}), ps.User()))
}
