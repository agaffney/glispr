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
		{types.Nil, types.ExpressionTypeNull},
		{types.NewCons(types.Nil, types.Nil), types.ExpressionTypeCons},
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
		{
			types.NewCons(types.NewSymbol("a"), types.NewSymbol("b")),
			`(|a| . |b|)`,
		},
		{
			types.NewCons(types.NewSymbol("a"), types.NewCons(types.NewSymbol("b"), types.NewSymbol("c"))),
			`(|a| |b| . |c|)`,
		},
		{
			types.NewList([]types.Expression{
				types.NewList([]types.Expression{types.NewSymbol("a")}),
				types.NewSymbol("b"),
			}),
			`((|a|) |b|)`,
		},
		{
			types.NewCons(types.NewCons(types.NewSymbol("a"), types.NewSymbol("b")), types.Nil),
			`((|a| . |b|))`,
		},
		{
			types.NewCons(types.NewSymbol("a"), types.NewList([]types.Expression{types.NewSymbol("b")})),
			`(|a| |b|)`,
		},
		{types.NewCons(types.Nil, types.Nil), `(())`},
		{(*types.Cons)(nil), `<nil>`},
		{types.NewLiteral(types.NewSymbol("foo")), `'|foo|`},
		{types.NewLiteral(types.NewList(nil)), `'()`},
		{types.NewString("foo bar"), `"foo bar"`},
		{types.NewSymbol("foo"), `|foo|`},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, testDef.expr.String())
	}
}

func TestNewList(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")

	assert.Same(t, types.Nil, types.NewList(nil))
	assert.Same(t, types.Nil, types.NewList([]types.Expression{}))

	l := types.NewList([]types.Expression{foo, bar})
	cons, ok := l.(*types.Cons)
	assert.True(t, ok)
	assert.Same(t, foo, cons.Car())
	rest, ok := cons.Cdr().(*types.Cons)
	assert.True(t, ok)
	assert.Same(t, bar, rest.Car())
	assert.Same(t, types.Nil, rest.Cdr())

	// Nil items are rejected like in NewCons
	assert.Panics(t, func() { types.NewList([]types.Expression{foo, nil}) })
}

func TestListItems(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")

	items, ok := types.ListItems(types.Nil)
	assert.True(t, ok)
	assert.Empty(t, items)

	items, ok = types.ListItems(types.NewList([]types.Expression{foo, bar}))
	assert.True(t, ok)
	assert.Equal(t, []types.Expression{foo, bar}, items)

	// Dotted lists and non-lists are not proper lists
	_, ok = types.ListItems(types.NewCons(foo, bar))
	assert.False(t, ok)
	_, ok = types.ListItems(foo)
	assert.False(t, ok)

	// A nil *Cons is not a proper list
	var nilCons *types.Cons
	items, ok = types.ListItems(nilCons)
	assert.False(t, ok)
	assert.Nil(t, items)
}

func TestAccessors(t *testing.T) {
	sym := types.NewSymbol("foo")
	assert.Equal(t, "foo", sym.Name())
	assert.Equal(t, "foo", types.NewString("foo").Value())
	assert.Same(t, sym, types.NewLiteral(sym).Item())

	cons := types.NewCons(sym, types.Nil)
	assert.Same(t, sym, cons.Car())
	assert.Same(t, types.Nil, cons.Cdr())

	// Accessors on a nil *Cons panic rather than dereferencing nil
	var nilCons *types.Cons
	assert.Panics(t, func() { nilCons.Car() })
	assert.Panics(t, func() { nilCons.Cdr() })
}

func TestNilString(t *testing.T) {
	assert.Equal(t, "()", types.Nil.String())
}

func TestNewConsRejectsNil(t *testing.T) {
	foo := types.NewSymbol("foo")
	var nilCons *types.Cons

	assert.Panics(t, func() { types.NewCons(nil, types.Nil) })
	assert.Panics(t, func() { types.NewCons(foo, nil) })
	assert.Panics(t, func() { types.NewCons(nilCons, types.Nil) })
	assert.Panics(t, func() { types.NewCons(foo, nilCons) })
}

func TestNewListCopiesInput(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")
	input := []types.Expression{foo}
	l := types.NewList(input)

	// Changing the slice passed to NewList must not affect the list
	input[0] = bar
	items, ok := types.ListItems(l)
	assert.True(t, ok)
	assert.Equal(t, []types.Expression{foo}, items)
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

func TestNewListTail(t *testing.T) {
	foo := types.NewSymbol("foo")
	bar := types.NewSymbol("bar")
	baz := types.NewSymbol("baz")

	// With no items the tail is returned as is
	assert.Same(t, baz, types.NewListTail(nil, baz))
	assert.Same(t, types.Nil, types.NewListTail(nil, types.Nil))

	assert.Equal(t, `(|foo| |bar| . |baz|)`, types.NewListTail([]types.Expression{foo, bar}, baz).String())
	assert.Equal(t, `(|foo| |bar|)`, types.NewListTail([]types.Expression{foo, bar}, types.Nil).String())

	var nilCons *types.Cons
	assert.Panics(t, func() { types.NewListTail(nil, nil) })
	assert.Panics(t, func() { types.NewListTail([]types.Expression{foo}, nil) })
	assert.Panics(t, func() { types.NewListTail(nil, nilCons) })
}
