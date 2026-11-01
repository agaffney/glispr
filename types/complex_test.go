package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func intNum(n int64) *types.Number { return types.NewNumberInteger(big.NewInt(n)) }

func ratNum(a, b int64) *types.Number { return types.NewNumberRatio(big.NewRat(a, b)) }

func singleNum(f float64) *types.Number {
	return types.NewNumberFloat(big.NewFloat(f), types.FloatFormatSingle)
}

func doubleNum(f float64) *types.Number {
	return types.NewNumberFloat(big.NewFloat(f), types.FloatFormatDouble)
}

func mustComplex(t *testing.T, re, im *types.Number) *types.Number {
	t.Helper()
	c, err := types.NewNumberComplex(re, im)
	assert.NoError(t, err)
	return c
}

func TestNumberComplex(t *testing.T) {
	c := mustComplex(t, intNum(1), intNum(2))
	assert.Equal(t, types.NumberTypeComplex, c.NumType())
	assert.Equal(t, types.ExpressionTypeNumber, c.Type())
	assert.True(t, c.Real().Equal(intNum(1)))
	assert.True(t, c.Imag().Equal(intNum(2)))
	assert.Equal(t, "#C(1 2)", c.String())
	// The other accessors have nothing for it
	assert.Nil(t, c.Int())
	assert.Nil(t, c.Float())
	assert.Nil(t, c.Rat())
	assert.Equal(t, types.FloatFormatNone, c.Format())

	// And a complex number is not there for the others
	assert.Nil(t, intNum(1).Real())
	assert.Nil(t, intNum(1).Imag())
	assert.Nil(t, singleNum(1).Real())

	testDefs := []struct {
		name   string
		re, im *types.Number
		// want is the printed form of the result
		want string
	}{
		{"integers", intNum(3), intNum(-4), "#C(3 -4)"},
		{"ratios", ratNum(1, 2), ratNum(-3, 4), "#C(1/2 -3/4)"},
		{"integer and ratio", intNum(1), ratNum(1, 2), "#C(1 1/2)"},
		{"zero real part", intNum(0), intNum(1), "#C(0 1)"},
		// A zero imaginary part of a rational makes it a real
		{"integer zero imaginary", intNum(5), intNum(0), "5"},
		{"ratio with zero imaginary", ratNum(1, 2), intNum(0), "1/2"},
		{"zero both", intNum(0), intNum(0), "0"},
		// Floats keep it complex, and the parts are floats of the same format
		{"singles", singleNum(1.5), singleNum(-2.5), "#C(1.5f0 -2.5f0)"},
		{"doubles", doubleNum(1.5), doubleNum(-2.5), "#C(1.5d0 -2.5d0)"},
		{"float zero imaginary", singleNum(5), singleNum(0), "#C(5.0f0 0.0f0)"},
		{"integer and single", intNum(1), singleNum(2.5), "#C(1.0f0 2.5f0)"},
		{"single and integer", singleNum(2.5), intNum(3), "#C(2.5f0 3.0f0)"},
		{"ratio and double", ratNum(1, 4), doubleNum(2), "#C(0.25d0 2.0d0)"},
		{"zero integer and single", intNum(7), singleNum(0), "#C(7.0f0 0.0f0)"},
		{"integer zero and float", intNum(0), singleNum(1), "#C(0.0f0 1.0f0)"},
		// The larger format wins
		{"single and double", singleNum(1.5), doubleNum(2.5), "#C(1.5d0 2.5d0)"},
		{"double and single", doubleNum(1.5), singleNum(2.5), "#C(1.5d0 2.5d0)"},
		{"single, double, zero", singleNum(1), doubleNum(0), "#C(1.0d0 0.0d0)"},
	}
	for _, tc := range testDefs {
		got := mustComplex(t, tc.re, tc.im)
		assert.Equal(t, tc.want, got.String(), tc.name)
	}

	// A real is returned as it is
	re := intNum(5)
	assert.Same(t, re, mustComplex(t, re, intNum(0)))

	// A single widened to a double is the same number
	got := mustComplex(t, singleNum(0.1), doubleNum(1))
	assert.True(t, got.Real().Equal(types.NewNumberFloat(singleNum(0.1).Float(), types.FloatFormatDouble)))
	assert.Equal(t, types.FloatFormatDouble, got.Real().Format())
	assert.Equal(t, types.FloatFormatDouble, got.Imag().Format())
}

