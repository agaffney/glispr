package reader_test

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/reader"
	"github.com/agaffney/glispr/types"
)

// testPackages is shared by the readers and expected symbols in these tests,
// so that a symbol read has the same identity as the one expected.
var testPackages = types.NewPackages()

func newTestReader() *reader.Reader { return reader.NewReader(testPackages) }

// testSym returns the symbol with the given name in COMMON-LISP-USER.
func testSym(name string) *types.Symbol {
	sym, _ := testPackages.User().Intern(name)
	return sym
}

// singleFloat and doubleFloat return the float that reading f in that format
// should produce.
func singleFloat(f float64) *types.Number {
	return types.NewNumberFloat(big.NewFloat(float64(float32(f))), types.FloatFormatSingle)
}

func doubleFloat(f float64) *types.Number {
	return types.NewNumberFloat(big.NewFloat(f), types.FloatFormatDouble)
}

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
	r := newTestReader()
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
		// Printable non-ASCII characters are constituents
		{
			input:        `→`,
			expectedName: `→`,
		},
		{
			input:        `a→b`,
			expectedName: `A→B`,
		},
		{
			input:        `✓ok`,
			expectedName: `✓OK`,
		},
		{
			input:        `λ😀`,
			expectedName: `Λ😀`,
		},
		{
			input:        `|a→b|`,
			expectedName: `a→b`,
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
	r := newTestReader()
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
	sym := testSym
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
	r := newTestReader()
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

func TestReaderInvalidCharacter(t *testing.T) {
	inputs := []string{
		"\x00",
		"a\x01b",
		"(a \x1b)",
		"|a\x01b|",
		// Invisible characters and non-ASCII spaces, including in the middle of a token
		"\u200b",
		"a\u200bb",
		"(a\u3000b)",
		"a\u00a0b",
	}
	r := newTestReader()
	for _, input := range inputs {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.ErrorContains(t, err, "invalid character", "input: %q", input)
		assert.Nil(t, tmp, "input: %q", input)
	}

	// A single escape takes the next character as is, so it can escape an
	// invalid character, which then only has the alphabetic trait
	i, err := reader.NewInput(strings.NewReader("a\\\x01b"))
	assert.NoError(t, err)
	tmp, err := r.Read(i)
	assert.NoError(t, err)
	assert.Equal(t, testSym("A\x01B"), tmp)
}

func TestReaderMalformedUTF8(t *testing.T) {
	testDefs := []struct {
		input      string
		wantLine   int
		wantOffset int
	}{
		{"\xff", 1, 1},
		{"abc\xff", 1, 4},
		{"(a \xc3", 1, 4},
		{"\"a\xffb\"", 1, 3},
		{"; comment \xff\n", 1, 11},
		{"|a\xffb|", 1, 3},
		{"a\n\xff", 2, 1},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		// Skip over any complete expressions before the bad byte
		var readErr error
		for {
			tmp, err := r.Read(i)
			if err != nil || tmp == nil {
				readErr = err
				break
			}
		}
		var readerErr reader.ReaderError
		if !assert.True(t, errors.As(readErr, &readerErr), "expected ReaderError for %q", testDef.input) {
			continue
		}
		assert.ErrorContains(t, readErr, "invalid UTF-8", "input: %q", testDef.input)
		assert.Equal(t, testDef.wantLine, readerErr.Line, "wrong line for %q", testDef.input)
		assert.Equal(t, testDef.wantOffset, readerErr.Offset, "wrong offset for %q", testDef.input)
	}
}

// Characters that are valid in the input but unusual, which are accepted
func TestReaderUnusualCharacters(t *testing.T) {
	sym := testSym
	testDefs := []struct {
		input string
		want  types.Expression
	}{
		// A real U+FFFD is not malformed UTF-8
		{"\ufffd", sym("\ufffd")},
		{"a\ufffdb", sym("A\ufffdB")},
		// Zero-width joiner emoji sequence and non-joiner
		{"\U0001F468\u200d\U0001F469\u200d\U0001F467", sym("\U0001F468\u200d\U0001F469\u200d\U0001F467")},
		{"a\u200cb", sym("A\u200cB")},
		// Control characters are allowed in strings and comments
		{"\"a\x01b\"", types.NewString("a\x01b")},
		{"; \x01\nfoo", sym("FOO")},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err, "unexpected error for %q", testDef.input)
		assert.Equal(t, testDef.want, tmp, "wrong result for %q", testDef.input)
	}
}

func TestReaderDispatchNonASCIIDigits(t *testing.T) {
	testDefs := []struct {
		input   string
		wantErr string
	}{
		// Arabic-indic three, then a fullwidth one
		{"#\u0663r10", "unknown dispatch macro"},
		{"#\uff11r10", "unknown dispatch macro"},
		{"#1\u0663r10", "expected 'r'"},
		{"#\u0663", "unknown dispatch macro"},
		// A non-ASCII digit first, with ASCII digits after it
		{"#\u06631r10", "unknown dispatch macro"},
		{"#\u06631", "unknown dispatch macro"},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.ErrorContains(t, err, testDef.wantErr, "input: %q", testDef.input)
		assert.Nil(t, tmp, "input: %q", testDef.input)
	}

	// ASCII digits still work, with more than one digit and either case of the r
	for _, testDef := range []struct {
		input string
		want  int64
	}{
		{"#2r101", 5},
		{"#16rFF", 255},
		{"#16RFF", 255},
		{"#36rZ", 35},
		{"#36Rz", 35},
		{"#010r12", 12},
	} {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, testDef.input)
		assert.Equal(t, types.NewNumberInteger(big.NewInt(testDef.want)), got, testDef.input)
	}
}

