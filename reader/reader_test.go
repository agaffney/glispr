package reader_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/agaffney/glispr/reader"
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
		assert.IsType(t, &reader.String{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpVal := tmp.(*reader.String).Value()
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
		assert.IsType(t, &reader.Symbol{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpName := tmp.(*reader.Symbol).Name()
		assert.Equal(t, testDef.expectedName, tmpName, "symbol name does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderList(t *testing.T) {
	testDefs := []struct {
		input         string
		expectedItems []reader.Expression
	}{
		{
			input: `("foo")`,
			expectedItems: []reader.Expression{
				reader.NewString("foo"),
			},
		},
		{
			input: `(foo bar)`,
			expectedItems: []reader.Expression{
				reader.NewSymbol("FOO"),
				reader.NewSymbol("BAR"),
			},
		},
		{
			input: `(foo "bar")`,
			expectedItems: []reader.Expression{
				reader.NewSymbol("FOO"),
				reader.NewString("bar"),
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
		assert.IsType(t, &reader.List{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpItems := tmp.(*reader.List).Items()
		assert.Equal(t, testDef.expectedItems, tmpItems, "list items does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderLiteral(t *testing.T) {
	testDefs := []struct {
		input        string
		expectedItem reader.Expression
	}{
		{
			input:        `'foo`,
			expectedItem: reader.NewSymbol("FOO"),
		},
		{
			input: `'(foo bar)`,
			expectedItem: reader.NewList(
				[]reader.Expression{
					reader.NewSymbol("FOO"),
					reader.NewSymbol("BAR"),
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
		assert.IsType(t, &reader.Literal{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpItem := tmp.(*reader.Literal).Item()
		assert.Equal(t, testDef.expectedItem, tmpItem, "literal item does not match expected value for input: %s", testDef.input)
	}
}

func TestReaderNumber(t *testing.T) {
	testDefs := []struct {
		input       string
		expectedNum *reader.Number
	}{
		{
			input: `123`,
			expectedNum: reader.NewNumberInteger(
				big.NewInt(123),
			),
		},
		{
			input:       `0`,
			expectedNum: reader.NewNumberInteger(big.NewInt(0)),
		},
		{
			input:       `-0`,
			expectedNum: reader.NewNumberInteger(big.NewInt(0)),
		},
		{
			input:       `+6`,
			expectedNum: reader.NewNumberInteger(big.NewInt(6)),
		},
		{
			input:       `+28`,
			expectedNum: reader.NewNumberInteger(big.NewInt(28)),
		},
		{
			input:       `1024.`,
			expectedNum: reader.NewNumberInteger(big.NewInt(1024)),
		},
		{
			input:       `-1`,
			expectedNum: reader.NewNumberInteger(big.NewInt(-1)),
		},
		{
			// 25 factorial (25!)
			input:       `15511210043330985984000000.`,
			expectedNum: reader.NewNumberInteger(new(big.Int).MulRange(1, 25)),
		},
		{
			input:       `2/3`,
			expectedNum: reader.NewNumberRatio(big.NewRat(2, 3)),
		},
		{
			input:       `4/6`,
			expectedNum: reader.NewNumberRatio(big.NewRat(4, 6)),
		},
		{
			input:       `-17/23`,
			expectedNum: reader.NewNumberRatio(big.NewRat(-17, 23)),
		},
		{
			input:       `-30517578125/32768`,
			expectedNum: reader.NewNumberRatio(big.NewRat(-30517578125, 32768)),
		},
		{
			input:       `10/5`,
			expectedNum: reader.NewNumberRatio(big.NewRat(10, 5)),
		},
		{
			input:       `0.0`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `0E0`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `-.0`,
			expectedNum: reader.NewNumberFloat(new(big.Float).Neg(big.NewFloat(0.0))),
		},
		{
			input:       `0.0s0`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `0s0`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(0.0)),
		},
		{
			input:       `3.1415926535897932384d0`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(3.1415926535897932384e0)),
		},
		{
			input:       `6.02E+23`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(6.02e+23)),
		},
		{
			input:       `602E+21`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(602e+21)),
		},
		{
			input:       `3.010299957f-1`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(3.010299957e-1)),
		},
		{
			input:       `-0.000000001s9`,
			expectedNum: reader.NewNumberFloat(big.NewFloat(-0.000000001e9)),
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
		assert.IsType(t, &reader.Number{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpNum := tmp.(*reader.Number)
		if !tmpNum.Equal(testDef.expectedNum) {
			t.Errorf("number does not match expected value for input: %s\n\n  Expected: %s\n  Actual: %s", testDef.input, testDef.expectedNum.String(), tmpNum.String())
		}
	}
}

func TestReaderExpression(t *testing.T) {
	testDefs := []struct {
		input        string
		expectedExpr reader.Expression
	}{
		{
			input: `(+ 2 3)`,
			expectedExpr: reader.NewList([]reader.Expression{
				reader.NewSymbol(`+`),
				reader.NewNumberInteger(big.NewInt(2)),
				reader.NewNumberInteger(big.NewInt(3)),
			}),
		},
		{
			input: "(1 2 3) ; some comment",
			expectedExpr: reader.NewList([]reader.Expression{
				reader.NewNumberInteger(big.NewInt(1)),
				reader.NewNumberInteger(big.NewInt(2)),
				reader.NewNumberInteger(big.NewInt(3)),
			}),
		},
		{
			input: "; a comment\n(foo)",
			expectedExpr: reader.NewList([]reader.Expression{
				reader.NewSymbol(`FOO`),
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
		assert.Implements(t, (*reader.Expression)(nil), tmp, "did not get expected expression type for input: %s", testDef.input)
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
		assert.IsType(t, &reader.List{}, tmp)
		list := tmp.(*reader.List)
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
				assert.IsType(t, &reader.List{}, tmp)
				list := tmp.(*reader.List)

				if tc.itemName == "" {
					assert.Empty(t, list.Items())
				} else {
					assert.Len(t, list.Items(), 1)
					sym, ok := list.Items()[0].(*reader.Symbol)
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
		expectedNum *reader.Number
	}{
		{input: `#b11010101`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#o325`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#xD5`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#2r11010101`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#8r325`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#16rD5`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#16r+D5`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#b+11010101`, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{input: `#o-300`, expectedNum: reader.NewNumberInteger(big.NewInt(-192))},
		{input: `#3r-21010`, expectedNum: reader.NewNumberInteger(big.NewInt(-192))},
		{input: `#25R-7H`, expectedNum: reader.NewNumberInteger(big.NewInt(-192))},
		{input: `#xACCEDED`, expectedNum: reader.NewNumberInteger(big.NewInt(181202413))},
		{input: `#o-101/75`, expectedNum: reader.NewNumberRatio(big.NewRat(-65, 61))},
		{input: `#3r120/21`, expectedNum: reader.NewNumberRatio(big.NewRat(15, 7))},
		{input: `#Xbc/ad`, expectedNum: reader.NewNumberRatio(big.NewRat(188, 173))},
		{input: `#xFADED/FACADE`, expectedNum: reader.NewNumberRatio(big.NewRat(1027565, 16435934))},
		{input: `#xFF.`, expectedNum: reader.NewNumberInteger(big.NewInt(255))},
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
		assert.IsType(t, &reader.Number{}, tmp, "did not get expected expression type for input: %s", testDef.input)
		tmpNum := tmp.(*reader.Number)
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
		assert.IsType(t, &reader.Symbol{}, tmp)
		assert.Equal(t, "#", tmp.(*reader.Symbol).Name())
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
		inputs := []string{"#xz", "#2rabc", "#x", "#2r", "#2r ", "#xF.F", "#1r0", "#37r0", "#999999999999999999999999r0"}
		for _, input := range inputs {
			i, err := reader.NewInput(strings.NewReader(input))
			assert.NoError(t, err)

			tmp, err := r.Read(i)
			assert.Error(t, err, "expected %q to error", input)
			assert.Nil(t, tmp)
		}
	})
}

func TestReaderError(t *testing.T) {
	testDefs := []struct {
		input      string
		wantLine   int
		wantOffset int
	}{
		{input: ")", wantLine: 1, wantOffset: 1},
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

func TestNewNumberFromString(t *testing.T) {
	testDefs := []struct {
		val         string
		radix       int
		expectedNum *reader.Number
	}{
		{val: "123", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(123))},
		{val: "0", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(0))},
		{val: "-0", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(0))},
		{val: "+6", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(6))},
		{val: "+28", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(28))},
		{val: "-1", radix: 10, expectedNum: reader.NewNumberInteger(big.NewInt(-1))},
		{val: "15511210043330985984000000", radix: 10, expectedNum: reader.NewNumberInteger(new(big.Int).MulRange(1, 25))},
		{val: "11010101", radix: 2, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{val: "325", radix: 8, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{val: "D5", radix: 16, expectedNum: reader.NewNumberInteger(big.NewInt(213))},
		{val: "FF.", radix: 16, expectedNum: reader.NewNumberInteger(big.NewInt(255))},
		{val: "-7H", radix: 25, expectedNum: reader.NewNumberInteger(big.NewInt(-192))},
		{val: "ACCEDED", radix: 16, expectedNum: reader.NewNumberInteger(big.NewInt(181202413))},
		{val: "2/3", radix: 10, expectedNum: reader.NewNumberRatio(big.NewRat(2, 3))},
		{val: "4/6", radix: 10, expectedNum: reader.NewNumberRatio(big.NewRat(4, 6))},
		{val: "-17/23", radix: 10, expectedNum: reader.NewNumberRatio(big.NewRat(-17, 23))},
		{val: "10/5", radix: 10, expectedNum: reader.NewNumberRatio(big.NewRat(10, 5))},
		{val: "-101/75", radix: 8, expectedNum: reader.NewNumberRatio(big.NewRat(-65, 61))},
		{val: "120/21", radix: 3, expectedNum: reader.NewNumberRatio(big.NewRat(15, 7))},
		{val: "bc/ad", radix: 16, expectedNum: reader.NewNumberRatio(big.NewRat(188, 173))},
		{val: "FADED/FACADE", radix: 16, expectedNum: reader.NewNumberRatio(big.NewRat(1027565, 16435934))},
		{val: "0.0", radix: 10, expectedNum: reader.NewNumberFloat(big.NewFloat(0.0))},
		{val: "0E0", radix: 10, expectedNum: reader.NewNumberFloat(big.NewFloat(0.0))},
		{val: "3.1415926535897932384", radix: 10, expectedNum: reader.NewNumberFloat(big.NewFloat(3.1415926535897932384e0))},
	}
	for _, testDef := range testDefs {
		tmp := reader.NewNumberFromString(testDef.val, testDef.radix)
		if tmp == nil {
			t.Errorf("NewNumberFromString(%q, %d) returned nil", testDef.val, testDef.radix)
			continue
		}
		if !tmp.Equal(testDef.expectedNum) {
			t.Errorf("NewNumberFromString(%q, %d) does not match expected value\n\n  Expected: %s\n  Actual: %s", testDef.val, testDef.radix, testDef.expectedNum.String(), tmp.String())
		}
	}
}