func TestNumberComplexConversionRange(t *testing.T) {
	huge := types.NewNumberInteger(new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil))
	tiny := types.NewNumberRatio(new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(60), nil)))

	// Too large or small for a single, which is the format the other part has
	for _, part := range []*types.Number{huge, tiny} {
		c, err := types.NewNumberComplex(part, singleNum(1))
		assert.ErrorIs(t, err, types.ErrFloatRange)
		assert.Nil(t, c)
		c, err = types.NewNumberComplex(singleNum(1), part)
		assert.ErrorIs(t, err, types.ErrFloatRange)
		assert.Nil(t, c)
	}
	// But fine for a double
	c, err := types.NewNumberComplex(huge, doubleNum(1))
	assert.NoError(t, err)
	assert.Equal(t, types.FloatFormatDouble, c.Real().Format())

	// And beyond a double, which is too large
	enormous := types.NewNumberInteger(new(big.Int).Exp(big.NewInt(10), big.NewInt(400), nil))
	_, err = types.NewNumberComplex(enormous, doubleNum(1))
	assert.ErrorIs(t, err, types.ErrFloatRange)

	// Rationals alone are never converted, so they can be any size
	c, err = types.NewNumberComplex(enormous, intNum(1))
	assert.NoError(t, err)
	assert.Equal(t, types.NumberTypeComplex, c.NumType())
}

func TestNumberComplexPanics(t *testing.T) {
	c := mustComplex(t, intNum(1), intNum(2))
	assert.Panics(t, func() { _, _ = types.NewNumberComplex(nil, intNum(1)) })
	assert.Panics(t, func() { _, _ = types.NewNumberComplex(intNum(1), nil) })
	assert.Panics(t, func() { _, _ = types.NewNumberComplex(c, intNum(1)) })
	assert.Panics(t, func() { _, _ = types.NewNumberComplex(intNum(1), c) })
}

func TestNumberComplexEqual(t *testing.T) {
	a := mustComplex(t, intNum(1), intNum(2))
	assert.True(t, a.Equal(mustComplex(t, intNum(1), intNum(2))))
	assert.False(t, a.Equal(mustComplex(t, intNum(1), intNum(3))))
	assert.False(t, a.Equal(mustComplex(t, intNum(2), intNum(2))))
	// The type of the parts counts, as it does for the reals
	assert.False(t, a.Equal(mustComplex(t, singleNum(1), singleNum(2))))
	assert.False(t, mustComplex(t, singleNum(1), singleNum(2)).Equal(mustComplex(t, doubleNum(1), doubleNum(2))))
	assert.True(t, mustComplex(t, singleNum(1), singleNum(2)).Equal(mustComplex(t, singleNum(1), singleNum(2))))
	// And a complex number is never a real
	assert.False(t, a.Equal(intNum(1)))
	assert.False(t, intNum(1).Equal(a))
	assert.False(t, a.Equal(ratNum(1, 2)))
}

func TestFormatComplex(t *testing.T) {
	ps := types.NewPackages()
	g, _ := ps.User().Intern("G")
	c := mustComplex(t, intNum(10), ratNum(1, 2))
	list := types.NewList([]types.Expression{g, c})
	assert.Equal(t, "(G #C(10 1/2))", types.Format(list, ps.User()))
	// In another base the parts are written so that they read the same
	assert.Equal(t, "(G #C(10. #10r1/2))", types.FormatBase(list, ps.User(), 16))
	assert.Equal(t, "#C(1.5f0 2.0f0)", types.FormatBase(mustComplex(t, singleNum(1.5), intNum(2)), ps.User(), 16))
	assert.Equal(t, "#(#C(1 2))", types.Format(types.NewVector([]types.Expression{mustComplex(t, intNum(1), intNum(2))}), nil))
}
