package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestFormatSymbol(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)
	spaced, err := ps.Make("my lib", nil, nil)
	assert.NoError(t, err)
	hashed, err := ps.Make("#X", nil, nil)
	assert.NoError(t, err)
	numbered, err := ps.Make("1", nil, nil)
	assert.NoError(t, err)
	// Uses the other two, so what they export is accessible
	both, err := ps.Make("BOTH", nil, []*types.Package{lib, ps.CommonLisp()})
	assert.NoError(t, err)

	intern := func(p *types.Package, name string, export bool) *types.Symbol {
		sym, _ := p.Intern(name)
		if export {
			assert.NoError(t, p.Export(sym))
		}
		return sym
	}
	pub := intern(lib, "PUB", true)
	priv := intern(lib, "PRIV", false)
	userX := intern(ps.User(), "X", false)
	clT, _ := ps.CommonLisp().Find("T")
	kw := intern(ps.Keyword(), "FOO", false)

	testDefs := []struct {
		sym     *types.Symbol
		current *types.Package
		want    string
	}{
		// Accessible symbols are bare, others are qualified
		{pub, nil, `LIB:PUB`},
		{pub, lib, `PUB`},
		{pub, ps.User(), `LIB:PUB`},
		{pub, both, `PUB`},
		{priv, nil, `LIB::PRIV`},
		{priv, lib, `PRIV`},
		{priv, both, `LIB::PRIV`},
		{userX, nil, `COMMON-LISP-USER::X`},
		{userX, ps.User(), `X`},
		{userX, lib, `COMMON-LISP-USER::X`},
		// Inherited symbols are accessible too
		{clT, nil, `COMMON-LISP:T`},
		{clT, ps.User(), `T`},
		{clT, both, `T`},
		{clT, lib, `COMMON-LISP:T`},
		// Keywords and uninterned symbols don't depend on the current package
		{kw, nil, `:FOO`},
		{kw, ps.User(), `:FOO`},
		{types.NewSymbol("FOO"), nil, `#:FOO`},
		{types.NewSymbol("FOO"), ps.User(), `#:FOO`},
		// A package name is escaped like a symbol name, but is never a number
		{intern(spaced, "A", true), nil, `|my lib|:A`},
		{intern(hashed, "A", true), nil, `|#X|:A`},
		{intern(numbered, "A", true), nil, `1:A`},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, types.Format(testDef.sym, testDef.current), "%q in %v", testDef.sym.Name(), testDef.current)
	}
	assert.Equal(t, `LIB:PUB`, pub.String())
	assert.Equal(t, `:FOO`, kw.String())
}

func TestFormatSymbolEscapes(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)

	testDefs := []struct {
		name string
		// Written with nothing before it in the token, and after a marker
		bare, qualified string
	}{
		{"FOO", `FOO`, `FOO`},
		{"A1", `A1`, `A1`},
		{"foo", `|foo|`, `|foo|`},
		{"aB", `|aB|`, `|aB|`},
		{"", `||`, `||`},
		{"A B", `|A B|`, `|A B|`},
		{"A\tB", "|A\tB|", "|A\tB|"},
		{"A\nB", "|A\nB|", "|A\nB|"},
		{"A\x7fB", "|A\x7fB|", "|A\x7fB|"},
		{`A"B`, `|A"B|`, `|A"B|`},
		{`A'B`, `|A'B|`, `|A'B|`},
		{`(`, `|(|`, `|(|`},
		{`)`, `|)|`, `|)|`},
		{`,`, `|,|`, `|,|`},
		{`;`, `|;|`, `|;|`},
		{"`", "|`|", "|`|"},
		{`A:B`, `|A:B|`, `|A:B|`},
		{`A|B`, `|A\|B|`, `|A\|B|`},
		{`A\B`, `|A\\B|`, `|A\\B|`},
		{`|`, `|\||`, `|\||`},
		{`\`, `|\\|`, `|\\|`},
		// Invalid in the readtable even inside |, so each needs a \ before it
		{"A\x01B", "|A\\\x01B|", "|A\\\x01B|"},
		{"A\u00a0B", "|A\\\u00a0B|", "|A\\\u00a0B|"},
		{"A\u200bB", "|A\\\u200bB|", "|A\\\u200bB|"},
		{"A\u3000B", "|A\\\u3000B|", "|A\\\u3000B|"},
		{"\x00", "|\\\x00|", "|\\\x00|"},
		// Valid in the readtable, unlike the ones above
		{"A\u200dB", "A\u200dB", "A\u200dB"},
		// Characters that are fine in the middle of a token
		{`A#B`, `A#B`, `A#B`},
		{`A.B`, `A.B`, `A.B`},
		{`1+`, `1+`, `1+`},
		{`+`, `+`, `+`},
		{`-`, `-`, `-`},
		{`A/B`, `A/B`, `A/B`},
		{`Λ`, `Λ`, `Λ`},
		// Lowercase in any script, as upcasing changes it
		{`λ`, `|λ|`, `|λ|`},
		// A leading # is a dispatch macro, but only when nothing comes before it
		{`#A`, `|#A|`, `#A`},
		// Names that read as numbers, unless there is a marker before them
		{`1`, `|1|`, `1`},
		{`-12`, `|-12|`, `-12`},
		{`1.`, `|1.|`, `1.`},
		{`1.5`, `|1.5|`, `1.5`},
		{`1/2`, `|1/2|`, `1/2`},
		{`1E5`, `|1E5|`, `1E5`},
		{`1E999`, `|1E999|`, `1E999`},
		{`1/0`, `|1/0|`, `1/0`},
		// Dots are an error on their own, in any position
		{`.`, `|.|`, `|.|`},
		{`...`, `|...|`, `|...|`},
	}
	for _, testDef := range testDefs {
		sym, _ := lib.Intern(testDef.name)
		assert.Equal(t, testDef.bare, types.Format(sym, lib), "bare %q", testDef.name)
		assert.Equal(t, "LIB::"+testDef.qualified, types.Format(sym, nil), "qualified %q", testDef.name)
		assert.Equal(t, "#:"+testDef.qualified, types.Format(types.NewSymbol(testDef.name), nil), "uninterned %q", testDef.name)
		kw, _ := ps.Keyword().Intern(testDef.name)
		assert.Equal(t, ":"+testDef.qualified, types.Format(kw, nil), "keyword %q", testDef.name)
	}
}