func TestReaderVector(t *testing.T) {
	sym := testSym
	vec := func(items ...types.Expression) *types.Vector { return types.NewVector(items) }
	num := func(n int64) *types.Number { return types.NewNumberInteger(big.NewInt(n)) }
	testDefs := []struct {
		input string
		want  []types.Expression
	}{
		{`#()`, []types.Expression{vec()}},
		{`#( )`, []types.Expression{vec()}},
		{`#(a)`, []types.Expression{vec(sym("A"))}},
		{`#(1 2 3)`, []types.Expression{vec(num(1), num(2), num(3))}},
		{`#(a "b" 1 #\c 2/3)`, []types.Expression{
			vec(sym("A"), types.NewString("b"), num(1), types.NewCharacter('c'), types.NewNumberRatio(big.NewRat(2, 3))),
		}},
		// Elements that are lists and vectors
		{`#((a) () #(b))`, []types.Expression{
			vec(types.NewList([]types.Expression{sym("A")}), types.Nil, vec(sym("B"))),
		}},
		{`#(#())`, []types.Expression{vec(vec())}},
		// Comments and whitespace inside
		{"#(a #| c |# b ; d\n c)", []types.Expression{vec(sym("A"), sym("B"), sym("C"))}},
		{"#(\n  a\n  b\n)", []types.Expression{vec(sym("A"), sym("B"))}},
		// Where the vector ends
		{`#(a)b`, []types.Expression{vec(sym("A")), sym("B")}},
		{`#(a)(b)`, []types.Expression{vec(sym("A")), types.NewList([]types.Expression{sym("B")})}},
		{`#(a) #(b)`, []types.Expression{vec(sym("A")), vec(sym("B"))}},
		// In other expressions
		{`(#(1) 2)`, []types.Expression{types.NewList([]types.Expression{vec(num(1)), num(2)})}},
		{`(a . #(b))`, []types.Expression{types.NewListTail([]types.Expression{sym("A")}, vec(sym("B")))}},
		{`'#(a)`, []types.Expression{types.NewLiteral(vec(sym("A")))}},
		// Dots in tokens are not the dot
		{`#(a.b .c)`, []types.Expression{vec(sym("A.B"), sym(".C"))}},
		{`#(\. |.|)`, []types.Expression{vec(sym("."), sym("."))}},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		var got []types.Expression
		for {
			tmp, err := r.Read(i)
			if !assert.NoError(t, err, "unexpected error for %q", testDef.input) || tmp == nil {
				break
			}
			got = append(got, tmp)
		}
		if !assert.Len(t, got, len(testDef.want), "wrong number of results for %q", testDef.input) {
			continue
		}
		for k := range testDef.want {
			assert.Equal(t, testDef.want[k], got[k], "wrong result %d for %q", k, testDef.input)
		}
	}
}

func TestReaderVectorLength(t *testing.T) {
	sym := testSym
	vec := func(items ...types.Expression) *types.Vector { return types.NewVector(items) }
	testDefs := []struct {
		input string
		want  *types.Vector
	}{
		// The last element is repeated to fill the vector
		{`#3(a)`, vec(sym("A"), sym("A"), sym("A"))},
		{`#5(a b)`, vec(sym("A"), sym("B"), sym("B"), sym("B"), sym("B"))},
		{`#3(a b c)`, vec(sym("A"), sym("B"), sym("C"))},
		{`#1(a)`, vec(sym("A"))},
		{`#0()`, vec()},
		// The length is decimal, and leading zeros don't matter
		{`#03(a)`, vec(sym("A"), sym("A"), sym("A"))},
		{`#010(a)`, vec(sym("A"), sym("A"), sym("A"), sym("A"), sym("A"), sym("A"), sym("A"), sym("A"), sym("A"), sym("A"))},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, "unexpected error for %q", testDef.input)
		assert.Equal(t, testDef.want, got, "wrong result for %q", testDef.input)
	}

	// The filled elements are the same object, not copies
	i, err := reader.NewInput(strings.NewReader(`#3(#(b))`))
	assert.NoError(t, err)
	got, err := r.Read(i)
	assert.NoError(t, err)
	if v, ok := got.(*types.Vector); assert.True(t, ok) {
		assert.Same(t, v.At(0), v.At(1))
		assert.Same(t, v.At(0), v.At(2))
	}

	// The largest allowed length
	i, err = reader.NewInput(strings.NewReader(`#1048576(a)`))
	assert.NoError(t, err)
	got, err = r.Read(i)
	assert.NoError(t, err)
	if v, ok := got.(*types.Vector); assert.True(t, ok) {
		assert.Equal(t, 1048576, v.Len())
	}
}

// A close paren after a vector is not part of it, so it is left for the next
// read, which treats it as unmatched like any other stray one.
func TestReaderVectorStrayCloseParen(t *testing.T) {
	r := newTestReader()
	for _, input := range []string{`#(a b))`, `#())`, `#3(a))`} {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)

		got, err := r.Read(i)
		assert.NoError(t, err, "input: %q", input)
		assert.IsType(t, &types.Vector{}, got, "input: %q", input)

		got, err = r.Read(i)
		assert.ErrorContains(t, err, "unmatched close parenthesis", "input: %q", input)
		assert.Nil(t, got, "input: %q", input)
	}
}

