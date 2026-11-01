package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestStandardPackages(t *testing.T) {
	ps := types.NewPackages()

	cl := ps.CommonLisp()
	assert.Equal(t, "COMMON-LISP", cl.Name())
	assert.Equal(t, []string{"CL"}, cl.Nicknames())
	assert.Empty(t, cl.Uses())

	user := ps.User()
	assert.Equal(t, "COMMON-LISP-USER", user.Name())
	assert.Equal(t, []string{"CL-USER"}, user.Nicknames())
	assert.Equal(t, []*types.Package{cl}, user.Uses())

	kw := ps.Keyword()
	assert.Equal(t, "KEYWORD", kw.Name())
	assert.Empty(t, kw.Nicknames())
	assert.Empty(t, kw.Uses())

	// Found by name or nickname, with the case as written
	assert.Same(t, cl, ps.Find("COMMON-LISP"))
	assert.Same(t, cl, ps.Find("CL"))
	assert.Same(t, user, ps.Find("COMMON-LISP-USER"))
	assert.Same(t, user, ps.Find("CL-USER"))
	assert.Same(t, kw, ps.Find("KEYWORD"))
	assert.Nil(t, ps.Find("cl"))
	assert.Nil(t, ps.Find("FOO"))
	assert.Nil(t, ps.Find(""))

	// These are exported from COMMON-LISP, and so inherited by the user package
	for _, name := range []string{"NIL", "T", "QUOTE", "FUNCTION"} {
		sym, status := cl.Find(name)
		if assert.NotNil(t, sym, name) {
			assert.Equal(t, types.SymbolStatusExternal, status, name)
			assert.Same(t, cl, sym.Package(), name)
		}
		userSym, status := user.Find(name)
		assert.Same(t, sym, userSym, name)
		assert.Equal(t, types.SymbolStatusInherited, status, name)
	}

	// The backquote symbols are exported from GLISPR, which uses nothing
	glispr := ps.Glispr()
	assert.Same(t, glispr, ps.Find("GLISPR"))
	assert.Empty(t, glispr.Uses())
	for _, name := range []string{"QUASIQUOTE", "UNQUOTE", "UNQUOTE-SPLICING", "UNQUOTE-NSPLICING"} {
		sym, status := glispr.Find(name)
		if assert.NotNil(t, sym, name) {
			assert.Equal(t, types.SymbolStatusExternal, status, name)
			assert.Same(t, glispr, sym.Package(), name)
		}
		// Not inherited by the user package
		userSym, _ := user.Find(name)
		assert.Nil(t, userSym, name)
	}

	// Nothing else is there yet, and each set of packages is separate
	sym, status := cl.Find("CAR")
	assert.Nil(t, sym)
	assert.Equal(t, types.SymbolStatusNone, status)
	assert.NotSame(t, cl, types.NewPackages().CommonLisp())
}

func TestPackageNicknamesAreCopied(t *testing.T) {
	ps := types.NewPackages()
	ps.CommonLisp().Nicknames()[0] = "X"
	assert.Equal(t, []string{"CL"}, ps.CommonLisp().Nicknames())
	ps.User().Uses()[0] = nil
	assert.NotNil(t, ps.User().Uses()[0])

	nicknames := []string{"F"}
	use := []*types.Package{ps.CommonLisp()}
	p, err := ps.Make("FOO", nicknames, use)
	assert.NoError(t, err)
	nicknames[0] = "G"
	use[0] = nil
	assert.Equal(t, []string{"F"}, p.Nicknames())
	assert.Equal(t, []*types.Package{ps.CommonLisp()}, p.Uses())
}

