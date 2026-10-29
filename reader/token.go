package reader

import (
	"slices"
	"strings"
)

// traitSet is a set of constituent traits (CLHS 2.1.4.2). The traits of a
// character are fixed by the standard and are not part of the readtable.
type traitSet uint16

const (
	traitAlphabetic traitSet = 1 << iota
	// traitDigit means the character can be a digit in some radix (0-9, a-z and
	// A-Z). Whether it is a digit in the current *read-base* is up to the code
	// interpreting the token.
	traitDigit
	traitPlusSign
	traitMinusSign
	traitDot
	traitDecimalPoint
	traitRatioMarker
	traitExponentMarker
	traitPackageMarker
	traitInvalid
)

// constituentTraits returns the standard constituent traits of r, following
// CLHS figure 2-7. Characters the standard doesn't mention are alphabetic.
func constituentTraits(r rune) traitSet {
	switch {
	case r == '\b':
		return traitInvalid
	case r == '\t' || r == '\n' || r == '\f' || r == '\r' || r == ' ' || r == 0x7f:
		// Whitespace in the standard readtable, so only constituents if a
		// readtable makes them so, and then invalid
		return traitInvalid
	case r >= '0' && r <= '9':
		return traitAlphabetic | traitDigit
	case strings.ContainsRune("eEsSfFdDlL", r):
		return traitAlphabetic | traitDigit | traitExponentMarker
	case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		return traitAlphabetic | traitDigit
	case r == '+':
		return traitAlphabetic | traitPlusSign
	case r == '-':
		return traitAlphabetic | traitMinusSign
	case r == '.':
		return traitAlphabetic | traitDot | traitDecimalPoint
	case r == '/':
		return traitAlphabetic | traitRatioMarker
	case r == ':':
		return traitPackageMarker
	}
	return traitAlphabetic
}

// tokenChar is one character of a token along with how it was written.
type tokenChar struct {
	r       rune
	escaped bool
	traits  traitSet
}

// token is the characters accumulated while reading a token, before it is
// turned into a number or symbol.
type token struct {
	chars []tokenChar
	// sawEscape is set when an escape character appeared in the token, even if
	// it contributed no characters, as in the empty multiple escape of `1||`.
	sawEscape bool
	// escapes holds the number of characters the token had at each multiple
	// escape character, to show where an empty escape (`||`) came in a token.
	escapes []int
}

// add appends r to the token. An escaped character is treated as if it were a
// constituent whose only trait is alphabetic (CLHS 2.2), and marks the token as
// having used an escape.
func (t *token) add(r rune, escaped bool) {
	t.sawEscape = t.sawEscape || escaped
	traits := constituentTraits(r)
	if escaped {
		traits = traitAlphabetic
	}
	t.chars = append(t.chars, tokenChar{r: r, escaped: escaped, traits: traits})
}

// started reports whether any part of a token has been read. An empty multiple
// escape (`||`) starts a token even though it adds no characters.
func (t *token) started() bool { return len(t.chars) > 0 || t.sawEscape }

// markEscape records a multiple escape character at the current end of the
// token.
func (t *token) markEscape() {
	t.sawEscape = true
	t.escapes = append(t.escapes, len(t.chars))
}

// hasEscapeAt reports whether a multiple escape character was read when the
// token had n characters, so it came between character n-1 and character n.
func (t *token) hasEscapeAt(n int) bool { return slices.Contains(t.escapes, n) }

// hasEscape reports whether the token was written using an escape character.
func (t *token) hasEscape() bool { return t.sawEscape }

// allDots reports whether the token is made up only of dots. Escaped characters
// only have the alphabetic trait, so a token with an escape never qualifies.
func (t *token) allDots() bool {
	if len(t.chars) == 0 {
		return false
	}
	for _, c := range t.chars {
		if c.traits&traitDot == 0 {
			return false
		}
	}
	return true
}

// isSingleDot reports whether the token is exactly one unescaped dot, which is
// the dot of a dotted list rather than a symbol.
func (t *token) isSingleDot() bool {
	return len(t.chars) == 1 && t.allDots()
}

// String returns the characters of the token exactly as read, without case
// conversion.
func (t *token) String() string {
	var sb strings.Builder
	for _, c := range t.chars {
		sb.WriteRune(c.r)
	}
	return sb.String()
}