func TestReaderVectorErrors(t *testing.T) {
	testDefs := []struct {
		input   string
		wantErr string
	}{
		{`#(`, "end of file in vector"},
		{`#(a`, "end of file in vector"},
		{`#(a ;c`, "end of file in vector"},
		{`#(a #(b)`, "end of file in vector"},
		{`(#(a)`, "end of file in list"},
		{`#3(`, "end of file in vector"},
		// A vector has no dotted tail
		{`#(.)`, "dot in vector"},
		{`#(a . b)`, "dot in vector"},
		{`#(a .)`, "dot in vector"},
		{`#(. a)`, "dot in vector"},
		{`#2(a . b)`, "dot in vector"},
		// Too many elements, or none to fill the vector with
		{`#0(a)`, "too many elements (1)"},
		{`#2(a b c)`, "too many elements (3)"},
		{`#1(a b)`, "too many elements (2)"},
		{`#3()`, "no elements to fill"},
		{`#1()`, "no elements to fill"},
		// Lengths that are too large
		{`#1048577(a)`, "invalid vector length"},
		{`#99999999999999999999(a)`, "invalid vector length"},
		{`#999999999999999999999999999999(a)`, "invalid vector length"},
		// An argument must be followed by r or (
		{`#3`, "end of file after dispatch argument"},
		{`#3a`, "expected 'r' or '('"},
		{`#3 (a)`, "expected 'r' or '('"},
		{`#3)`, "expected 'r' or '('"},
		{`#3#(a)`, "expected 'r' or '('"},
		// The length is not a radix
		{`#1r(a)`, "radix must be between 2 and 36"},
		{`#99999999999999999999r1`, "radix must be between 2 and 36"},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.ErrorContains(t, err, testDef.wantErr, "input: %q", testDef.input)
		assert.Nil(t, tmp, "input: %q", testDef.input)
		var readerErr reader.ReaderError
		assert.True(t, errors.As(err, &readerErr), "expected ReaderError for %q", testDef.input)
	}

	// Radix forms with an argument still read numbers
	for input, want := range map[string]int64{"#2r101": 5, "#16rff": 255, "#3r10": 3} {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, input)
		assert.Equal(t, types.NewNumberInteger(big.NewInt(want)), got, input)
	}
}

func TestReaderCharacter(t *testing.T) {
	char := func(r rune) *types.Character { return types.NewCharacter(r) }
	testDefs := []struct {
		input string
		want  []types.Expression
	}{
		// A single character is itself, with its case as written
		{`#\a`, []types.Expression{char('a')}},
		{`#\A`, []types.Expression{char('A')}},
		{`#\z`, []types.Expression{char('z')}},
		{`#\1`, []types.Expression{char('1')}},
		{`#\é`, []types.Expression{char('é')}},
		{"#\\\U0001f600", []types.Expression{char(0x1f600)}},
		// Syntax characters after the backslash
		{`#\(`, []types.Expression{char('(')}},
		{`#\)`, []types.Expression{char(')')}},
		{`#\;`, []types.Expression{char(';')}},
		{`#\"`, []types.Expression{char('"')}},
		{`#\'`, []types.Expression{char('\'')}},
		{`#\\`, []types.Expression{char('\\')}},
		{`#\|`, []types.Expression{char('|')}},
		{`#\#`, []types.Expression{char('#')}},
		{`#\.`, []types.Expression{char('.')}},
		{`#\ `, []types.Expression{char(' ')}},
		// Whitespace and invalid characters after the backslash
		{"#\\\t", []types.Expression{char('\t')}},
		{"#\\\n", []types.Expression{char('\n')}},
		{"#\\\x00", []types.Expression{char(0)}},
		{"#\\\x7f", []types.Expression{char(0x7f)}},
		{"#\\\u200b", []types.Expression{char(0x200b)}},
		// Names, ignoring case
		{`#\Space`, []types.Expression{char(' ')}},
		// Escapes are allowed in names
		{`#\Sp|ac|e`, []types.Expression{char(' ')}},
		{`#\Sp\ace`, []types.Expression{char(' ')}},
		{`#\SPACE`, []types.Expression{char(' ')}},
		{`#\space`, []types.Expression{char(' ')}},
		{`#\Newline`, []types.Expression{char('\n')}},
		{`#\newline`, []types.Expression{char('\n')}},
		{`#\Linefeed`, []types.Expression{char('\n')}},
		{`#\Tab`, []types.Expression{char('\t')}},
		{`#\Page`, []types.Expression{char('\f')}},
		{`#\Return`, []types.Expression{char('\r')}},
		{`#\Rubout`, []types.Expression{char(0x7f)}},
		{`#\Backspace`, []types.Expression{char('\b')}},
		{`#\U+41`, []types.Expression{char('A')}},
		{`#\u+1F600`, []types.Expression{char(0x1f600)}},
		{`#\U+0`, []types.Expression{char(0)}},
		// Where the token ends
		{`#\a b`, []types.Expression{char('a'), testSym("B")}},
		{`#\a;comment`, []types.Expression{char('a')}},
		{`#\Space;comment`, []types.Expression{char(' ')}},
		{`#\a"b"`, []types.Expression{char('a'), types.NewString("b")}},
		{`#\a'b`, []types.Expression{char('a'), types.NewLiteral(testSym("B"))}},
		{`#\a(b)`, []types.Expression{char('a'), types.NewList([]types.Expression{testSym("B")})}},
		{`#\a #\b`, []types.Expression{char('a'), char('b')}},
		// Characters in other expressions
		{`(#\a)`, []types.Expression{types.NewList([]types.Expression{char('a')})}},
		{`(#\a #\b)`, []types.Expression{types.NewList([]types.Expression{char('a'), char('b')})}},
		{`(#\Space)`, []types.Expression{types.NewList([]types.Expression{char(' ')})}},
		{`(#\()`, []types.Expression{types.NewList([]types.Expression{char('(')})}},
		{`(#\))`, []types.Expression{types.NewList([]types.Expression{char(')')})}},
		{`(a . #\b)`, []types.Expression{types.NewListTail([]types.Expression{testSym("A")}, char('b'))}},
		{`'#\a`, []types.Expression{types.NewLiteral(char('a'))}},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		var got []types.Expression
		for {
			tmp, err := r.Read(i)
			if !assert.NoError(t, err, "unexpected error for %q", testDef.input) || tmp == nil {
				break
			}
			got = append(got, tmp)
		}
		if !assert.Len(t, got, len(testDef.want), "wrong number of results for %q", testDef.input) {
			continue
		}
		for k := range testDef.want {
			assert.Equal(t, testDef.want[k], got[k], "wrong result %d for %q", k, testDef.input)
		}
	}
}