func TestMakePackage(t *testing.T) {
	ps := types.NewPackages()

	foo, err := ps.Make("FOO", []string{"F", "FO"}, []*types.Package{ps.CommonLisp()})
	assert.NoError(t, err)
	assert.Equal(t, "FOO", foo.Name())
	assert.Equal(t, []string{"F", "FO"}, foo.Nicknames())
	assert.Same(t, foo, ps.Find("FOO"))
	assert.Same(t, foo, ps.Find("F"))
	assert.Same(t, foo, ps.Find("FO"))

	// Package names are case sensitive
	lower, err := ps.Make("foo", nil, nil)
	assert.NoError(t, err)
	assert.NotSame(t, foo, lower)

	for _, tc := range []struct {
		name      string
		nicknames []string
		wantErr   error
	}{
		{"FOO", nil, types.ErrPackageExists},
		{"F", nil, types.ErrPackageExists},
		{"CL", nil, types.ErrPackageExists},
		{"KEYWORD", nil, types.ErrPackageExists},
		{"BAR", []string{"F"}, types.ErrPackageExists},
		{"BAR", []string{"CL-USER"}, types.ErrPackageExists},
		{"BAR", []string{"B", "B"}, types.ErrPackageExists},
		{"BAR", []string{"BAR"}, types.ErrPackageExists},
		{"", nil, types.ErrEmptyPackageName},
		{"BAR", []string{""}, types.ErrEmptyPackageName},
	} {
		p, err := ps.Make(tc.name, tc.nicknames, nil)
		assert.ErrorIs(t, err, tc.wantErr, "Make(%q, %q)", tc.name, tc.nicknames)
		assert.Nil(t, p)
	}

	// A failed make doesn't take any of the names
	assert.Nil(t, ps.Find("BAR"))
	assert.Nil(t, ps.Find("B"))
	_, err = ps.Make("BAR", []string{"B"}, nil)
	assert.NoError(t, err)
}

func TestMakePackageUse(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)

	// A package to use must be in the set
	other := types.NewPackages()
	for _, tc := range []struct {
		use     []*types.Package
		wantMsg string
	}{
		{[]*types.Package{nil}, "<nil>"},
		{[]*types.Package{a, nil}, "<nil>"},
		{[]*types.Package{other.CommonLisp()}, "COMMON-LISP"},
		{[]*types.Package{a, other.Keyword()}, "KEYWORD"},
	} {
		p, err := ps.Make("BAD", nil, tc.use)
		assert.ErrorIs(t, err, types.ErrInvalidUse)
		assert.ErrorContains(t, err, tc.wantMsg)
		assert.Nil(t, p)
	}
	// and a failed make doesn't take the name or leave a package using it
	assert.Nil(t, ps.Find("BAD"))
	sym, _ := a.Intern("X")
	assert.NoError(t, a.Export(sym))
	_, err := ps.Make("BAD", nil, nil)
	assert.NoError(t, err)

	// A package that is repeated is used once
	dup, err := ps.Make("DUP", nil, []*types.Package{a, a, ps.CommonLisp(), a})
	assert.NoError(t, err)
	assert.Equal(t, []*types.Package{a, ps.CommonLisp()}, dup.Uses())
	got, status := dup.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)

	// An empty list of packages to use is fine
	none, err := ps.Make("NONE", nil, []*types.Package{})
	assert.NoError(t, err)
	assert.Empty(t, none.Uses())
}

func TestInternSymbol(t *testing.T) {
	ps := types.NewPackages()
	foo, err := ps.Make("FOO", nil, nil)
	assert.NoError(t, err)

	// A new symbol is created, and is present but not exported
	sym, status := foo.Intern("X")
	assert.Equal(t, types.SymbolStatusNone, status)
	assert.Equal(t, "X", sym.Name())
	assert.Same(t, foo, sym.Package())
	assert.Equal(t, types.ExpressionTypeSymbol, sym.Type())

	found, status := foo.Find("X")
	assert.Same(t, sym, found)
	assert.Equal(t, types.SymbolStatusInternal, status)

	// Interning it again gives the same symbol
	again, status := foo.Intern("X")
	assert.Same(t, sym, again)
	assert.Equal(t, types.SymbolStatusInternal, status)

	// Names are case sensitive, and may be empty or unusual
	for _, name := range []string{"x", "", " ", "a b", "NIL"} {
		other, status := foo.Intern(name)
		assert.NotSame(t, sym, other, name)
		assert.Equal(t, types.SymbolStatusNone, status, name)
		assert.Equal(t, name, other.Name())
	}

	// Another package has its own symbol with the same name
	bar, err := ps.Make("BAR", nil, nil)
	assert.NoError(t, err)
	barSym, status := bar.Intern("X")
	assert.NotSame(t, sym, barSym)
	assert.Equal(t, types.SymbolStatusNone, status)
	assert.Same(t, bar, barSym.Package())
}

