package reader

import "unicode"

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

type ReadtableMacroFunc func(*Reader, *Input, rune) (Expression, error)

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
		// backspace and delete
		case rune(8), rune(127):
			return ReadtableSyntaxTypeConstituent
		// whitespace
		case '\t', '\n', '\r', ' ':
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
			// digits
			if ch >= '0' && ch <= '9' {
				return ReadtableSyntaxTypeConstituent
			}
			// letters
			if unicode.IsLetter(ch) {
				return ReadtableSyntaxTypeConstituent
			}
		}
		return ReadtableSyntaxTypeInvalid
	},
	MacroFuncs: map[rune]ReadtableMacroFunc{
		'"':  readString,
		'\'': readLiteral,
		'(':  readList,
		';':  readComment,
		// TODO: add more entries for macro characters
	},
}
