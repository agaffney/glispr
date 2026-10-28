package reader_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/agaffney/glispr/reader"
	"github.com/agaffney/glispr/types"
	"github.com/stretchr/testify/assert"
)

func TestReaderString(t *testing.T) {
	testDefs := []struct {
		input         string
		expectedValue string
	}{
		{
			input:         `"foo"`,
			expectedValue: `foo`,
		},
		{
			input:         `"foo bar"`,
			expectedValue: `foo bar`,
		},
		{
			input:         `"foo \" bar"`,
			expectedValue: `foo " bar`,
		},
		{
			input:         `"foo \\ bar"`,
			expectedValue: `foo \ bar`,
		},
		{
			input:         `"\\"`,
			expectedValue: `\`,
		},
		{
			input:         `"foo \\\" bar"`,
			expectedValue: `foo \" bar`,
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.IsType(t, &types.String{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpVal := tmp.(*types.String).Value()
		assert.Equal(t, testDef.expectedValue, tmpVal, "string value does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderSymbol(t *testing.T) {
	testDefs := []struct {
		input        string
		expectedName string
	}{
		{
			input:        `FROBBOZ`,
			expectedName: `FROBBOZ`,
		},
		{
			input:        `frobboz`,
			expectedName: `FROBBOZ`,
		},
		{
			input:        `fRObBoz`,
			expectedName: `FROBBOZ`,
		},
		{
			input:        `unwind-protect`,
			expectedName: `UNWIND-PROTECT`,
		},
		{
			input:        `+$`,
			expectedName: `+$`,
		},
		{
			input:        `1+`,
			expectedName: `1+`,
		},
		{
			input:        `pascal_style`,
			expectedName: `PASCAL_STYLE`,
		},
		{
			input:        `file.rel.43`,
			expectedName: `FILE.REL.43`,
		},
		{
			input:        `\(`,
			expectedName: `(`,
		},
		{
			input:        `\+1`,
			expectedName: `+1`,
		},
		{
			input:        `\frobboz`,
			expectedName: `fROBBOZ`,
		},
		{
			input:        `3.14159265\s0`,
			expectedName: `3.14159265s0`,
		},
		{
			input:        `3.14159265\S0`,
			expectedName: `3.14159265S0`,
		},
		{
			input:        `APL\\360`,
			expectedName: `APL\360`,
		},
		{
			input:        `apl\\360`,
			expectedName: `APL\360`,
		},
		// These come from the spec, but the spec makes no mention in the same area
		// about evaluating them as expressions and returning the display syntax
		/*
			{
				input:        `\(b^2\)\-\4*a*c`,
				expectedName: `(B^2) - 4*A*C`,
			},
			{
				input:        `\(\b^2\)\-\4*\a*\c`,
				expectedName: `(b^2) - 4*a*c`,
			},
		*/
		{
			input:        `|"|`,
			expectedName: `"`,
		},
		{
			input:        `|(b^2) - 4*a*c|`,
			expectedName: `(b^2) - 4*a*c`,
		},
		{
			input:        `|frobboz|`,
			expectedName: `frobboz`,
		},
		{
			input:        `|APL\360|`,
			expectedName: `APL360`,
		},
		{
			input:        `|APL\\360|`,
			expectedName: `APL\360`,
		},
		{
			input:        `|apl\\360|`,
			expectedName: `apl\360`,
		},
		{
			input:        `|\|\||`,
			expectedName: `||`,
		},
		{
			input:        `|(B^2) - 4*A*C|`,
			expectedName: `(B^2) - 4*A*C`,
		},
		{
			input:        `|(b^2) - 4*a*c|`,
			expectedName: `(b^2) - 4*a*c`,
		},
		{
			input:        `a#b`,
			expectedName: `A#B`,
		},
		{
			input:        `a#b#c`,
			expectedName: `A#B#C`,
		},
		{
			input:        `foo#`,
			expectedName: `FOO#`,
		},
		// An empty multiple escape is a symbol with an empty name
		{
			input:        `||`,
			expectedName: ``,
		},
		{
			input:        `|| `,
			expectedName: ``,
		},
		{
			input:        "||\n",
			expectedName: ``,
		},
		{
			input:        `||||`,
			expectedName: ``,
		},
		{
			input:        `||a`,
			expectedName: `A`,
		},
		{
			input:        `a||b`,
			expectedName: `AB`,
		},
		{
			input:        `1||`,
			expectedName: `1`,
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.IsType(t, &types.Symbol{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpName := tmp.(*types.Symbol).Name()
		assert.Equal(t, testDef.expectedName, tmpName, "symbol name does not match expected value for input: %s", testDef.input)
	}
}

// An empty multiple escape starts a token, so it must be handled the same way as
// any other token no matter what ends it.
func TestReaderEmptyMultipleEscapeTerminators(t *testing.T) {
	sym := types.NewSymbol
	testDefs := []struct {
		input string
		want  []types.Expression
	}{
		// Single escape before the empty multiple escape
		{`\a||`, []types.Expression{sym("a")}},
		{`||\a`, []types.Expression{sym("a")}},
		// Ended by a terminating macro character other than a close paren
		{`||'a`, []types.Expression{sym(""), types.NewLiteral(sym("A"))}},
		{`||"a"`, []types.Expression{sym(""), types.NewString("a")}},
		{`||;c`, []types.Expression{sym("")}},
		{`||(a)`, []types.Expression{sym(""), types.NewList([]types.Expression{sym("A")})}},
		// A non-terminating macro character is a constituent inside the token
		{`||#`, []types.Expression{sym("#")}},
		{`||a#b`, []types.Expression{sym("A#B")}},
		{`|| #`, []types.Expression{sym(""), sym("#")}},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		got := []types.Expression{}
		for {
			tmp, err := r.Read(i)
			if !assert.NoError(t, err, "unexpected error for %q", testDef.input) || tmp == nil {
				break
			}
			got = append(got, tmp)
		}
		assert.Equal(t, testDef.want, got, "wrong result for %q", testDef.input)
	}
}

func TestReaderList(t *testing.T) {
	testDefs := []struct {
		input         string
		expectedItems []types.Expression
	}{
		{
			input: `("foo")`,
			expectedItems: []types.Expression{
				types.NewString("foo"),
			},
		},
		{
			input: `(foo bar)`,
			expectedItems: []types.Expression{
				types.NewSymbol("FOO"),
				types.NewSymbol("BAR"),
			},
		},
		{
			input: `(foo "bar")`,
			expectedItems: []types.Expression{
				types.NewSymbol("FOO"),
				types.NewString("bar"),
			},
		},
		{
			input: `(a#b c)`,
			expectedItems: []types.Expression{
				types.NewSymbol("A#B"),
				types.NewSymbol("C"),
			},
		},
		{
			input: `(||)`,
			expectedItems: []types.Expression{
				types.NewSymbol(""),
			},
		},
		{
			input: `(a || b)`,
			expectedItems: []types.Expression{
				types.NewSymbol("A"),
				types.NewSymbol(""),
				types.NewSymbol("B"),
			},
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmpItems, ok := types.ListItems(tmp)
		assert.True(t, ok, "did not get a proper list for input: %s", testDef.input)
		assert.Equal(t, testDef.expectedItems, tmpItems, "list items does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderLiteral(t *testing.T) {
	testDefs := []struct {
		input        string
		expectedItem types.Expression
	}{
		{
			input:        `'foo`,
			expectedItem: types.NewSymbol("FOO"),
		},
		{
			input: `'(foo bar)`,
			expectedItem: types.NewList(
				[]types.Expression{
					types.NewSymbol("FOO"),
					types.NewSymbol("BAR"),
				},
			),
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.IsType(t, &types.Literal{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpItem := tmp.(*types.Literal).Item()
		assert.Equal(t, testDef.expectedItem, tmpItem, "literal item does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderNumber(t *testing.T) {
	testDefs := []struct {
		input       string
		expectedNum *types.Number
	}{
		{
			input: `123`,
			expectedNum: types.NewNumberInteger(
				big.NewInt(123),
			),
		},
		{
			input:       `0`,
			expectedNum: types.NewNumberInteger(big.NewInt(0)),
		},
		{
			input:       `-0`,
			expectedNum: types.NewNumberInteger(big.NewInt(0)),
		},
		{
			input:       `+6`,
			expectedNum: types.NewNumberInteger(big.NewInt(6)),
		},
		{
			input:       `+28`,
			expectedNum: types.NewNumberInteger(big.NewInt(28)),
		},
		{
			input:       `1024.`,
			expectedNum: types.NewNumberInteger(big.NewInt(1024)),
		},
		{
			input:       `-1`,
			expectedNum: types.NewNumberInteger(big.NewInt(-1)),
		},
		{
			// 25 factorial (25!)
			input:       `15511210043330985984000000.`,
			expectedNum: types.NewNumberInteger(new(big.Int).MulRange(1, 25)),
		},
		{
			input:       `2/3`,
			expectedNum: types.NewNumberRatio(big.NewRat(2, 3)),
		},
		{
			input:       `4/6`,
			expectedNum: types.NewNumberRatio(big.NewRat(4, 6)),
		},
		{
			input:       `-17/23`,
			expectedNum: types.NewNumberRatio(big.NewRat(-17, 23)),
		},
		{
			input:       `-30517578125/32768`,
			expectedNum: types.NewNumberRatio(big.NewRat(-30517578125, 32768)),
		},
		{
			input:       `10/5`,
			expectedNum: types.NewNumberRatio(big.NewRat(10, 5)),
		},
		{
			input:       `0.0`,
			expectedNum: types.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `0E0`,
			expectedNum: types.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `-.0`,
			expectedNum: types.NewNumberFloat(new(big.Float).Neg(big.NewFloat(0.0))),
		},
		{
			input:       `0.0s0`,
			expectedNum: types.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `0s0`,
			expectedNum: types.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `3.1415926535897932384d0`,
			expectedNum: types.NewNumberFloat(big.NewFloat(3.1415926535897932384e0)),
		},
		{
			input:       `6.02E+23`,
			expectedNum: types.NewNumberFloat(big.NewFloat(6.02e+23)),
		},
		{
			input:       `602E+21`,
			expectedNum: types.NewNumberFloat(big.NewFloat(602e+21)),
		},
		{
			input:       `3.010299957f-1`,
			expectedNum: types.NewNumberFloat(big.NewFloat(3.010299957e-1)),
		},
		{
			input:       `-0.000000001s9`,
			expectedNum: types.NewNumberFloat(big.NewFloat(-0.000000001e9)),
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.IsType(t, &types.Number{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpNum := tmp.(*types.Number)
		if !tmpNum.Equal(testDef.expectedNum) {
			t.Errorf("number does not match expected value for input: %s\n\n  Expected: %s\n  Actual: %s", testDef.input, testDef.expectedNum.String(), tmpNum.String())
		}
	}
}

func TestReaderExpression(t *testing.T) {
	testDefs := []struct {
		input        string
		expectedExpr types.Expression
	}{
		{
			input: `(+ 2 3)`,
			expectedExpr: types.NewList([]types.Expression{
				types.NewSymbol(`+`),
				types.NewNumberInteger(big.NewInt(2)),
				types.NewNumberInteger(big.NewInt(3)),
			}),
		},
		{
			input: "(1 2 3) ; some comment",
			expectedExpr: types.NewList([]types.Expression{
				types.NewNumberInteger(big.NewInt(1)),
				types.NewNumberInteger(big.NewInt(2)),
				types.NewNumberInteger(big.NewInt(3)),
			}),
		},
		{
			input: "; a comment\n(foo)",
			expectedExpr: types.NewList([]types.Expression{
				types.NewSymbol(`FOO`),
			}),
		},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(
			strings.NewReader(testDef.input),
		)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.Implements(t, (*types.Expression)(nil), tmp, "did not get expected expression type for input: %s", testDef.input)
		assert.Equal(t, testDef.expectedExpr, tmp, "expression does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderCloseParen(t *testing.T) {
	r := reader.NewReader()

	t.Run("unmatched close parenthesis errors", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader(")"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.Error(t, err, "expected unmatched close parenthesis to error")
		assert.Nil(t, tmp)
	})

	t.Run("empty list consumes close parenthesis", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("()"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.Same(t, types.Nil, tmp)
	})

	t.Run("list with whitespace before close parenthesis", func(t *testing.T) {
		tests := []struct {
			input    string
			itemName string
		}{
			{input: "( )", itemName: ""},
			{input: "(foo )", itemName: "FOO"},
			{input: "( foo )", itemName: "FOO"},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.input, func(t *testing.T) {
				i, err := reader.NewInput(strings.NewReader(tc.input))
				assert.NoError(t, err)

				tmp, err := r.Read(i)
				assert.NoError(t, err)
				items, ok := types.ListItems(tmp)
				assert.True(t, ok)

				if tc.itemName == "" {
					assert.Same(t, types.Nil, tmp)
				} else {
					assert.Len(t, items, 1)
					sym, ok := items[0].(*types.Symbol)
					assert.True(t, ok)
					assert.Equal(t, tc.itemName, sym.Name())
				}
			})
		}
	})
}

func TestReaderDispatchRadix(t *testing.T) {
	r := reader.NewReader()

	testDefs := []struct {
		input       string
		expectedNum *types.Number
	}{
		{input: `#b11010101`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#o325`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#xD5`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#2r11010101`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#8r325`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#16rD5`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#16r+D5`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#b+11010101`, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{input: `#o-300`, expectedNum: types.NewNumberInteger(big.NewInt(-192))},
		{input: `#3r-21010`, expectedNum: types.NewNumberInteger(big.NewInt(-192))},
		{input: `#25R-7H`, expectedNum: types.NewNumberInteger(big.NewInt(-192))},
		{input: `#xACCEDED`, expectedNum: types.NewNumberInteger(big.NewInt(181202413))},
		{input: `#o-101/75`, expectedNum: types.NewNumberRatio(big.NewRat(-65, 61))},
		{input: `#3r120/21`, expectedNum: types.NewNumberRatio(big.NewRat(15, 7))},
		{input: `#Xbc/ad`, expectedNum: types.NewNumberRatio(big.NewRat(188, 173))},
		{input: `#xFADED/FACADE`, expectedNum: types.NewNumberRatio(big.NewRat(1027565, 16435934))},
		{input: `#xFF.`, expectedNum: types.NewNumberInteger(big.NewInt(255))},
	}
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		tmp, err := r.Read(i)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		assert.IsType(t, &types.Number{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpNum := tmp.(*types.Number)
		if !tmpNum.Equal(testDef.expectedNum) {
			t.Errorf("number does not match expected value for input: %s\n\n  Expected: %s\n  Actual: %s", testDef.input, testDef.expectedNum.String(), tmpNum.String())
		}
	}

	t.Run("unknown dispatch macro errors", func(t *testing.T) {
		inputs := []string{"#p", "#q", "#?", "#!", "#&", "#*", "#+", "#-"}
		for _, input := range inputs {
			i, err := reader.NewInput(strings.NewReader(input))
			assert.NoError(t, err)

			tmp, err := r.Read(i)
			assert.Error(t, err, "expected %q to error", input)
			assert.Nil(t, tmp)
		}
	})

	t.Run("hash at EOF returns hash symbol", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("#"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.IsType(t, &types.Symbol{}, tmp)
		assert.Equal(t, "#", tmp.(*types.Symbol).Name())
	})

	t.Run("digit radix without r errors", func(t *testing.T) {
		inputs := []string{"#123", "#123x"}
		for _, input := range inputs {
			i, err := reader.NewInput(strings.NewReader(input))
			assert.NoError(t, err)

			tmp, err := r.Read(i)
			assert.Error(t, err, "expected %q to error", input)
			assert.Nil(t, tmp)
		}
	})

	t.Run("malformed radix numbers error", func(t *testing.T) {
		inputs := []string{"#xz", "#2rabc", "#x", "#2r", "#2r ", "#xF.F", "#1r0", "#37r0", "#999999999999999999999999r0", "#x1/0", "#2r-1/0"}
		for _, input := range inputs {
			i, err := reader.NewInput(strings.NewReader(input))
			assert.NoError(t, err)

			tmp, err := r.Read(i)
			assert.Error(t, err, "expected %q to error", input)
			assert.Nil(t, tmp)
		}
	})
}

func TestReaderBlockComment(t *testing.T) {
	r := reader.NewReader()

	t.Run("discards block comment", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("#| comment |#foo"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.IsType(t, &types.Symbol{}, tmp)
		assert.Equal(t, "FOO", tmp.(*types.Symbol).Name())
	})

	t.Run("supports nested block comments", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("#| outer #| inner |# outer |#bar"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.IsType(t, &types.Symbol{}, tmp)
		assert.Equal(t, "BAR", tmp.(*types.Symbol).Name())
	})

	t.Run("works inside lists", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("(foo #| comment |# bar)"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.Equal(t, types.NewList([]types.Expression{
			types.NewSymbol("FOO"),
			types.NewSymbol("BAR"),
		}), tmp)
	})

	t.Run("trailing block comment inside list", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("(foo #| comment |#)"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.Equal(t, types.NewList([]types.Expression{
			types.NewSymbol("FOO"),
		}), tmp)
	})

	t.Run("unterminated block comment errors", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("#| unterminated"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unterminated block comment")
		assert.Nil(t, tmp)
	})
}

func TestReaderEndOfInput(t *testing.T) {
	// Reaching the end of the input between expressions is not an error
	for _, input := range []string{"", " ", "\n\n", "; comment", "#| comment |#", " #| a |# ; b"} {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)

		tmp, err := reader.NewReader().Read(i)
		assert.NoError(t, err, "input: %q", input)
		assert.Nil(t, tmp, "input: %q", input)
	}
}

func TestReaderError(t *testing.T) {
	testDefs := []struct {
		input      string
		wantLine   int
		wantOffset int
	}{
		{input: ")", wantLine: 1, wantOffset: 1},
		{input: "1/0", wantLine: 1, wantOffset: 3},
		{input: "(1/0)", wantLine: 1, wantOffset: 4},
		// Unreading the newline that ends a token must not move the error position
		{input: "1/0\n", wantLine: 1, wantOffset: 3},
		{input: "\n1/0\n", wantLine: 2, wantOffset: 3},
		// Unreading the first character of a line must stay on that line
		{input: "(\n1/0)", wantLine: 2, wantOffset: 3},
		// Unexpected end of file
		{input: `"abc`, wantLine: 1, wantOffset: 4},
		{input: `"abc\`, wantLine: 1, wantOffset: 5},
		{input: `|abc`, wantLine: 1, wantOffset: 4},
		{input: `abc\`, wantLine: 1, wantOffset: 4},
		{input: `(`, wantLine: 1, wantOffset: 1},
		{input: `(a b`, wantLine: 1, wantOffset: 4},
		{input: `(||`, wantLine: 1, wantOffset: 3},
		{input: `(a ||`, wantLine: 1, wantOffset: 5},
		{input: `(a ;c`, wantLine: 1, wantOffset: 5},
		{input: `(a #| c |#`, wantLine: 1, wantOffset: 10},
		{input: `'`, wantLine: 1, wantOffset: 1},
		{input: `(a '`, wantLine: 1, wantOffset: 4},
		{input: `' ;c`, wantLine: 1, wantOffset: 4},
		{input: `''`, wantLine: 1, wantOffset: 2},
		// Offsets stay 1-based across newlines, including when EOF follows one
		{input: "(a\n", wantLine: 2, wantOffset: 1},
		{input: "(a\n b", wantLine: 2, wantOffset: 2},
		{input: "\"abc\n", wantLine: 2, wantOffset: 1},
		{input: "\"a\nbc", wantLine: 2, wantOffset: 2},
		{input: "|abc\n", wantLine: 2, wantOffset: 1},
		{input: "|a\nbc", wantLine: 2, wantOffset: 2},
		{input: "'\n", wantLine: 2, wantOffset: 1},
		{input: "(a\n\n'\n", wantLine: 4, wantOffset: 1},
		// Dot context errors
		{input: ".", wantLine: 1, wantOffset: 1},
		{input: ". ", wantLine: 1, wantOffset: 1},
		{input: "'.", wantLine: 1, wantOffset: 2},
		{input: "(a '.)", wantLine: 1, wantOffset: 5},
		{input: "(.)", wantLine: 1, wantOffset: 2},
		{input: "(. a)", wantLine: 1, wantOffset: 2},
		{input: "(a . . b)", wantLine: 1, wantOffset: 6},
		{input: "(a b . . )", wantLine: 1, wantOffset: 8},
		{input: "..", wantLine: 1, wantOffset: 2},
		{input: "(a ... b)", wantLine: 1, wantOffset: 6},
		{input: "(a . )", wantLine: 1, wantOffset: 6},
		{input: "(a . b c)", wantLine: 1, wantOffset: 8},
		{input: "(a . b . c)", wantLine: 1, wantOffset: 8},
		{input: "(a ..)", wantLine: 1, wantOffset: 5},
		{input: "(a\n.)", wantLine: 2, wantOffset: 2},
		{input: "#p", wantLine: 1, wantOffset: 2},
		{input: "\n#p", wantLine: 2, wantOffset: 2},
		{input: "#1r0", wantLine: 1, wantOffset: 3},
		{input: "#37r0", wantLine: 1, wantOffset: 4},
		{input: "#99999999999999999999r0", wantLine: 1, wantOffset: 22},
	}
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := reader.NewReader().Read(i)
		assert.Nil(t, tmp)
		if !assert.Error(t, err, "expected %q to error", testDef.input) {
			continue
		}
		var readerErr reader.ReaderError
		if !assert.True(t, errors.As(err, &readerErr), "expected ReaderError for %q", testDef.input) {
			continue
		}
		assert.Equal(t, testDef.wantLine, readerErr.Line, "wrong line for %q", testDef.input)
		assert.Equal(t, testDef.wantOffset, readerErr.Offset, "wrong offset for %q", testDef.input)
	}
}

func TestReaderDottedList(t *testing.T) {
	sym := types.NewSymbol
	list := types.NewList
	tail := types.NewListTail
	exprs := func(e ...types.Expression) []types.Expression { return e }

	testDefs := []struct {
		input string
		want  types.Expression
	}{
		{`(a . b)`, tail(exprs(sym("A")), sym("B"))},
		{`(a b . c)`, tail(exprs(sym("A"), sym("B")), sym("C"))},
		{`(a . "b")`, tail(exprs(sym("A")), types.NewString("b"))},
		// A delimiter ends a token, so no whitespace is needed around the dot there
		{`(a .(b))`, tail(exprs(sym("A")), list(exprs(sym("B"))))},
		{`((a). b)`, tail(exprs(list(exprs(sym("A")))), sym("B"))},
		// Dots inside larger tokens are not the dot
		{`(a .b)`, list(exprs(sym("A"), sym(".B")))},
		{`(a. b)`, list(exprs(sym("A."), sym("B")))},
		{`(a .5 b)`, list(exprs(sym("A"), types.NewNumberFloat(big.NewFloat(0.5)), sym("B")))},
		// Escaped dots are symbols
		{`(a \. b)`, list(exprs(sym("A"), sym("."), sym("B")))},
		{`(a |.| b)`, list(exprs(sym("A"), sym("."), sym("B")))},
		{`(a \.\. b)`, list(exprs(sym("A"), sym(".."), sym("B")))},
		{`(a . \.)`, tail(exprs(sym("A")), sym("."))},
		// The tail and the items can be quoted
		{`(a . 'b)`, tail(exprs(sym("A")), types.NewLiteral(sym("B")))},
		{`('a . b)`, tail(exprs(types.NewLiteral(sym("A"))), sym("B"))},
		// A list in the tail makes the whole list proper
		{`(a . (b c))`, list(exprs(sym("A"), sym("B"), sym("C")))},
		{`(a . ())`, list(exprs(sym("A")))},
		// Comments are allowed around the dot and the tail
		{`(a #| x |# . #| y |# b #| z |#)`, tail(exprs(sym("A")), sym("B"))},
		{"(a . b ; c\n)", tail(exprs(sym("A")), sym("B"))},
		// Nested
		{`((a . b) . (c . d))`, tail(
			exprs(tail(exprs(sym("A")), sym("B"))),
			tail(exprs(sym("C")), sym("D")),
		)},
	}
	r := reader.NewReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		if !assert.NoError(t, err, "unexpected error for %q", testDef.input) {
			continue
		}
		assert.Equal(t, testDef.want, tmp, "wrong result for %q", testDef.input)
	}
}

func TestReaderDotError(t *testing.T) {
	inputs := []string{
		// Dot outside of a list
		`.`,
		`. `,
		`'.`,
		`(a '.)`,
		// Misplaced dot in a list
		`(.)`,
		`(. a)`,
		`(a .)`,
		`(a . )`,
		`(a . #| c |# )`,
		`(a . b c)`,
		`(a . b . c)`,
		`(a . . b)`,
		`(a b . . )`,
		// Tokens made only of dots
		`..`,
		`(a ..)`,
		`(a ... b)`,
	}
	r := reader.NewReader()
	for _, input := range inputs {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.Error(t, err, "expected an error for %q", input)
		assert.Nil(t, tmp, "expected no expression for %q", input)
		var readerErr reader.ReaderError
		assert.True(t, errors.As(err, &readerErr), "expected ReaderError for %q", input)
	}

	// A dot with nothing after it is an end of file error
	i, err := reader.NewInput(strings.NewReader(`(a .`))
	assert.NoError(t, err)
	_, err = r.Read(i)
	assert.ErrorContains(t, err, "end of file")
}
