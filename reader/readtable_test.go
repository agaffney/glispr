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
		'3':  reader.ReadtableSyntaxTypeConstituent,
		'd':  reader.ReadtableSyntaxTypeConstituent,
		'D':  reader.ReadtableSyntaxTypeConstituent,
		'λ':  reader.ReadtableSyntaxTypeConstituent,
		'#':  reader.ReadtableSyntaxTypeMacroNonTerminating,
		')':  reader.ReadtableSyntaxTypeMacroTerminating,
		'†':  reader.ReadtableSyntaxTypeInvalid,
		'\\': reader.ReadtableSyntaxTypeSingleEscape,
		'|':  reader.ReadtableSyntaxTypeMultipleEscape,
	}
	for k, v := range expectedSyntaxTypes {
		st := rt.SyntaxTypeFunc(k)
		assert.Equal(t, st, v, "unexpected character syntax type")
	}
}
