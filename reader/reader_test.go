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
		assert.IsType(t, &types.List{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpItems := tmp.(*types.List).Items()
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
		assert.IsType(t, &types.List{}, tmp)
		list := tmp.(*types.List)
		assert.Empty(t, list.Items())
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
				assert.IsType(t, &types.List{}, tmp)
				list := tmp.(*types.List)

				if tc.itemName == "" {
					assert.Empty(t, list.Items())
				} else {
					assert.Len(t, list.Items(), 1)
					sym, ok := list.Items()[0].(*types.Symbol)
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
