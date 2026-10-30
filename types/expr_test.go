package types_test

import (
	"math"
	"math/big"
	"math/rand"
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
		{types.NewList([]types.Expression{types.NewSymbol("foo")}), `(#:|foo|)`},
		{
			types.NewList([]types.Expression{
				types.NewSymbol("foo"),
				types.NewString("bar"),
				types.NewNumberInteger(big.NewInt(1)),
				types.NewList([]types.Expression{types.NewNumberRatio(big.NewRat(1, 2))}),
			}),
			`(#:|foo| "bar" 1 (1/2))`,
		},
		{
			types.NewCons(types.NewSymbol("a"), types.NewSymbol("b")),
			`(#:|a| . #:|b|)`,
		},
		{
			types.NewCons(types.NewSymbol("a"), types.NewCons(types.NewSymbol("b"), types.NewSymbol("c"))),
			`(#:|a| #:|b| . #:|c|)`,
		},
		{
			types.NewList([]types.Expression{
				types.NewList([]types.Expression{types.NewSymbol("a")}),
				types.NewSymbol("b"),
			}),
			`((#:|a|) #:|b|)`,
		},
		{
			types.NewCons(types.NewCons(types.NewSymbol("a"), types.NewSymbol("b")), types.Nil),
			`((#:|a| . #:|b|))`,
		},
		{
			types.NewCons(types.NewSymbol("a"), types.NewList([]types.Expression{types.NewSymbol("b")})),
			`(#:|a| #:|b|)`,
		},
		{types.NewCons(types.Nil, types.Nil), `(())`},
		{(*types.Cons)(nil), `<nil>`},
		{types.NewString("foo bar"), `"foo bar"`},
		{types.NewString(""), `""`},
		{types.NewString(`a"b`), `"a\"b"`},
		{types.NewString(`a\b`), `"a\\b"`},
		{types.NewString(`\"`), `"\\\""`},
		{types.NewString(`"`), `"\""`},
		{types.NewString("a\nb"), "\"a\nb\""},
		{types.NewString("a'(;|#b"), `"a'(;|#b"`},
		{types.NewSymbol("foo"), `#:|foo|`},
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

	f := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble)
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
	oneFloat := types.NewNumberFloat(big.NewFloat(1), types.FloatFormatDouble)
	oneRatio := types.NewNumberRatio(big.NewRat(1, 1))

	assert.True(t, one.Equal(types.NewNumberInteger(big.NewInt(1))))
	assert.False(t, one.Equal(types.NewNumberInteger(big.NewInt(2))))
	assert.True(t, oneFloat.Equal(types.NewNumberFloat(big.NewFloat(1), types.FloatFormatDouble)))
	assert.False(t, oneFloat.Equal(types.NewNumberFloat(big.NewFloat(2), types.FloatFormatDouble)))
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
	assert.Equal(t, "1.5d0", types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble).String())
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

	assert.Equal(t, `(#:|foo| #:|bar| . #:|baz|)`, types.NewListTail([]types.Expression{foo, bar}, baz).String())
	assert.Equal(t, `(#:|foo| #:|bar|)`, types.NewListTail([]types.Expression{foo, bar}, types.Nil).String())

	var nilCons *types.Cons
	assert.Panics(t, func() { types.NewListTail(nil, nil) })
	assert.Panics(t, func() { types.NewListTail([]types.Expression{foo}, nil) })
	assert.Panics(t, func() { types.NewListTail(nil, nilCons) })
}

func TestFloatFormat(t *testing.T) {
	assert.Equal(t, uint(24), types.FloatFormatSingle.Precision())
	assert.Equal(t, uint(53), types.FloatFormatDouble.Precision())
	assert.Equal(t, uint(0), types.FloatFormatNone.Precision())
	assert.Equal(t, "single-float", types.FloatFormatSingle.String())
	assert.Equal(t, "double-float", types.FloatFormatDouble.String())

	single := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatSingle)
	double := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble)
	assert.Equal(t, types.FloatFormatSingle, single.Format())
	assert.Equal(t, types.FloatFormatDouble, double.Format())
	assert.Equal(t, types.FloatFormatNone, types.NewNumberInteger(big.NewInt(1)).Format())
	assert.Equal(t, types.FloatFormatNone, types.NewNumberRatio(big.NewRat(1, 2)).Format())

	// The value is rounded to the precision of the format
	third := big.NewFloat(1.0 / 3.0)
	assert.Equal(t, uint(24), types.NewNumberFloat(third, types.FloatFormatSingle).Float().Prec())
	assert.Equal(t, uint(53), types.NewNumberFloat(third, types.FloatFormatDouble).Float().Prec())
	f32, _ := types.NewNumberFloat(third, types.FloatFormatSingle).Float().Float64()
	assert.Equal(t, float64(float32(1.0/3.0)), f32)

	assert.Panics(t, func() { types.NewNumberFloat(big.NewFloat(1), types.FloatFormatNone) })
	assert.Panics(t, func() { types.NewNumberFloat(big.NewFloat(1), types.FloatFormat(99)) })
}