// Printing a character and reading it back must give the same character.
func TestReaderCharacterRoundTrip(t *testing.T) {
	r := newTestReader()
	check := func(code rune) {
		c := types.NewCharacter(code)
		i, err := reader.NewInput(strings.NewReader(c.String()))
		assert.NoError(t, err)
		got, err := r.Read(i)
		if !assert.NoError(t, err, "reading %q for %#x", c.String(), code) {
			return
		}
		assert.Equal(t, c, got, "reading %q for %#x", c.String(), code)
	}

	// Every code point through the controls, punctuation, and common scripts
	for code := rune(0); code < 0x3100; code++ {
		check(code)
	}
	// Sample the rest to keep the test fast, skipping the surrogates
	for code := rune(0xe000); code < 0x10000; code += 7 {
		check(code)
	}
	// A prime step avoids lining up with block boundaries
	for code := rune(0x10000); code <= 0x10ffff; code += 251 {
		check(code)
	}
	// The sampling can miss the end
	check(0x10ffff)
	check(0x1f600)
}

func TestReaderCharacterErrors(t *testing.T) {
	testDefs := []struct {
		input   string
		wantErr string
	}{
		{`#\`, "end of file after #"},
		{`(#\`, "end of file after #"},
		{`'#\`, "end of file after #"},
		{`#\ab`, "unrecognized character name"},
		{`#\Foo`, "unrecognized character name"},
		{`#\Spac`, "unrecognized character name"},
		{`#\Spaces`, "unrecognized character name"},
		// The backslash escapes the first |, so the second starts a multiple escape
		{`#\|Space|`, "end of file in multiple escape"},
		// Escaped letters are part of the name
		{`#\a\b`, "unrecognized character name"},
		{`#\a|b|`, "unrecognized character name"},
		// The long s and Kelvin sign are not ASCII letters
		{"#\\\u017fpace", "unrecognized character name"},
		{"#\\Bac\u212aspace", "unrecognized character name"},
		{`#\a#`, "unrecognized character name"},
		{`#\a1`, "unrecognized character name"},
		{`#\U+`, "unrecognized character name"},
		{`#\U+G`, "unrecognized character name"},
		{`#\U+110000`, "unrecognized character name"},
		{`#\U+D800`, "unrecognized character name"},
		// A name is the whole token, so nothing can follow it
		{`#\Space1`, "unrecognized character name"},
		{`(#\Foo)`, "unrecognized character name"},
		// Characters after the first are read like any other token
		{"#\\a\x01", "invalid character"},
		{"#\\\xff", "invalid UTF-8"},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.ErrorContains(t, err, testDef.wantErr, "input: %q", testDef.input)
		assert.Nil(t, tmp, "input: %q", testDef.input)
		var readerErr reader.ReaderError
		assert.True(t, errors.As(err, &readerErr), "expected ReaderError for %q", testDef.input)
	}
}

func TestReaderRubout(t *testing.T) {
	sym := testSym
	testDefs := []struct {
		input string
		want  []types.Expression
	}{
		// Rubout separates tokens and is skipped between them
		{"a\x7fb", []types.Expression{sym("A"), sym("B")}},
		{"\x7fa\x7f", []types.Expression{sym("A")}},
		{"\x7f\x7f", []types.Expression{}},
		{"1\x7f2", []types.Expression{
			types.NewNumberInteger(big.NewInt(1)),
			types.NewNumberInteger(big.NewInt(2)),
		}},
		{"(a\x7fb)", []types.Expression{
			types.NewList([]types.Expression{sym("A"), sym("B")}),
		}},
		{"( \x7f)", []types.Expression{types.Nil}},
		{"(a\x7f)", []types.Expression{types.NewList([]types.Expression{sym("A")})}},
		{"'\x7fa", []types.Expression{types.NewLiteral(sym("A"))}},
		{"(\x7f(\x7f)\x7f)", []types.Expression{
			types.NewList([]types.Expression{types.Nil}),
		}},
		// Escaped, or inside a multiple escape, it is part of the token
		{"a\\\x7fb", []types.Expression{sym("A\x7fB")}},
		{"|a\x7fb|", []types.Expression{sym("a\x7fb")}},
	}
	r := newTestReader()
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		var got []types.Expression
		for {
			tmp, err := r.Read(i)
			if !assert.NoError(t, err, "unexpected error for %q", testDef.input) || tmp == nil {
				break
			}
			got = append(got, tmp)
		}
		// Compare lengths and elements so that nil and empty slices don't matter
		if !assert.Len(t, got, len(testDef.want), "wrong number of results for %q", testDef.input) {
			continue
		}
		for k := range testDef.want {
			assert.Equal(t, testDef.want[k], got[k], "wrong result %d for %q", k, testDef.input)
		}
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
				testSym("FOO"),
				testSym("BAR"),
			},
		},
		{
			input: `(foo "bar")`,
			expectedItems: []types.Expression{
				testSym("FOO"),
				types.NewString("bar"),
			},
		},
		{
			input: `(a#b c)`,
			expectedItems: []types.Expression{
				testSym("A#B"),
				testSym("C"),
			},
		},
		{
			input: `(||)`,
			expectedItems: []types.Expression{
				testSym(""),
			},
		},
		{
			input: `(a || b)`,
			expectedItems: []types.Expression{
				testSym("A"),
				testSym(""),
				testSym("B"),
			},
		},
	}
	r := newTestReader()
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
			expectedItem: testSym("FOO"),
		},
		{
			input: `'(foo bar)`,
			expectedItem: types.NewList(
				[]types.Expression{
					testSym("FOO"),
					testSym("BAR"),
				},
			),
		},
	}
	r := newTestReader()
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
			// A ratio equal to a whole number is an integer
			input:       `10/5`,
			expectedNum: types.NewNumberInteger(big.NewInt(2)),
		},
		{
			input:       `0.0`,
			expectedNum: singleFloat(0.0),
		},
		{
			input:       `0E0`,
			expectedNum: singleFloat(0.0),
		},
		{
			input:       `-.0`,
			expectedNum: singleFloat(math.Copysign(0, -1)),
		},
		{
			input:       `0.0s0`,
			expectedNum: singleFloat(0.0),
		},
		{
			input:       `0s0`,
			expectedNum: singleFloat(0.0),
		},
		{
			input:       `3.1415926535897932384d0`,
			expectedNum: doubleFloat(3.1415926535897932384e0),
		},
		{
			input:       `6.02E+23`,
			expectedNum: singleFloat(6.02e+23),
		},
		{
			input:       `602E+21`,
			expectedNum: singleFloat(602e+21),
		},
		{
			input:       `3.010299957f-1`,
			expectedNum: singleFloat(3.010299957e-1),
		},
		{
			input:       `-0.000000001s9`,
			expectedNum: singleFloat(-0.000000001e9),
		},
	}
	r := newTestReader()
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

