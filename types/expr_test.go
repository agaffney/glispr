package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestExpressionType(t *testing.T) {
	testDefs := []struct {
		expr types.Expression
		want types.ExpressionType
	}{
		{types.NewList(nil), types.ExpressionTypeList},
		{types.NewLiteral(types.NewSymbol("a")), types.ExpressionTypeLiteral},
		{types.NewString("a"), types.ExpressionTypeString},
		{types.NewNumberInteger(big.NewInt(1)), types.ExpressionTypeNumber},
		{types.NewSymbol("a"), types.ExpressionTypeSymbol},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, testDef.expr.Type(), "wrong type for %s", testDef.expr.String())
	}
}

func TestExpressionString(t *testing.T) {
	testDefs := []struct {
		expr types.Expression
		want string
	}{
		{types.NewList(nil), `()`},
		{types.NewList([]types.Expression{types.NewSymbol("foo")}), `(|foo|)`},
		{
			types.NewList([]types.Expression{
				types.NewSymbol("foo"),
				types.NewString("bar"),
				types.NewNumberInteger(big.NewInt(1)),
				types.NewList([]types.Expression{types.NewNumberRatio(big.NewRat(1, 2))}),
			}),
			`(|foo| "bar" 1 (1/2))`,
		},
		{types.NewLiteral(types.NewSymbol("foo")), `'|foo|`},
		{types.NewLiteral(types.NewList(nil)), `'()`},
		{types.NewString("foo bar"), `"foo bar"`},
		{types.NewSymbol("foo"), `|foo|`},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, testDef.expr.String())
	}
}

func TestListItemsAreCopied(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")
	input := []types.Expression{foo}
	l := types.NewList(input)

	// Changing the slice passed to NewList must not affect the list
	input[0] = bar
	assert.Equal(t, []types.Expression{foo}, l.Items())

	// Changing the slice returned by Items must not affect the list
	l.Items()[0] = bar
	assert.Equal(t, []types.Expression{foo}, l.Items())
}

func TestAccessors(t *testing.T) {
	sym := types.NewSymbol("foo")
	assert.Equal(t, "foo", sym.Name())
	assert.Equal(t, "foo", types.NewString("foo").Value())
	assert.Same(t, sym, types.NewLiteral(sym).Item())
}

func TestNumberAccessors(t *testing.T) {
	i := types.NewNumberInteger(big.NewInt(5))
	assert.Equal(t, types.NumberTypeInteger, i.NumType())
	assert.Equal(t, 0, i.Int().Cmp(big.NewInt(5)))

	f := types.NewNumberFloat(big.NewFloat(1.5))
	assert.Equal(t, types.NumberTypeFloat, f.NumType())
	assert.Equal(t, 0, f.Float().Cmp(big.NewFloat(1.5)))

	r := types.NewNumberRatio(big.NewRat(1, 3))
	assert.Equal(t, types.NumberTypeRatio, r.NumType())
	assert.Equal(t, 0, r.Rat().Cmp(big.NewRat(1, 3)))
}

func TestNumberConstructorsCopyInput(t *testing.T) {
	in := big.NewInt(5)
	n := types.NewNumberInteger(in)
	in.SetInt64(9)
	assert.Equal(t, "5", n.String())
}

func TestNumberEqual(t *testing.T) {
	one := types.NewNumberInteger(big.NewInt(1))
	oneFloat := types.NewNumberFloat(big.NewFloat(1))
	oneRatio := types.NewNumberRatio(big.NewRat(1, 1))

	assert.True(t, one.Equal(types.NewNumberInteger(big.NewInt(1))))
	assert.False(t, one.Equal(types.NewNumberInteger(big.NewInt(2))))
	assert.True(t, oneFloat.Equal(types.NewNumberFloat(big.NewFloat(1))))
	assert.False(t, oneFloat.Equal(types.NewNumberFloat(big.NewFloat(2))))
	assert.True(t, oneRatio.Equal(types.NewNumberRatio(big.NewRat(2, 2))))
	assert.False(t, oneRatio.Equal(types.NewNumberRatio(big.NewRat(1, 2))))

	// Numbers of different types are never equal, even with the same value
	assert.False(t, one.Equal(oneFloat))
	assert.False(t, one.Equal(oneRatio))
	assert.False(t, oneFloat.Equal(oneRatio))

	// A zero-value Number has no type and is never equal
	assert.False(t, (&types.Number{}).Equal(&types.Number{}))
}

func TestNumberString(t *testing.T) {
	assert.Equal(t, "-42", types.NewNumberInteger(big.NewInt(-42)).String())
	assert.Equal(t, "1.5", types.NewNumberFloat(big.NewFloat(1.5)).String())
	assert.Equal(t, "2/3", types.NewNumberRatio(big.NewRat(4, 6)).String())
	assert.Equal(t, "<unknown>", (&types.Number{}).String())
}