func TestFloatFormatEqual(t *testing.T) {
	single := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatSingle)
	double := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble)

	assert.True(t, single.Equal(types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatSingle)))
	assert.True(t, double.Equal(types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble)))
	// The same value in different formats is not the same number
	assert.False(t, single.Equal(double))
	assert.False(t, double.Equal(single))
}

func TestFloatString(t *testing.T) {
	single := func(f float64) *types.Number {
		return types.NewNumberFloat(big.NewFloat(float64(float32(f))), types.FloatFormatSingle)
	}
	double := func(f float64) *types.Number {
		return types.NewNumberFloat(big.NewFloat(f), types.FloatFormatDouble)
	}
	testDefs := []struct {
		num  *types.Number
		want string
	}{
		// The marker tells the formats apart even for the same value
		{single(1.5), "1.5f0"},
		{double(1.5), "1.5d0"},
		{single(1), "1.0f0"},
		{double(1), "1.0d0"},
		{single(-2), "-2.0f0"},
		// The shortest digits that read back as the same value
		{double(0.1), "0.1d0"},
		{single(0.1), "0.1f0"},
		{double(1.0 / 3.0), "0.3333333333333333d0"},
		{single(1.0 / 3.0), "0.33333334f0"},
		// Exponents have no sign or leading zeros for positive values
		{double(1e6), "1.0d6"},
		{double(1.5e300), "1.5d300"},
		{double(1e-7), "1.0d-7"},
		{single(1e10), "1.0f10"},
		{single(3.4028235e38), "3.4028235f38"},
		{double(math.MaxFloat64), "1.7976931348623157d308"},
		{double(math.SmallestNonzeroFloat64), "5.0d-324"},
		{single(1e-45), "1.0f-45"},
		{single(16777216), "1.6777216f7"},
		{double(math.Copysign(0, -1)), "-0.0d0"},
		{single(0), "0.0f0"},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, testDef.num.String())
	}
}

// Printing a float and reading it back must give an equal number, in both
// formats, whatever the default format is.
func TestFloatStringRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	check := func(num *types.Number) {
		text := num.String()
		for _, def := range []types.FloatFormat{types.FloatFormatSingle, types.FloatFormatDouble} {
			got, err := types.ParseNumber(text, 10, def)
			if !assert.NoError(t, err, "ParseNumber(%q, 10, %s)", text, def) || !assert.NotNil(t, got, "ParseNumber(%q, 10, %s)", text, def) {
				return
			}
			assert.True(t, num.Equal(got), "%q read back as %s, with default %s", text, got, def)
			assert.Equal(t, num.Format(), got.Format(), "format of %q", text)
		}
	}
	for n := 0; n < 5000; n++ {
		if f := math.Float64frombits(rng.Uint64()); !math.IsNaN(f) && !math.IsInf(f, 0) {
			check(types.NewNumberFloat(big.NewFloat(f), types.FloatFormatDouble))
		}
		if f := math.Float32frombits(rng.Uint32()); !math.IsNaN(float64(f)) && !math.IsInf(float64(f), 0) {
			check(types.NewNumberFloat(big.NewFloat(float64(f)), types.FloatFormatSingle))
		}
	}
	// Zero and the extremes
	for _, f := range []float64{0, math.Copysign(0, -1), 1, -1, math.MaxFloat32, math.SmallestNonzeroFloat32} {
		check(types.NewNumberFloat(big.NewFloat(f), types.FloatFormatSingle))
	}
	for _, f := range []float64{0, math.Copysign(0, -1), 1, -1, math.MaxFloat64, math.SmallestNonzeroFloat64} {
		check(types.NewNumberFloat(big.NewFloat(f), types.FloatFormatDouble))
	}
}

func TestNumberAccessorsReturnCopies(t *testing.T) {
	i := types.NewNumberInteger(big.NewInt(5))
	i.Int().SetInt64(9)
	assert.Equal(t, "5", i.String())

	f := types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatSingle)
	f.Float().SetPrec(200).SetFloat64(9)
	assert.Equal(t, uint(24), f.Float().Prec())
	assert.Equal(t, "1.5f0", f.String())

	r := types.NewNumberRatio(big.NewRat(1, 3))
	r.Rat().SetInt64(9)
	assert.Equal(t, "1/3", r.String())

	// Accessors for the wrong kind of number return nil
	assert.Nil(t, f.Int())
	assert.Nil(t, f.Rat())
	assert.Nil(t, i.Float())
	assert.Nil(t, i.Rat())
	assert.Nil(t, r.Int())
	assert.Nil(t, r.Float())
}
