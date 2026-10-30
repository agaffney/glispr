package reader

import (
	"unicode"

	"github.com/agaffney/glispr/types"
)

type ReadtableCaseMode int

const (
	ReadtableCaseModeNone ReadtableCaseMode = iota
	ReadtableCaseModeUpcase
	ReadtableCaseModeDowncase
	ReadtableCaseModePreserve
	ReadtableCaseModeInvert
)

type ReadtableSyntaxType int

const (
	ReadtableSyntaxTypeInvalid ReadtableSyntaxType = iota
	ReadtableSyntaxTypeConstituent
	ReadtableSyntaxTypeWhitespace
	ReadtableSyntaxTypeMacroTerminating
	ReadtableSyntaxTypeMacroNonTerminating
	ReadtableSyntaxTypeSingleEscape
	ReadtableSyntaxTypeMultipleEscape
)

type ReadtableMacroFunc func(*Reader, *Input, rune) (types.Expression, error)

type ReadtableSyntaxTypeFunc func(rune) ReadtableSyntaxType

type Readtable struct {
	CaseMode       ReadtableCaseMode
	Chars          map[rune]ReadtableSyntaxType
	SyntaxTypeFunc ReadtableSyntaxTypeFunc
	MacroFuncs     map[rune]ReadtableMacroFunc
}

var StandardReadtable = Readtable{
	CaseMode: ReadtableCaseModeUpcase,
	SyntaxTypeFunc: func(ch rune) ReadtableSyntaxType {
		switch ch {
		// backspace
		case rune(8):
			return ReadtableSyntaxTypeConstituent
		// whitespace, including rubout (delete)
		case '\t', '\n', '\r', '\f', ' ', rune(127):
			return ReadtableSyntaxTypeWhitespace
		// special characters
		case '!', '$', '%', '&', '*', '+', '-', '.', '/', ':', '<', '=', '>', '?', '@', '[', ']', '^', '_', '{', '}', '~':
			return ReadtableSyntaxTypeConstituent
		// non-terminating macro characters
		case '#':
			return ReadtableSyntaxTypeMacroNonTerminating
		// terminating macro characters
		case '"', '\'', '(', ')', ',', ';', '`':
			return ReadtableSyntaxTypeMacroTerminating
		// single escape
		case '\\':
			return ReadtableSyntaxTypeSingleEscape
		// multiple escape
		case '|':
			return ReadtableSyntaxTypeMultipleEscape
		default:
			// Any other printable character, such as digits, letters, and
			// non-ASCII symbols, is a constituent, as are the zero-width joiner
			// and non-joiner that emoji sequences and some scripts need.
			//
			// Other invisible characters and non-ASCII spaces are invalid on
			// purpose, so they can't silently end up in a symbol name. What is
			// printable depends on the Unicode version of the Go toolchain.
			if unicode.IsPrint(ch) || ch == '\u200c' || ch == '\u200d' {
				return ReadtableSyntaxTypeConstituent
			}
		}
		return ReadtableSyntaxTypeInvalid
	},
	MacroFuncs: map[rune]ReadtableMacroFunc{
		'"':  readString,
		'\'': readQuote,
		'(':  readList,
		';':  readComment,
		'#':  readDispatch,
		')':  readCloseParen,
		// TODO: add more entries for macro characters
	},
}
