package types

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrPackageExists is returned when a package or nickname is already taken.
	ErrPackageExists = errors.New("package name already in use")
	// ErrEmptyPackageName is returned when a package is given an empty name.
	ErrEmptyPackageName = errors.New("package name is empty")
	// ErrInvalidUse is returned when a package to use is nil or isn't in the set.
	ErrInvalidUse = errors.New("package to use is nil or not in this set")
	// ErrSymbolNotAccessible is returned when a symbol isn't accessible in a package.
	ErrSymbolNotAccessible = errors.New("symbol is not accessible in package")
	// ErrNameConflict is returned when exporting a symbol would make it clash
	// with a different symbol of the same name in a package that uses the package.
	ErrNameConflict = errors.New("symbol name conflicts with a symbol in a package that uses this one")
)

// SymbolStatus is how a symbol is accessible in a package.
type SymbolStatus int

const (
	// SymbolStatusNone means the symbol isn't accessible. Intern returns it
	// when it had to create the symbol.
	SymbolStatusNone SymbolStatus = iota
	// SymbolStatusInternal means the symbol is present but not exported.
	SymbolStatusInternal
	// SymbolStatusExternal means the symbol is present and exported.
	SymbolStatusExternal
	// SymbolStatusInherited means the symbol is exported by a package that this
	// package uses.
	SymbolStatusInherited
)

// Package maps names to symbols. It is not safe for concurrent use.
type Package struct {
	name      string
	nicknames []string
	// symbols are those present in the package, whether exported or not
	symbols  map[string]*Symbol
	external map[string]bool
	uses     []*Package
	usedBy   []*Package
	// exportNew makes symbols interned in the package external, as in KEYWORD
	exportNew bool
}

func (p *Package) Name() string { return p.name }

// Nicknames returns a copy of the other names of the package.
func (p *Package) Nicknames() []string { return slices.Clone(p.nicknames) }

// Uses returns a copy of the list of packages whose exported symbols this one
// inherits.
func (p *Package) Uses() []*Package { return slices.Clone(p.uses) }

// Find looks for a symbol named name that is accessible in the package: one
// that is present in it, or exported by a package it uses. Of the used packages,
// the first in the order they were given wins. The symbol is nil and the status
// SymbolStatusNone if there isn't one.
func (p *Package) Find(name string) (*Symbol, SymbolStatus) {
	if sym, ok := p.symbols[name]; ok {
		if p.external[name] {
			return sym, SymbolStatusExternal
		}
		return sym, SymbolStatusInternal
	}
	for _, used := range p.uses {
		if used.external[name] {
			return used.symbols[name], SymbolStatusInherited
		}
	}
	return nil, SymbolStatusNone
}

// Intern returns the accessible symbol named name, with how it was accessible.
// If there is none it makes a new one that is present in the package, and the
// status is SymbolStatusNone. In a package that exports new symbols, such as
// KEYWORD, the new symbol is exported.
func (p *Package) Intern(name string) (*Symbol, SymbolStatus) {
	if sym, status := p.Find(name); sym != nil {
		return sym, status
	}
	sym := &Symbol{
		name: name,
		pkg:  p,
	}
	p.symbols[name] = sym
	if p.exportNew {
		p.external[name] = true
	}
	return sym, SymbolStatusNone
}

// Export makes sym external in the package. The symbol must be accessible in
// the package. If it is inherited, it becomes present in the package first. It
// is an error if a package that uses this one can already reach a different
// symbol with the same name, because that would make the name ambiguous there.
//
// This only looks at the packages that use this one. Two packages that were
// used together and already export the same name are not detected when the
// package is made, and the first one used wins.
func (p *Package) Export(sym *Symbol) error {
	found, status := p.Find(sym.name)
	if found != sym {
		return fmt.Errorf("%w: %s in %s", ErrSymbolNotAccessible, sym.name, p.name)
	}
	if status == SymbolStatusExternal {
		return nil
	}
	for _, user := range p.usedBy {
		if other, _ := user.Find(sym.name); other != nil && other != sym {
			return fmt.Errorf("%w: %s in %s", ErrNameConflict, sym.name, user.name)
		}
	}
	p.symbols[sym.name] = sym
	p.external[sym.name] = true
	return nil
}

// Packages is a set of packages, found by name or nickname. Names are case
// sensitive, and the standard ones are in upper case. It is not safe for
// concurrent use.
type Packages struct {
	byName map[string]*Package
	cl     *Package
	user   *Package
	kw     *Package
}

// Names of the standard packages and their nicknames.
const (
	NameCommonLisp     = "COMMON-LISP"
	NicknameCommonLisp = "CL"
	NameUser           = "COMMON-LISP-USER"
	NicknameUser       = "CL-USER"
	NameKeyword        = "KEYWORD"
)

// NewPackages returns a set with the standard packages: COMMON-LISP, with the
// symbols NIL and T exported, KEYWORD, and COMMON-LISP-USER, which uses
// COMMON-LISP.
func NewPackages() *Packages {
	ps := &Packages{
		byName: map[string]*Package{},
	}
	// These can't fail, because the set is empty and the names are different
	ps.cl, _ = ps.Make(NameCommonLisp, []string{NicknameCommonLisp}, nil)
	ps.kw, _ = ps.Make(NameKeyword, nil, nil)
	ps.kw.exportNew = true
	ps.user, _ = ps.Make(NameUser, []string{NicknameUser}, []*Package{ps.cl})
	for _, name := range []string{"NIL", "T"} {
		sym, _ := ps.cl.Intern(name)
		_ = ps.cl.Export(sym)
	}
	return ps
}

// CommonLisp returns the COMMON-LISP package.
func (ps *Packages) CommonLisp() *Package { return ps.cl }

// User returns the COMMON-LISP-USER package.
func (ps *Packages) User() *Package { return ps.user }

// Keyword returns the KEYWORD package.
func (ps *Packages) Keyword() *Package { return ps.kw }

// Find returns the package with the given name or nickname, or nil.
func (ps *Packages) Find(name string) *Package { return ps.byName[name] }

// Make adds a new package that uses the given packages, in that order, which
// must be in this set. A package that is repeated is only used once. It returns
// an error if the name is empty, if the name or any nickname is already the name
// or nickname of another package, or if a package to use is nil or not in the
// set.
func (ps *Packages) Make(name string, nicknames []string, use []*Package) (*Package, error) {
	names := append([]string{name}, nicknames...)
	for _, n := range names {
		if n == "" {
			return nil, ErrEmptyPackageName
		}
		if ps.byName[n] != nil {
			return nil, fmt.Errorf("%w: %s", ErrPackageExists, n)
		}
	}
	// A nickname can't repeat the name or another nickname
	for k, n := range names {
		if slices.Contains(names[:k], n) {
			return nil, fmt.Errorf("%w: %s", ErrPackageExists, n)
		}
	}
	var uses []*Package
	for _, used := range use {
		if used == nil {
			return nil, fmt.Errorf("%w: <nil>", ErrInvalidUse)
		}
		if ps.byName[used.name] != used {
			return nil, fmt.Errorf("%w: %s", ErrInvalidUse, used.name)
		}
		if !slices.Contains(uses, used) {
			uses = append(uses, used)
		}
	}
	p := &Package{
		name:      name,
		nicknames: slices.Clone(nicknames),
		symbols:   map[string]*Symbol{},
		external:  map[string]bool{},
		uses:      uses,
	}
	for _, used := range uses {
		used.usedBy = append(used.usedBy, p)
	}
	for _, n := range names {
		ps.byName[n] = p
	}
	return p, nil
}
