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
	return FormatBase(e, current, 10)
}

// FormatBase is like Format, but for a reader that reads tokens as numbers in
// base, the equivalent of *read-base*. It must be from 2 to 36, or FormatBase
// panics. A symbol that would read as a number in that base is written with
// escapes, as in |FF| for base 16. When the base is not 10, an integer is
// written with a trailing decimal point and a ratio after #10r, which a reader
// reads as decimal whatever its base is. A float is always decimal.
func FormatBase(e Expression, current *Package, base int) string {
	if base < 2 || base > 36 {
		panic("types: FormatBase called with a base outside 2 to 36")
	}
	var sb strings.Builder
	writeExpr(&sb, e, current, 0, base)
	return sb.String()
}

// writeExpr writes e, which is inside depth levels of backquote, for a reader
// with the read base base.
func writeExpr(sb *strings.Builder, e Expression, current *Package, depth, base int) {
	switch v := e.(type) {
	case *Symbol:
		writeSymbol(sb, v, current, base)
	case *Cons:
		if v == nil {
			sb.WriteString("<nil>")
			return
		}
		if form, item, ok := prefixFormOf(v, depth); ok {
			sb.WriteString(form.prefix)
			if form.prefix != "," {
				writeExpr(sb, item, current, depth+form.delta, base)
				return
			}
			// ,@ and ,. are the splicing forms when they follow a comma
			var inner strings.Builder
			writeExpr(&inner, item, current, depth+form.delta, base)
			text := inner.String()
			if strings.HasPrefix(text, "@") || strings.HasPrefix(text, ".") {
				sb.WriteString(" ")
			}
			sb.WriteString(text)
			return
		}
		sb.WriteString("(")
		writeExpr(sb, v.car, current, depth, base)
		rest := v.cdr
		for {
			next, ok := rest.(*Cons)
			if !ok {
				break
			}
			sb.WriteString(" ")
			writeExpr(sb, next.car, current, depth, base)
			rest = next.cdr
		}
		if rest != Nil {
			sb.WriteString(" . ")
			writeExpr(sb, rest, current, depth, base)
		}
		sb.WriteString(")")
	case *Vector:
		sb.WriteString("#(")
		for i, item := range v.items {
			if i > 0 {
				sb.WriteString(" ")
			}
			writeExpr(sb, item, current, depth, base)
		}
		sb.WriteString(")")
	case *Number:
		switch {
		case base != 10 && v.typ == NumberTypeInteger:
			sb.WriteString(v.String() + ".")
		case base != 10 && v.typ == NumberTypeRatio:
			sb.WriteString("#10r" + v.String())
		default:
			sb.WriteString(v.String())
		}
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

func writeSymbol(sb *strings.Builder, sym *Symbol, current *Package, base int) {
	switch {
	case sym.pkg == nil:
		sb.WriteString("#:")
		sb.WriteString(escapeName(sym.name, nameAfterMarker, base))
	case sym.pkg.name == NameKeyword:
		sb.WriteString(":")
		sb.WriteString(escapeName(sym.name, nameAfterMarker, base))
	default:
		if current != nil {
			if found, _ := current.Find(sym.name); found == sym {
				sb.WriteString(escapeName(sym.name, nameBare, base))
				return
			}
		}
		sb.WriteString(escapeName(sym.pkg.name, nameFirstPart, base))
		// The symbol is present in its home package, so Find looks no further
		if _, status := sym.pkg.Find(sym.name); status == SymbolStatusExternal {
			sb.WriteString(":")
		} else {
			sb.WriteString("::")
		}
		sb.WriteString(escapeName(sym.name, nameAfterMarker, base))
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
func escapeName(name string, pos namePosition, base int) string {
	if !needsEscape(name, pos, base) {
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

func needsEscape(name string, pos namePosition, base int) bool {
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
	// Plain tokens are read in the read base, and an error still means it has
	// number syntax
	num, err := ParseNumber(name, base, FloatFormatSingle)
	return num != nil || err != nil
}