func TestReaderFloatFormat(t *testing.T) {
	testDefs := []struct {
		input   string
		def     types.FloatFormat
		want    *types.Number
		wantStr string
	}{
		// Without a marker, or with e, the default format is used
		{`0.1`, types.FloatFormatSingle, singleFloat(0.1), ""},
		{`0.1`, types.FloatFormatDouble, doubleFloat(0.1), ""},
		{`1e5`, types.FloatFormatSingle, singleFloat(1e5), ""},
		{`1E5`, types.FloatFormatDouble, doubleFloat(1e5), ""},
		{`.1e0`, types.FloatFormatSingle, singleFloat(0.1), ""},
		{`.1e0`, types.FloatFormatDouble, doubleFloat(0.1), ""},
		// s and f are always single, d and l are always double
		{`0.1s0`, types.FloatFormatDouble, singleFloat(0.1), ""},
		{`0.1S0`, types.FloatFormatDouble, singleFloat(0.1), ""},
		{`0.1f0`, types.FloatFormatDouble, singleFloat(0.1), ""},
		{`0.1F0`, types.FloatFormatDouble, singleFloat(0.1), ""},
		{`0.1d0`, types.FloatFormatSingle, doubleFloat(0.1), ""},
		{`0.1D0`, types.FloatFormatSingle, doubleFloat(0.1), ""},
		{`0.1l0`, types.FloatFormatSingle, doubleFloat(0.1), ""},
		{`0.1L0`, types.FloatFormatSingle, doubleFloat(0.1), ""},
		{`1d0`, types.FloatFormatSingle, doubleFloat(1), ""},
		{`-1.5f2`, types.FloatFormatDouble, singleFloat(-150), ""},
		// Within a list
		{`(1.5d0)`, types.FloatFormatSingle, nil, "(1.5)"},
	}
	for _, testDef := range testDefs {
		r := newTestReader()
		r.DefaultFloatFormat = testDef.def
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		if !assert.NoError(t, err, "unexpected error for %q", testDef.input) {
			continue
		}
		if testDef.want == nil {
			items, ok := types.ListItems(tmp)
			if assert.True(t, ok) && assert.Len(t, items, 1) {
				assert.Equal(t, doubleFloat(1.5), items[0])
			}
			continue
		}
		assert.Equal(t, testDef.want, tmp, "wrong result for %q with default %s", testDef.input, testDef.def)
	}

	// The standard default is single-float
	assert.Equal(t, types.FloatFormatSingle, newTestReader().DefaultFloatFormat)
}

