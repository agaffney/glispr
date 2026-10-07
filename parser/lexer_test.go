package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/agaffney/glispr/parser"
)

func TestNewLexer(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		r, err := parser.NewLexer(strings.NewReader("foo"))
		if r == nil {
			t.Error("expected non-nil *Lexer")
		}
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	})
	t.Run("nil reader", func(t *testing.T) {
		r, err := parser.NewLexer(nil)
		if r != nil {
			t.Error("expected nil *Lexer")
		}
		if err == nil {
			t.Error("expected non-nil error")
		} else {
			if err.Error() != "nil io.Reader" {
				t.Errorf("expected error 'nil io.Reader', got: %s", err)
			}
		}
	})
}

func TestLexerTokens(t *testing.T) {
	testDefs := []struct {
		input          string
		expectedTokens []*parser.Token
	}{
		{
			input: `(+ 2 3)`,
			expectedTokens: []*parser.Token{
				{
					Type:    parser.TokenTypeLParen,
					Value:   `(`,
					Line:    1,
					LinePos: 1,
				},
				{
					Type:    parser.TokenTypeAtom,
					Value:   `+`,
					Line:    1,
					LinePos: 2,
				},
				{
					Type:    parser.TokenTypeNumber,
					Value:   `2`,
					Line:    1,
					LinePos: 4,
				},
				{
					Type:    parser.TokenTypeNumber,
					Value:   `3`,
					Line:    1,
					LinePos: 6,
				},
				{
					Type:    parser.TokenTypeRParen,
					Value:   `)`,
					Line:    1,
					LinePos: 7,
				},
			},
		},
		{
			input: "(1 2 3) ; some comment",
			expectedTokens: []*parser.Token{
				{
					Type:    parser.TokenTypeLParen,
					Value:   `(`,
					Line:    1,
					LinePos: 1,
				},
				{
					Type:    parser.TokenTypeNumber,
					Value:   `1`,
					Line:    1,
					LinePos: 2,
				},
				{
					Type:    parser.TokenTypeNumber,
					Value:   `2`,
					Line:    1,
					LinePos: 4,
				},
				{
					Type:    parser.TokenTypeNumber,
					Value:   `3`,
					Line:    1,
					LinePos: 6,
				},
				{
					Type:    parser.TokenTypeRParen,
					Value:   `)`,
					Line:    1,
					LinePos: 7,
				},
				{
					Type:    parser.TokenTypeComment,
					Value:   `; some comment`,
					Line:    1,
					LinePos: 9,
				},
			},
		},
		{
			input: "; a comment\n(foo)",
			expectedTokens: []*parser.Token{
				{
					Type:    parser.TokenTypeComment,
					Value:   `; a comment`,
					Line:    1,
					LinePos: 1,
				},
				{
					Type:    parser.TokenTypeLParen,
					Value:   `(`,
					Line:    2,
					LinePos: 1,
				},
				{
					Type:    parser.TokenTypeAtom,
					Value:   `foo`,
					Line:    2,
					LinePos: 2,
				},
				{
					Type:    parser.TokenTypeRParen,
					Value:   `)`,
					Line:    2,
					LinePos: 5,
				},
			},
		},
		// TODO
	}
testLoop:
	for _, testDef := range testDefs {
		l, err := parser.NewLexer(strings.NewReader(testDef.input))
		if err != nil {
			t.Errorf("unexpected error creating Lexer: %s", err)
			continue
		}
		var testTokens []*parser.Token
		for {
			tok, err := l.NextToken()
			if err != nil {
				t.Errorf("unexpected error getting next token: %s", err)
				continue testLoop
			}
			if tok.Type == parser.TokenTypeEOF {
				break
			}
			testTokens = append(testTokens, tok)
		}
		if len(testTokens) != len(testDef.expectedTokens) {
			t.Errorf("did not get expected number of tokens: got %d, wanted %d", len(testTokens), len(testDef.expectedTokens))
			continue testLoop
		}
		for i, testToken := range testTokens {
			if !reflect.DeepEqual(testToken, testDef.expectedTokens[i]) {
				t.Errorf("did not get expected token\n     got %#v\n  wanted %#v", testToken, testDef.expectedTokens[i])
				continue testLoop
			}
		}
	}
}
