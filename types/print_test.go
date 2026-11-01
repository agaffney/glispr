package types_test

import (
	"math/big"
	"strings"
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

func TestFormatBackquote(t *testing.T) {
	ps := types.NewPackages()
	glispr := func(name string) *types.Symbol {
		sym, _ := ps.Glispr().Find(name)
		return sym
	}
	a, _ := ps.User().Intern("A")
	at, _ := ps.User().Intern("@A")
	dot, _ := ps.User().Intern(".A")
	list := func(items ...types.Expression) types.Expression { return types.NewList(items) }
	bq := func(x types.Expression) types.Expression { return list(glispr(types.NameQuasiquote), x) }
	uq := func(x types.Expression) types.Expression { return list(glispr(types.NameUnquote), x) }
	splice := func(x types.Expression) types.Expression { return list(glispr(types.NameUnquoteSplicing), x) }
	nsplice := func(x types.Expression) types.Expression { return list(glispr(types.NameUnquoteNSplicing), x) }

	testDefs := []struct {
		expr types.Expression
		want string
	}{
		{bq(a), "`A"},
		{bq(uq(a)), "`,A"},
		{bq(splice(a)), "`,@A"},
		{bq(nsplice(a)), "`,.A"},
		{bq(list(a, uq(a), splice(a), nsplice(a))), "`(A ,A ,@A ,.A)"},
		// The tail of (a . ,a) is the list (unquote a), so it prints as the
		// elements of a list, which reads back as the same structure
		{bq(types.NewCons(a, uq(a))), "`(A GLISPR:UNQUOTE A)"},
		{bq(types.NewVector([]types.Expression{uq(a)})), "`#(,A)"},
		{bq(list(a, bq(uq(uq(a))))), "`(A `,,A)"},
		// Each comma ends one level of backquote, and any more are plain lists
		{bq(uq(uq(a))), "`,(GLISPR:UNQUOTE A)"},
		{uq(a), "(GLISPR:UNQUOTE A)"},
		{splice(a), "(GLISPR:UNQUOTE-SPLICING A)"},
		{list(a, nsplice(a)), "(A (GLISPR:UNQUOTE-NSPLICING A))"},
		{bq(list(a, bq(a), uq(a))), "`(A `A ,A)"},
		// The comma forms are checked within a backquote that wasn't closed
		{list(bq(a), uq(a)), "(`A (GLISPR:UNQUOTE A))"},
		// A space keeps a symbol starting with @ or . from joining the comma
		{bq(uq(at)), "`, @A"},
		{bq(uq(dot)), "`, .A"},
		{bq(splice(at)), "`,@@A"},
		{bq(nsplice(dot)), "`,..A"},
		// Anything but the symbol and one item is a plain list
		{list(glispr(types.NameQuasiquote)), "(GLISPR:QUASIQUOTE)"},
		{list(glispr(types.NameQuasiquote), a, a), "(GLISPR:QUASIQUOTE A A)"},
		{bq(list(glispr(types.NameUnquote))), "`(GLISPR:UNQUOTE)"},
		{bq(list(glispr(types.NameUnquote), a, a)), "`(GLISPR:UNQUOTE A A)"},
		{bq(types.NewCons(glispr(types.NameUnquote), a)), "`(GLISPR:UNQUOTE . A)"},
		// Only the GLISPR symbols count
		{list(types.NewSymbol(types.NameQuasiquote), a), "(#:QUASIQUOTE A)"},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, types.Format(testDef.expr, ps.User()))
	}
	// Without a current package every symbol is qualified, but the forms are not
	assert.Equal(t, "`(COMMON-LISP-USER::A ,COMMON-LISP-USER::A)", types.Format(bq(list(a, uq(a))), nil))
	assert.Equal(t, "`(COMMON-LISP-USER::A ,COMMON-LISP-USER::A)", bq(list(a, uq(a))).String())
}

func TestFormatBase(t *testing.T) {
	ps := types.NewPackages()
	sym := func(name string) types.Expression {
		s, _ := ps.User().Intern(name)
		return s
	}
	num := func(n int64) types.Expression { return types.NewNumberInteger(big.NewInt(n)) }

	testDefs := []struct {
		base int
		expr types.Expression
		want string
	}{
		// Symbols that read as numbers in the base need escapes
		{10, sym("FF"), "FF"},
		{16, sym("FF"), "|FF|"},
		{10, sym("10"), "|10|"},
		{16, sym("10"), "|10|"},
		{2, sym("10"), "|10|"},
		{2, sym("12"), "12"},
		{16, sym("G"), "G"},
		{17, sym("G"), "|G|"},
		{36, sym("ZZ"), "|ZZ|"},
		{16, sym("A/B"), "|A/B|"},
		{10, sym("A/B"), "A/B"},
		// A decimal point makes it decimal, so it is a number in any base
		{16, sym("FF."), "FF."},
		{16, sym("10."), "|10.|"},
		// Floats are decimal in every base
		{16, sym("1.5"), "|1.5|"},
		{16, sym("1E5"), "|1E5|"},
		// Numbers are written so that the base doesn't change them
		{10, num(10), "10"},
		{16, num(10), "10."},
		{16, num(-255), "-255."},
		{10, types.NewNumberRatio(big.NewRat(1, 2)), "1/2"},
		{16, types.NewNumberRatio(big.NewRat(-1, 2)), "#10r-1/2"},
		{16, types.NewNumberFloat(big.NewFloat(1.5), types.FloatFormatDouble), "1.5d0"},
		{2, types.NewList([]types.Expression{num(3), sym("A"), sym("11")}), "(3. A |11|)"},
		{16, types.NewVector([]types.Expression{num(3), sym("FF")}), "#(3. |FF|)"},
	}
	for _, tc := range testDefs {
		assert.Equal(t, tc.want, types.FormatBase(tc.expr, ps.User(), tc.base), "base %d", tc.base)
	}

	// Format is base 10
	assert.Equal(t, types.FormatBase(num(10), nil, 10), types.Format(num(10), nil))

	for _, base := range []int{-1, 0, 1, 37} {
		assert.Panics(t, func() { types.FormatBase(num(1), nil, base) }, "base %d", base)
	}
}