func TestReaderUnknownFloatFormat(t *testing.T) {
	r := newTestReader()
	r.DefaultFloatFormat = types.FloatFormatNone

	// Floats that need the default are a reader error, not a panic
	for _, input := range []string{"1.5", "1e5", "(a 1.5e0)", ".5"} {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		var tmp types.Expression
		assert.NotPanics(t, func() { tmp, err = r.Read(i) }, input)
		var readerErr reader.ReaderError
		assert.True(t, errors.As(err, &readerErr), "expected ReaderError for %q", input)
		assert.ErrorContains(t, err, "unknown default float format", input)
		assert.Nil(t, tmp)
	}

	// Everything else still reads
	for input, want := range map[string]types.Expression{
		"1":     types.NewNumberInteger(big.NewInt(1)),
		"1/2":   types.NewNumberRatio(big.NewRat(1, 2)),
		"1.5d0": doubleFloat(1.5),
		"1.5f0": singleFloat(1.5),
		"a":     testSym("A"),
	} {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, input)
		assert.Equal(t, want, got, input)
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
				testSym(`+`),
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
				testSym(`FOO`),
			}),
		},
	}
	r := newTestReader()
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
	r := newTestReader()

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
	r := newTestReader()

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
		// A trailing decimal point makes a decimal integer in any radix
		{input: `#x10.`, expectedNum: types.NewNumberInteger(big.NewInt(10))},
		{input: `#b10.`, expectedNum: types.NewNumberInteger(big.NewInt(10))},
		{input: `#16r-10.`, expectedNum: types.NewNumberInteger(big.NewInt(-10))},
		// A ratio equal to a whole number is an integer
		{input: `#x4/2`, expectedNum: types.NewNumberInteger(big.NewInt(2))},
		{input: `#b110/11`, expectedNum: types.NewNumberInteger(big.NewInt(2))},
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
		inputs := []string{"#xz", "#2rabc", "#x", "#2r", "#2r ", "#xF.F", "#xFF.", "#x1.5", "#10r1.5", "#10r1e5", "#xF/F.", "#1r0", "#37r0", "#999999999999999999999999r0", "#x1/0", "#2r-1/0"}
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
	r := newTestReader()

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
			testSym("FOO"),
			testSym("BAR"),
		}), tmp)
	})

	t.Run("trailing block comment inside list", func(t *testing.T) {
		i, err := reader.NewInput(strings.NewReader("(foo #| comment |#)"))
		assert.NoError(t, err)

		tmp, err := r.Read(i)
		assert.NoError(t, err)
		assert.Equal(t, types.NewList([]types.Expression{
			testSym("FOO"),
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

		tmp, err := newTestReader().Read(i)
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
		// Floats out of range are errors, not symbols
		{input: "1e999999999", wantLine: 1, wantOffset: 11},
		{input: "(1e-999999999 a)", wantLine: 1, wantOffset: 13},
		{input: "1e99999999999999999999", wantLine: 1, wantOffset: 22},
		// The range depends on the format, which is single by default
		{input: "3.5e38", wantLine: 1, wantOffset: 6},
		{input: "1f39", wantLine: 1, wantOffset: 4},
		{input: "1e-46", wantLine: 1, wantOffset: 5},
		{input: "1d309", wantLine: 1, wantOffset: 5},
		// Only ASCII digits are digits in a dispatch macro's numeric argument
		{input: "#\u0663r10", wantLine: 1, wantOffset: 2},
		{input: "#1\u0663r10", wantLine: 1, wantOffset: 3},
		{input: "#\uff11r10", wantLine: 1, wantOffset: 2},
		// Vector errors
		{input: "#(", wantLine: 1, wantOffset: 2},
		{input: "#(a b", wantLine: 1, wantOffset: 5},
		{input: "#(.)", wantLine: 1, wantOffset: 3},
		{input: "#(a . b)", wantLine: 1, wantOffset: 5},
		{input: "#3()", wantLine: 1, wantOffset: 4},
		{input: "#2(a b c)", wantLine: 1, wantOffset: 9},
		{input: "#3a", wantLine: 1, wantOffset: 3},
		{input: "#3", wantLine: 1, wantOffset: 2},
		{input: "(a\n#(b\n", wantLine: 3, wantOffset: 1},
		// Character errors
		{input: `#\`, wantLine: 1, wantOffset: 2},
		{input: `#\ab`, wantLine: 1, wantOffset: 4},
		{input: `(#\Foo)`, wantLine: 1, wantOffset: 6},
		{input: "(a\n#\\Foo)", wantLine: 2, wantOffset: 5},
		{input: "#p", wantLine: 1, wantOffset: 2},
		{input: "\n#p", wantLine: 2, wantOffset: 2},
		{input: "#1r0", wantLine: 1, wantOffset: 3},
		{input: "#37r0", wantLine: 1, wantOffset: 4},
		{input: "#99999999999999999999r0", wantLine: 1, wantOffset: 22},
	}
	for _, testDef := range testDefs {
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)

		tmp, err := newTestReader().Read(i)
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
	sym := testSym
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
		{`(a .5 b)`, list(exprs(sym("A"), singleFloat(0.5), sym("B")))},
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
	r := newTestReader()
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
	r := newTestReader()
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

func TestReaderInterning(t *testing.T) {
	ps := types.NewPackages()
	r := reader.NewReader(ps)
	assert.Same(t, ps.User(), r.Package)

	read := func(r *reader.Reader, input string) types.Expression {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, "input %q", input)
		return got
	}

	// The same name is the same symbol, however it is written
	a := read(r, "foo")
	assert.Same(t, a, read(r, "FOO"))
	assert.Same(t, a, read(r, "|FOO|"))
	assert.NotSame(t, a, read(r, "|foo|"))
	assert.Same(t, ps.User(), a.(*types.Symbol).Package())
	found, _ := ps.User().Find("FOO")
	assert.Same(t, a, found)

	// A symbol in a list is the same object as the one read alone
	list := read(r, "(foo)").(*types.Cons)
	assert.Same(t, a, list.Car())

	// Readers on one set share symbols, and the current package decides where
	// a new symbol goes
	other := reader.NewReader(ps)
	assert.Same(t, a, read(other, "foo"))
	pkg, err := ps.Make("OTHER", nil, nil)
	assert.NoError(t, err)
	other.Package = pkg
	b := read(other, "foo")
	assert.NotSame(t, a, b)
	assert.Same(t, pkg, b.(*types.Symbol).Package())

	// A separate set has separate symbols
	assert.NotSame(t, a, read(reader.NewReader(types.NewPackages()), "foo"))
}

func TestReaderNil(t *testing.T) {
	ps := types.NewPackages()
	empty, err := ps.Make("EMPTY", nil, nil)
	assert.NoError(t, err)

	testDefs := []struct {
		pkg   *types.Package
		input string
		isNil bool
	}{
		{ps.User(), "nil", true},
		{ps.User(), "NIL", true},
		{ps.User(), "()", true},
		{ps.User(), "t", false},
		{ps.User(), "|nil|", false},
		{ps.User(), "nil2", false},
		{ps.CommonLisp(), "nil", true},
		// Without the COMMON-LISP symbols, NIL is an ordinary symbol
		{empty, "nil", false},
	}
	for _, testDef := range testDefs {
		r := reader.NewReader(ps)
		r.Package = testDef.pkg
		i, err := reader.NewInput(strings.NewReader(testDef.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err)
		if testDef.isNil {
			assert.Same(t, types.Nil, got, "input %q", testDef.input)
		} else {
			assert.NotSame(t, types.Nil, got, "input %q", testDef.input)
			assert.Equal(t, types.ExpressionTypeSymbol, got.Type(), "input %q", testDef.input)
		}
	}

	// NIL inside a list is Nil, so (a nil) has a Nil element and (a . nil) is (a)
	r := newTestReader()
	i, err := reader.NewInput(strings.NewReader("(a nil) (a . nil)"))
	assert.NoError(t, err)
	got, err := r.Read(i)
	assert.NoError(t, err)
	items, _ := types.ListItems(got)
	assert.Len(t, items, 2)
	assert.Same(t, types.Nil, items[1])
	got, err = r.Read(i)
	assert.NoError(t, err)
	items, _ = types.ListItems(got)
	assert.Len(t, items, 1)
}

func TestReaderPackageNotInSet(t *testing.T) {
	foreign := types.NewPackages().User()
	for name, set := range map[string]func(r *reader.Reader){
		"nil package":     func(r *reader.Reader) { r.Package = nil },
		"nil set":         func(r *reader.Reader) { r.Packages = nil },
		"foreign package": func(r *reader.Reader) { r.Package = foreign },
	} {
		r := newTestReader()
		set(r)
		for _, input := range []string{"a", "(a)", "#"} {
			i, err := reader.NewInput(strings.NewReader(input))
			assert.NoError(t, err)
			_, err = r.Read(i)
			assert.ErrorContains(t, err, "current package is not in the package set", "%s: %q", name, input)
		}
		// Input with no symbols still reads
		i, err := reader.NewInput(strings.NewReader(`1 "s"`))
		assert.NoError(t, err)
		_, err = r.Read(i)
		assert.NoError(t, err, name)
	}
}

func TestReaderPackageMarkers(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)
	exported, _ := lib.Intern("PUB")
	assert.NoError(t, lib.Export(exported))
	private, _ := lib.Intern("PRIV")
	// A lowercase name that case conversion can't reach without an escape
	lower, _ := lib.Intern("lower")
	assert.NoError(t, lib.Export(lower))
	kw, _ := ps.Keyword().Intern("FOO")
	emptyKw, _ := ps.Keyword().Intern("")
	empty, _ := lib.Intern("")
	assert.NoError(t, lib.Export(empty))
	user, _ := ps.User().Intern("X")
	clT, _ := ps.CommonLisp().Find("T")

	read := func(input string) (types.Expression, error) {
		r := reader.NewReader(ps)
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		return r.Read(i)
	}

	// Symbols that aren't in the table are identified by name and package
	type sameName struct {
		name string
		pkg  *types.Package
	}
	valid := []struct {
		input string
		want  any
	}{
		{"lib:pub", exported},
		{"LIB:PUB", exported},
		{"lib::pub", exported},
		{"lib::priv", private},
		{"|LIB|:pub", exported},
		{"lib:|lower|", lower},
		{":foo", kw},
		{":FOO", kw},
		{"keyword:foo", kw},
		{"keyword::foo", kw},
		{"cl-user::x", user},
		{"common-lisp:nil", types.Nil},
		{"cl:nil", types.Nil},
		{"cl::nil", types.Nil},
		{"cl:t", clT},
		// An empty escape is a name with no characters
		{":||", emptyKw},
		{"keyword:||", emptyKw},
		{"lib:||", empty},
		{"lib::||", empty},
		// Escaped and quoted markers are part of the name
		{`|a:b|`, sameName{"a:b", ps.User()}},
		{`a\:b`, sameName{"A:B", ps.User()}},
		{`a\:\:b`, sameName{"A::B", ps.User()}},
		// Not numbers
		{":1", sameName{"1", ps.Keyword()}},
		{"lib::1", sameName{"1", lib}},
	}
	for _, tc := range valid {
		got, err := read(tc.input)
		assert.NoError(t, err, "input %q", tc.input)
		if want, ok := tc.want.(sameName); ok {
			sym, ok := got.(*types.Symbol)
			if assert.True(t, ok, "input %q", tc.input) {
				assert.Equal(t, want.name, sym.Name(), "input %q", tc.input)
				assert.Same(t, want.pkg, sym.Package(), "input %q", tc.input)
			}
			continue
		}
		assert.Same(t, tc.want, got, "input %q", tc.input)
	}

	invalid := []struct {
		input string
		msg   string
	}{
		{"lib:priv", "not external"},
		{"lib:missing", "not external"},
		{"keyword:missing", "not external"},
		{"nope:x", "not found"},
		{"nope::x", "not found"},
		{"||:foo", "package \"\" not found"},
		{"||::foo", "package \"\" not found"},
		{"lib:", "symbol name missing"},
		{"lib:|", "end of file in multiple escape"},
		{"lib::", "symbol name missing"},
		{":", "symbol name missing"},
		{"::", "symbol name missing"},
		{"::x", "package name missing"},
		{":::x", "too many package markers"},
		{"lib:::x", "too many package markers"},
		{"a:b:c", "too many package markers"},
		{"a::b:c", "too many package markers"},
		{"a:b::c", "too many package markers"},
		{"lib:.", "only of dots"},
		{"lib:..", "only of dots"},
		// Inherited from another package is not external in this one
		{"cl-user:nil", "inherited, not external"},
	}
	for _, tc := range invalid {
		_, err := read(tc.input)
		assert.ErrorContains(t, err, tc.msg, "input %q", tc.input)
		var rerr reader.ReaderError
		assert.ErrorAs(t, err, &rerr, "input %q", tc.input)
	}

	// The double colon interns, so the symbol is present afterwards
	_, err = read("lib::tmp")
	assert.NoError(t, err)
	_, status := lib.Find("TMP")
	assert.Equal(t, types.SymbolStatusInternal, status)
}

func TestReaderUninterned(t *testing.T) {
	read := func(r *reader.Reader, input string) (types.Expression, error) {
		i, err := reader.NewInput(strings.NewReader(input))
		assert.NoError(t, err)
		return r.Read(i)
	}

	valid := []struct {
		input string
		name  string
	}{
		{"#:foo", "FOO"},
		{"#:FOO", "FOO"},
		{"#:|foo|", "foo"},
		{`#:|a|b`, "aB"},
		{`#:a\:b`, "A:B"},
		{`#:|a:b|`, "a:b"},
		{`#:a\:\:b`, "A::B"},
		{"#:||", ""},
		// Not numbers
		{"#:1", "1"},
		{"#:1.5", "1.5"},
		{"#:nil", "NIL"},
		{"#:t", "T"},
		// A character that is a macro at the start of a token is part of the name
		{"#:#a", "#A"},
		{"#:a#b", "A#B"},
		// The token ends where any other would
		{"#:a b", "A"},
		{"#:a(", "A"},
		{`#:a"`, "A"},
		{"#:a;b", "A"},
	}
	for _, tc := range valid {
		r := newTestReader()
		got, err := read(r, tc.input)
		assert.NoError(t, err, "input %q", tc.input)
		sym, ok := got.(*types.Symbol)
		if !assert.True(t, ok, "input %q", tc.input) {
			continue
		}
		assert.Equal(t, tc.name, sym.Name(), "input %q", tc.input)
		assert.Nil(t, sym.Package(), "input %q", tc.input)
	}

	// Each read is a different symbol, and none is the interned one
	r := newTestReader()
	interned, err := read(r, "foo")
	assert.NoError(t, err)
	a, err := read(r, "#:foo")
	assert.NoError(t, err)
	b, err := read(r, "#:foo")
	assert.NoError(t, err)
	assert.NotSame(t, a, b)
	assert.NotSame(t, interned, a)
	// The package is untouched
	found, _ := testPackages.User().Find("FOO")
	assert.Same(t, interned, found)

	// In a list, with the rest of the list still read
	got, err := read(r, "(#:a #:b)")
	assert.NoError(t, err)
	items, _ := types.ListItems(got)
	assert.Len(t, items, 2)

	invalid := []struct {
		input string
		msg   string
	}{
		{"#:", "symbol name missing"},
		{"#: foo", "symbol name missing"},
		{"#:\n", "symbol name missing"},
		{"#:(a)", "symbol name missing"},
		{"#:)", "symbol name missing"},
		{`#:"a"`, "symbol name missing"},
		{"#:;a", "symbol name missing"},
		{"#:'a", "symbol name missing"},
		{"#:a:b", "package marker"},
		{"#:a::b", "package marker"},
		{"#::a", "package marker"},
		{"#:a:", "package marker"},
		{"#::", "package marker"},
		{"#:.", "only of dots"},
		{"#:...", "only of dots"},
		{"#:a|b", "end of file in multiple escape"},
		{"#:\x01", "invalid character"},
	}
	for _, tc := range invalid {
		_, err := read(newTestReader(), tc.input)
		assert.ErrorContains(t, err, tc.msg, "input %q", tc.input)
		var rerr reader.ReaderError
		assert.ErrorAs(t, err, &rerr, "input %q", tc.input)
	}
}

// TestReaderFormatRoundTrip checks that every symbol with a short name made of
// characters that need care when printing reads back as the same symbol.
func TestReaderFormatRoundTrip(t *testing.T) {
	ps := types.NewPackages()
	lib, err := ps.Make("LIB", nil, nil)
	assert.NoError(t, err)
	// Uses lib, so that what lib exports is accessible bare
	both, err := ps.Make("BOTH", nil, []*types.Package{ps.CommonLisp(), lib})
	assert.NoError(t, err)

	// Lowercase, a digit, signs, dots, an exponent marker, a ratio marker, the
	// start of a dispatch macro, the escapes, the package marker, and a space
	// and characters that are invalid in the readtable
	alphabet := []string{"a", "A", "1", "+", ".", "e", "/", "#", "|", `\`, ":", " ", "(", "\x01", "\u00a0", "\u200d"}
	names := []string{""}
	for _, first := range alphabet {
		names = append(names, first)
		for _, second := range alphabet {
			names = append(names, first+second)
			for _, third := range alphabet {
				names = append(names, first+second+third)
			}
		}
	}

	read := func(current *types.Package, text string) types.Expression {
		r := reader.NewReader(ps)
		r.Package = current
		i, err := reader.NewInput(strings.NewReader(text))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err, "text %q", text)
		return got
	}

	// Package names go through the same escaping
	for _, name := range names[1:] {
		pkg, err := ps.Make(name, nil, nil)
		assert.NoError(t, err, "package %q", name)
		sym, _ := pkg.Intern("A")
		assert.NoError(t, pkg.Export(sym))
		text := types.Format(sym, nil)
		assert.Same(t, sym, read(ps.User(), text), "package %q: %q", name, text)
		internal, _ := pkg.Intern("B")
		text = types.Format(internal, nil)
		assert.Same(t, internal, read(ps.User(), text), "package %q: %q", name, text)
	}

	for n, name := range names {
		interned, _ := lib.Intern(name)
		if n%2 == 0 {
			assert.NoError(t, lib.Export(interned))
		}
		syms := []*types.Symbol{types.NewSymbol(name), interned}
		if name != "" {
			kw, _ := ps.Keyword().Intern(name)
			syms = append(syms, kw)
		}
		for _, sym := range syms {
			for _, current := range []*types.Package{lib, both, ps.User()} {
				text := types.Format(sym, current)
				got := read(current, text)
				if sym.Package() == nil {
					// A new symbol each time, so only the name can match
					gotSym, ok := got.(*types.Symbol)
					assert.True(t, ok && gotSym.Name() == name && gotSym.Package() == nil, "%q in %s: got %v", text, current.Name(), got)
					continue
				}
				if !assert.Same(t, sym, got, "%q in %s", text, current.Name()) {
					t.FailNow()
				}
			}
		}
	}
}

// TestReaderFormatEveryCharacter checks every code point in a symbol name and a
// string, so that what Format escapes can't drift from what the standard
// readtable accepts.
func TestReaderFormatEveryCharacter(t *testing.T) {
	if testing.Short() {
		t.Skip("reads over a million code points")
	}
	r := newTestReader()
	for c := rune(0); c <= unicode.MaxRune; c++ {
		if !utf8.ValidRune(c) {
			continue
		}
		name := "A" + string(c) + "B"
		text := types.Format(types.NewSymbol(name), nil)
		i, err := reader.NewInput(strings.NewReader(text))
		assert.NoError(t, err)
		got, err := r.Read(i)
		// Stop at the first one, as a mismatch would fail for thousands
		if !assert.NoError(t, err, "character %#x: %q", c, text) {
			t.FailNow()
		}
		sym, ok := got.(*types.Symbol)
		if !assert.True(t, ok && sym.Name() == name && sym.Package() == nil, "character %#x: %q read as %v", c, text, got) {
			t.FailNow()
		}

		// A string with the same character
		text = types.Format(types.NewString(name), nil)
		i, err = reader.NewInput(strings.NewReader(text))
		assert.NoError(t, err)
		got, err = r.Read(i)
		if !assert.NoError(t, err, "character %#x: %q", c, text) {
			t.FailNow()
		}
		str, ok := got.(*types.String)
		if !assert.True(t, ok && str.Value() == name, "character %#x: %q read as %v", c, text, got) {
			t.FailNow()
		}
	}
}
