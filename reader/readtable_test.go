package reader_test

import (
	"testing"

	"github.com/agaffney/glispr/reader"
	"github.com/stretchr/testify/assert"
)

func TestStandardReadtable(t *testing.T) {
	rt := reader.StandardReadtable
	assert.Equal(t, rt.CaseMode, reader.ReadtableCaseModeUpcase, "case mode should be Upcase")
	expectedSyntaxTypes := map[rune]reader.ReadtableSyntaxType{
		' ':  reader.ReadtableSyntaxTypeWhitespace,
		'\f': reader.ReadtableSyntaxTypeWhitespace,
		// Rubout is whitespace, but backspace is a constituent
		'\x7f':   reader.ReadtableSyntaxTypeWhitespace,
		'\b':     reader.ReadtableSyntaxTypeConstituent,
		'3':      reader.ReadtableSyntaxTypeConstituent,
		'd':      reader.ReadtableSyntaxTypeConstituent,
		'D':      reader.ReadtableSyntaxTypeConstituent,
		'λ':      reader.ReadtableSyntaxTypeConstituent,
		'#':      reader.ReadtableSyntaxTypeMacroNonTerminating,
		')':      reader.ReadtableSyntaxTypeMacroTerminating,
		'†':      reader.ReadtableSyntaxTypeConstituent,
		'→':      reader.ReadtableSyntaxTypeConstituent,
		'✓':      reader.ReadtableSyntaxTypeConstituent,
		'😀':      reader.ReadtableSyntaxTypeConstituent,
		'\x00':   reader.ReadtableSyntaxTypeInvalid,
		'\x01':   reader.ReadtableSyntaxTypeInvalid,
		'\x1b':   reader.ReadtableSyntaxTypeInvalid,
		'\u0085': reader.ReadtableSyntaxTypeInvalid,
		'\u200b': reader.ReadtableSyntaxTypeInvalid,
		'\\':     reader.ReadtableSyntaxTypeSingleEscape,
		'|':      reader.ReadtableSyntaxTypeMultipleEscape,
	}
	for k, v := range expectedSyntaxTypes {
		st := rt.SyntaxTypeFunc(k)
		assert.Equal(t, st, v, "unexpected character syntax type")
	}
}
