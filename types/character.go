package types

import (
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Character is a Unicode character.
type Character struct {
	r rune
}

// NewCharacter returns the character with the code point r. It panics if r is
// not a valid Unicode code point, such as a surrogate or a value above U+10FFFF.
func NewCharacter(r rune) *Character {
	if !utf8.ValidRune(r) {
		panic("types: NewCharacter called with an invalid code point")
	}
	return &Character{
		r: r,
	}
}

func (Character) Type() ExpressionType { return ExpressionTypeCharacter }

func (c Character) Rune() rune { return c.r }

// characterNames are the standard and semi-standard character names (CLHS
// 13.1.7). The first name listed for a character is the one it prints as.
var characterNames = []struct {
	name string
	r    rune
}{
	{"Backspace", '\b'},
	{"Tab", '\t'},
	{"Newline", '\n'},
	{"Linefeed", '\n'},
	{"Page", '\f'},
	{"Return", '\r'},
	{"Rubout", 0x7f}, // DEL
	{"Space", ' '},
}

// Name returns the name of the character, such as Space or Newline, or an empty
// string if it doesn't have one.
func (c Character) Name() string {
	for _, entry := range characterNames {
		if entry.r == c.r {
			return entry.name
		}
	}
	return ""
}

// String returns the character in reader syntax: #\ and the character or its
// name, or #\U+ and the hex code point if it has no name and isn't printable.
func (c Character) String() string {
	if name := c.Name(); name != "" {
		return `#\` + name
	}
	if unicode.IsPrint(c.r) {
		return `#\` + string(c.r)
	}
	return fmt.Sprintf(`#\U+%04X`, c.r)
}

// CharacterFromName returns the character with the given name, ignoring case.
// A name is one of the names Name returns, or U+ followed by one to six
// hexadecimal digits of a code point. The bool is false if there is no such
// character.
func CharacterFromName(name string) (rune, bool) {
	for _, entry := range characterNames {
		if equalFoldASCII(name, entry.name) {
			return entry.r, true
		}
	}
	// Six hex digits cover the largest code point, U+10FFFF
	const (
		prefix    = "U+"
		maxDigits = 6
	)
	if len(name) > len(prefix) && equalFoldASCII(name[:len(prefix)], prefix) && len(name) <= len(prefix)+maxDigits {
		// Radix 16, and 32 bits because that is the size of a rune
		code, err := strconv.ParseUint(name[len(prefix):], 16, 32)
		// Reject surrogates and values above U+10FFFF
		if err != nil || !utf8.ValidRune(rune(code)) {
			return 0, false
		}
		return rune(code), true
	}
	return 0, false
}

// equalFoldASCII is strings.EqualFold for ASCII letters only, so the long s
// (U+017F) and Kelvin sign (U+212A) don't match s and k.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for k := 0; k < len(a); k++ {
		if lowerASCII(a[k]) != lowerASCII(b[k]) {
			return false
		}
	}
	return true
}

// lowerASCII lowercases an ASCII letter.
func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}