func TestInternInherited(t *testing.T) {
	ps := types.NewPackages()
	cl, user := ps.CommonLisp(), ps.User()

	// Interning a name that a used package exports finds that symbol
	nilSym, _ := cl.Find("NIL")
	got, status := user.Intern("NIL")
	assert.Same(t, nilSym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)
	assert.Same(t, cl, got.Package())

	// and doesn't make one in the package
	_, status = user.Find("NIL")
	assert.Equal(t, types.SymbolStatusInherited, status)

	// Symbols that are not exported are not inherited
	hidden, _ := cl.Intern("HIDDEN")
	got, status = user.Intern("HIDDEN")
	assert.NotSame(t, hidden, got)
	assert.Equal(t, types.SymbolStatusNone, status)
	assert.Same(t, user, got.Package())
}

func TestUsePrecedence(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)
	b, _ := ps.Make("B", nil, nil)
	aSym, _ := a.Intern("X")
	bSym, _ := b.Intern("X")
	assert.NoError(t, a.Export(aSym))
	assert.NoError(t, b.Export(bSym))

	// When two used packages export the same name, the first one used wins
	ab, err := ps.Make("AB", nil, []*types.Package{a, b})
	assert.NoError(t, err)
	got, status := ab.Find("X")
	assert.Same(t, aSym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)

	ba, err := ps.Make("BA", nil, []*types.Package{b, a})
	assert.NoError(t, err)
	got, _ = ba.Find("X")
	assert.Same(t, bSym, got)
}

func TestUseIsNotTransitive(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)
	b, _ := ps.Make("B", nil, []*types.Package{a})
	c, _ := ps.Make("C", nil, []*types.Package{b})

	sym, _ := a.Intern("X")
	assert.NoError(t, a.Export(sym))
	got, status := b.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)

	// C uses B, but B only inherits X, it doesn't export it
	got, status = c.Find("X")
	assert.Nil(t, got)
	assert.Equal(t, types.SymbolStatusNone, status)

	// Once B exports it, C inherits it, and it is present in B
	assert.NoError(t, b.Export(sym))
	got, status = c.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)
	got, status = b.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusExternal, status)
	assert.Same(t, a, got.Package())
}

func TestKeywordPackage(t *testing.T) {
	ps := types.NewPackages()
	kw := ps.Keyword()

	// Symbols interned in KEYWORD are exported
	sym, status := kw.Intern("FOO")
	assert.Equal(t, types.SymbolStatusNone, status)
	assert.Same(t, kw, sym.Package())
	got, status := kw.Find("FOO")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusExternal, status)

	got, status = kw.Intern("FOO")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusExternal, status)

	// and are not shared with other packages
	other, _ := ps.User().Intern("FOO")
	assert.NotSame(t, sym, other)
}

func TestExport(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)
	b, _ := ps.Make("B", nil, []*types.Package{a})

	sym, _ := a.Intern("X")
	_, status := b.Find("X")
	assert.Equal(t, types.SymbolStatusNone, status)

	assert.NoError(t, a.Export(sym))
	got, status := a.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusExternal, status)
	got, status = b.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)

	// Exporting it again is fine
	assert.NoError(t, a.Export(sym))

	// Exporting an inherited symbol makes it present in the package
	sym2, _ := a.Intern("Y")
	assert.NoError(t, a.Export(sym2))
	assert.NoError(t, b.Export(sym2))
	got, status = b.Find("Y")
	assert.Same(t, sym2, got)
	assert.Equal(t, types.SymbolStatusExternal, status)
}