func TestFormatNested(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)
	a, _ := ps.User().Intern("A")
	b, _ := lib.Intern("B")
	kw, _ := ps.Keyword().Intern("K")

	list := types.NewListTail([]types.Expression{a, types.NewVector([]types.Expression{b, kw})}, b)
	assert.Equal(t, `(A #(LIB::B :K) . LIB::B)`, types.Format(list, ps.User()))
	assert.Equal(t, `(COMMON-LISP-USER::A #(LIB::B :K) . LIB::B)`, types.Format(list, nil))
	assert.Equal(t, `(COMMON-LISP-USER::A #(LIB::B :K) . LIB::B)`, list.String())

	// Other expressions print as they always do
	assert.Equal(t, `()`, types.Format(types.Nil, ps.User()))
	assert.Equal(t, `"s"`, types.Format(types.NewString("s"), ps.User()))
	assert.Equal(t, `#\a`, types.Format(types.NewCharacter('a'), ps.User()))
}

func TestFormatQuote(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)
	quote, _ := ps.CommonLisp().Find(types.NameQuote)
	function, _ := ps.CommonLisp().Find(types.NameFunction)
	libQuote, _ := lib.Intern(types.NameQuote)
	a, _ := ps.User().Intern("A")
	b, _ := ps.User().Intern("B")
	list := func(items ...types.Expression) types.Expression { return types.NewList(items) }

	testDefs := []struct {
		expr types.Expression
		want string
	}{
		{list(quote, a), `'A`},
		{list(function, a), `#'A`},
		{list(quote, list(quote, a)), `''A`},
		{list(quote, list(function, a)), `'#'A`},
		{list(quote, list(a, b)), `'(A B)`},
		{list(quote, types.Nil), `'()`},
		{list(a, list(quote, b)), `(A 'B)`},
		{types.NewVector([]types.Expression{list(quote, a)}), `#('A)`},
		{types.NewCons(a, list(quote, b)), `(A QUOTE B)`},
		{list(quote, types.NewCons(a, b)), `'(A . B)`},
		// Anything that isn't exactly the symbol and one item is a plain list
		{list(quote), `(QUOTE)`},
		{list(quote, a, b), `(QUOTE A B)`},
		{types.NewCons(quote, a), `(QUOTE . A)`},
		{types.NewListTail([]types.Expression{quote, a}, b), `(QUOTE A . B)`},
		{list(function, a, b), `(FUNCTION A B)`},
		// A symbol with that name from elsewhere isn't the quote symbol
		{list(libQuote, a), `(LIB::QUOTE A)`},
		{list(types.NewSymbol(types.NameQuote), a), `(#:QUOTE A)`},
		{list(a, a), `(A A)`},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, types.Format(testDef.expr, ps.User()))
	}
	assert.Equal(t, `'COMMON-LISP-USER::A`, list(quote, a).String())
	assert.Equal(t, `(COMMON-LISP:QUOTE)`, list(quote).String())
}
