package types

import (
	"strings"
	"unicode"
)

// Format returns e in reader syntax, as printed when current is the current
// package. A symbol accessible in current prints bare, and any other symbol is
// qualified with its package, so that reading the text in current gives back the
// same symbol. If current is nil, nothing is accessible and every interned
// symbol is qualified.
//
// The text of an expression the reader can produce can be read back by a reader
// with the standard readtable and the default case mode, Upcase. The one symbol
// that does not read back as itself is COMMON-LISP:NIL, which the reader returns
// as Nil instead. A value the reader can't produce prints as diagnostic text
// that it can't read back, such as <nil> for a nil *Cons.
func Format(e Expression, current *Package) string {
	var sb strings.Builder
	writeExpr(&sb, e, current, 0)
	return sb.String()
}

// writeExpr writes e, which is inside depth levels of backquote.
func writeExpr(sb *strings.Builder, e Expression, current *Package, depth int) {
	switch v := e.(type) {
	case *Symbol:
		writeSymbol(sb, v, current)
	case *Cons:
		if v == nil {
			sb.WriteString("<nil>")
			return
		}
		if form, item, ok := prefixFormOf(v, depth); ok {
			var inner strings.Builder
			writeExpr(&inner, item, current, depth+form.delta)
			sb.WriteString(form.prefix)
			// ,@ and ,. are the splicing forms when they follow a comma
			if form.prefix == "," && (strings.HasPrefix(inner.String(), "@") || strings.HasPrefix(inner.String(), ".")) {
				sb.WriteString(" ")
			}
			sb.WriteString(inner.String())
			return
		}
		sb.WriteString("(")
		writeExpr(sb, v.car, current, depth)
		rest := v.cdr
		for {
			next, ok := rest.(*Cons)
			if !ok {
				break
			}
			sb.WriteString(" ")
			writeExpr(sb, next.car, current, depth)
			rest = next.cdr
		}
		if rest != Nil {
			sb.WriteString(" . ")
			writeExpr(sb, rest, current, depth)
		}
		sb.WriteString(")")
	case *Vector:
		sb.WriteString("#(")
		for i, item := range v.items {
			if i > 0 {
				sb.WriteString(" ")
			}
			writeExpr(sb, item, current, depth)
		}
		sb.WriteString(")")
	default:
		sb.WriteString(e.String())
	}
}

// prefixForm is a list of a symbol and one item that is written as a prefix and
// the item, with delta as the change in backquote depth inside the item.
type prefixForm struct {
	pkg, name, prefix string
	delta             int
}

var prefixForms = []prefixForm{
	{NameCommonLisp, NameQuote, "'", 0},
	{NameCommonLisp, NameFunction, "#'", 0},
	{NameGlispr, NameQuasiquote, "`", 1},
	{NameGlispr, NameUnquote, ",", -1},
	{NameGlispr, NameUnquoteSplicing, ",@", -1},
	{NameGlispr, NameUnquoteNSplicing, ",.", -1},
}

// prefixFormOf reports whether c is one of prefixForms, and returns it and the
// item. A symbol with one of those names in another package doesn't count, and
// neither does a comma form outside of a backquote, as the reader would reject
// it. depth is the number of backquotes that c is inside.
func prefixFormOf(c *Cons, depth int) (prefixForm, Expression, bool) {
	sym, isSym := c.car.(*Symbol)
	rest, isCons := c.cdr.(*Cons)
	if !isSym || !isCons || rest.cdr != Nil || sym.pkg == nil {
		return prefixForm{}, nil, false
	}
	for _, form := range prefixForms {
		if sym.pkg.name == form.pkg && sym.name == form.name && (form.delta >= 0 || depth > 0) {
			return form, rest.car, true
		}
	}
	return prefixForm{}, nil, false
}

func writeSymbol(sb *strings.Builder, sym *Symbol, current *Package) {
	switch {
	case sym.pkg == nil:
		sb.WriteString("#:")
		sb.WriteString(escapeName(sym.name, nameAfterMarker))
	case sym.pkg.name == NameKeyword:
		sb.WriteString(":")
		sb.WriteString(escapeName(sym.name, nameAfterMarker))
	default:
		if current != nil {
			if found, _ := current.Find(sym.name); found == sym {
				sb.WriteString(escapeName(sym.name, nameBare))
				return
			}
		}
		sb.WriteString(escapeName(sym.pkg.name, nameFirstPart))
		// The symbol is present in its home package, so Find looks no further
		if _, status := sym.pkg.Find(sym.name); status == SymbolStatusExternal {
			sb.WriteString(":")
		} else {
			sb.WriteString("::")
		}
		sb.WriteString(escapeName(sym.name, nameAfterMarker))
	}
}

// namePosition is where a name is written in a token, which decides what else
// could change how it reads.
type namePosition int

const (
	// nameBare is a name with nothing around it, which could also read as a
	// number or start a dispatch macro.
	nameBare namePosition = iota
	// nameFirstPart is a package name, which starts the token and so could start
	// a dispatch macro, but is followed by a marker so is never a number.
	nameFirstPart
	// nameAfterMarker follows a package marker, which makes it neither.
	nameAfterMarker
)

// escapeName returns name as it must be written to read back as that name,
// surrounded by | if it would otherwise read as something else. Characters that
// the readtable treats as invalid, even inside |, are written after a \.
func escapeName(name string, pos namePosition) string {
	if !needsEscape(name, pos) {
		return name
	}
	var sb strings.Builder
	sb.WriteByte('|')
	for _, r := range name {
		if r == '|' || r == '\\' || isInvalid(r) {
			sb.WriteByte('\\')
		}
		sb.WriteRune(r)
	}
	sb.WriteByte('|')
	return sb.String()
}

// isInvalid reports whether r is invalid syntax in the standard readtable, which
// is any character that is not printable, apart from whitespace, backspace and
// the zero-width joiner and non-joiner. It can only be read after a \.
func isInvalid(r rune) bool {
	switch r {
	case '\b', '\t', '\n', '\r', '\f', 0x7f, '\u200c', '\u200d':
		return false
	}
	return !unicode.IsPrint(r)
}

func needsEscape(name string, pos namePosition) bool {
	if name == "" {
		return true
	}
	dots := true
	for _, r := range name {
		// Upcase mode would change it
		if unicode.ToUpper(r) != r || isInvalid(r) {
			return true
		}
		// Whitespace, macro characters, escapes, and the package marker
		if strings.ContainsRune("\t\n\f\r \x7f\"'(),;`|\\:", r) {
			return true
		}
		dots = dots && r == '.'
	}
	if dots {
		return true
	}
	if pos == nameAfterMarker {
		return false
	}
	// # starts a dispatch macro at the start of a token
	if name[0] == '#' {
		return true
	}
	if pos == nameFirstPart {
		return false
	}
	// Plain tokens are read in radix 10, and an error still means it has number
	// syntax
	num, err := ParseNumber(name, 10, FloatFormatSingle)
	return num != nil || err != nil
}