func TestExportErrors(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)
	b, _ := ps.Make("B", nil, []*types.Package{a})
	c, _ := ps.Make("C", nil, nil)

	// A symbol that is not accessible, because it is only in another package
	inC, _ := c.Intern("X")
	assert.ErrorIs(t, a.Export(inC), types.ErrSymbolNotAccessible)
	_, status := a.Find("X")
	assert.Equal(t, types.SymbolStatusNone, status)

	// An uninterned symbol is not accessible, even with the name of an interned one
	inA, _ := a.Intern("Y")
	assert.ErrorIs(t, a.Export(types.NewSymbol("Y")), types.ErrSymbolNotAccessible)
	got, status := a.Find("Y")
	assert.Same(t, inA, got)
	assert.Equal(t, types.SymbolStatusInternal, status)

	// A used package has a different symbol with the same name
	own, _ := b.Intern("Z")
	inA2, _ := a.Intern("Z")
	err := a.Export(inA2)
	assert.ErrorIs(t, err, types.ErrNameConflict)
	assert.ErrorContains(t, err, "Z in B")
	// and it was not exported, so B still sees its own
	got, status = a.Find("Z")
	assert.Same(t, inA2, got)
	assert.Equal(t, types.SymbolStatusInternal, status)
	got, status = b.Find("Z")
	assert.Same(t, own, got)
	assert.Equal(t, types.SymbolStatusInternal, status)
}

// Exporting a symbol is not a conflict if the packages that use the package
// can already reach that same symbol some other way.
func TestExportSymbolAlreadyReachable(t *testing.T) {
	ps := types.NewPackages()
	a, _ := ps.Make("A", nil, nil)
	b, _ := ps.Make("B", nil, []*types.Package{a})
	user, _ := ps.Make("USER", nil, []*types.Package{a, b})

	// USER gets X from A, and then B re-exports it
	sym, _ := a.Intern("X")
	assert.NoError(t, a.Export(sym))
	assert.NoError(t, b.Export(sym))
	got, status := user.Find("X")
	assert.Same(t, sym, got)
	assert.Equal(t, types.SymbolStatusInherited, status)
}

// A package that uses two packages can't reach different symbols with one name
// through them, whichever of the two is used first.
func TestExportConflictOrder(t *testing.T) {
	for _, firstUsed := range []string{"P", "Q"} {
		ps := types.NewPackages()
		p, _ := ps.Make("P", nil, nil)
		q, _ := ps.Make("Q", nil, nil)
		use := []*types.Package{p, q}
		if firstUsed == "Q" {
			use = []*types.Package{q, p}
		}
		user, err := ps.Make("USER", nil, use)
		assert.NoError(t, err)

		qSym, _ := q.Intern("X")
		assert.NoError(t, q.Export(qSym))
		pSym, _ := p.Intern("X")
		assert.ErrorIs(t, p.Export(pSym), types.ErrNameConflict, "first used %s", firstUsed)

		// The failed export changed nothing, so USER still gets the one from Q
		got, status := user.Find("X")
		assert.Same(t, qSym, got, "first used %s", firstUsed)
		assert.Equal(t, types.SymbolStatusInherited, status, "first used %s", firstUsed)
		_, status = p.Find("X")
		assert.Equal(t, types.SymbolStatusInternal, status, "first used %s", firstUsed)
	}
}

func TestUninternedSymbol(t *testing.T) {
	sym := types.NewSymbol("FOO")
	assert.Nil(t, sym.Package())
	assert.Equal(t, "FOO", sym.Name())

	// A new one every time, and never the interned symbol of that name
	assert.NotSame(t, sym, types.NewSymbol("FOO"))
	ps := types.NewPackages()
	interned, _ := ps.User().Intern("FOO")
	assert.NotSame(t, sym, interned)
	assert.Nil(t, sym.Package())
	got, _ := ps.User().Find("FOO")
	assert.Same(t, interned, got)
}