func TestFormatLabels(t *testing.T) {
	ps := types.NewPackages()
	sym := func(name string) types.Expression {
		s, _ := ps.User().Intern(name)
		return s
	}
	num := func(n int64) types.Expression { return types.NewNumberInteger(big.NewInt(n)) }
	list := func(items ...types.Expression) *types.Cons { return types.NewList(items).(*types.Cons) }
	format := func(e types.Expression) string { return types.Format(e, ps.User()) }
	// selfRef builds an expression that contains itself: build is given a
	// placeholder to put where it should be
	selfRef := func(build func(ph types.Expression) types.Expression) types.Expression {
		ph := types.NewPlaceholder(1)
		e := build(ph)
		types.Substitute(e, ph, e)
		return e
	}

	// Cycles through the cdr, the car, and both
	cdrCycle := selfRef(func(ph types.Expression) types.Expression { return types.NewCons(sym("A"), ph) })
	assert.Equal(t, "#1=(A . #1#)", format(cdrCycle))
	assert.Equal(t, "#1=(#1# A)", format(selfRef(func(ph types.Expression) types.Expression {
		return types.NewCons(ph, list(sym("A")))
	})))
	assert.Equal(t, "#1=(#1#)", format(selfRef(func(ph types.Expression) types.Expression {
		return types.NewCons(ph, types.Nil)
	})))
	assert.Equal(t, "#1=(#1# . #1#)", format(selfRef(func(ph types.Expression) types.Expression {
		return types.NewCons(ph, ph)
	})))
	// A cycle back to a cell after the first one
	second := selfRef(func(ph types.Expression) types.Expression {
		return types.NewCons(sym("B"), types.NewCons(sym("C"), ph))
	})
	assert.Equal(t, "(A . #1=(B C . #1#))", format(types.NewCons(sym("A"), second)))

	// Sharing without a cycle is labeled too, so it is the same when read back
	shared := list(sym("X"))
	assert.Equal(t, "(#1=(X) #1#)", format(list(shared, shared)))
	assert.Equal(t, "(#1=(X) #1# #1#)", format(list(shared, shared, shared)))
	assert.Equal(t, "(A X)", format(types.NewCons(sym("A"), shared)))
	assert.Equal(t, "((A . #1=(X)) (B . #1#))",
		format(list(types.NewCons(sym("A"), shared), types.NewCons(sym("B"), shared))))
	// Equal but separate lists are not shared
	assert.Equal(t, "((X) (X))", format(list(list(sym("X")), list(sym("X")))))
	// Symbols and numbers are never labeled
	assert.Equal(t, "(A A 1 1)", format(list(sym("A"), sym("A"), num(1), num(1))))

	// Labels are numbered in the order they are written
	other := list(sym("Y"))
	assert.Equal(t, "(#1=(X) #2=(Y) #2# #1#)", format(list(shared, other, other, shared)))
	// A cell that is only in a cycle is the only one labeled
	inner := list(sym("A"))
	outer := selfRef(func(ph types.Expression) types.Expression {
		return list(sym("B"), inner, ph)
	})
	assert.Equal(t, "#1=(B (A) #1#)", format(outer))

	// Vectors
	vec := selfRef(func(ph types.Expression) types.Expression {
		return types.NewVector([]types.Expression{sym("A"), ph})
	})
	assert.Equal(t, "#1=#(A #1#)", format(vec))
	assert.Equal(t, "#1=#(A #1#)", types.Format(vec, ps.User()))
	sharedVec := types.NewVector([]types.Expression{sym("A")})
	assert.Equal(t, "(#1=#(A) #1#)", format(list(sharedVec, sharedVec)))
	// A cycle through a list and a vector
	assert.Equal(t, "#1=(#(#1#))", format(selfRef(func(ph types.Expression) types.Expression {
		return list(types.NewVector([]types.Expression{ph}))
	})))

	// A prefix form with a label
	quoteSym, _ := ps.CommonLisp().Find(types.NameQuote)
	quoted := list(quoteSym, sym("A"))
	assert.Equal(t, "(#1='A #1#)", format(list(quoted, quoted)))
	// The cell of the item is part of the form, so if that is shared it is a list
	item := list(sym("A"))
	assert.Equal(t, "((QUOTE . #1=(A)) #1#)", format(list(types.NewCons(quoteSym, item), item)))

	// The text of a cycle with no package is still finite
	assert.Equal(t, "#1=(COMMON-LISP-USER::A . #1#)", cdrCycle.String())
	assert.Equal(t, "#1=#(COMMON-LISP-USER::A #1#)", vec.String())

	// Sharing at every level doesn't make the text grow without bound
	doubled := types.Expression(list(sym("X")))
	for k := 0; k < 60; k++ {
		doubled = list(doubled, doubled)
	}
	text := format(doubled)
	assert.Less(t, len(text), 2000)
	assert.True(t, strings.HasPrefix(text, "(#1=(#2=("), text)
}
