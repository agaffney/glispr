package reader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

var testPackages = types.NewPackages()

func newTestReader() *Reader { return NewReader(testPackages) }

func TestReaderCaseUpcaseEscapes(t *testing.T) {
	r := newTestReader()
	testCases := []struct {
		input string
		want  string
	}{
		{input: `FROBBOZ`, want: `FROBBOZ`},
		{input: `frobboz`, want: `FROBBOZ`},
		{input: `fRObBoz`, want: `FROBBOZ`},
		{input: `|frobboz|`, want: `frobboz`},
		{input: `|FrobBoz|`, want: `FrobBoz`},
		{input: `\frobboz`, want: `fROBBOZ`},
		{input: `foo\bar`, want: `FOObAR`},
		// Escaped and unescaped parts in the same token
		{input: `|a|b`, want: `aB`},
		{input: `a|b|c`, want: `AbC`},
		{input: `a\bc`, want: `AbC`},
		{input: `a\Bc`, want: `ABC`},
	}
	for _, tc := range testCases {
		i, err := NewInput(strings.NewReader(tc.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err)
		sym, ok := got.(*types.Symbol)
		assert.True(t, ok, "expected symbol for %q", tc.input)
		if ok {
			assert.Equal(t, tc.want, sym.Name(), "input %q", tc.input)
		}
	}
}

func TestReaderCaseDowncase(t *testing.T) {
	r := newTestReader()
	r.rt.CaseMode = ReadtableCaseModeDowncase
	testCases := []struct {
		input string
		want  string
	}{
		{input: `FROBBOZ`, want: `frobboz`},
		{input: `frobboz`, want: `frobboz`},
		{input: `|FrobBoz|`, want: `FrobBoz`},
		{input: `\Frobboz`, want: `Frobboz`},
		{input: `A\Bc`, want: `aBc`},
		{input: `|A|B`, want: `Ab`},
	}
	for _, tc := range testCases {
		i, err := NewInput(strings.NewReader(tc.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err)
		sym, ok := got.(*types.Symbol)
		assert.True(t, ok, "expected symbol for %q", tc.input)
		if ok {
			assert.Equal(t, tc.want, sym.Name(), "input %q", tc.input)
		}
	}
}

func TestReaderCaseInvert(t *testing.T) {
	r := newTestReader()
	r.rt.CaseMode = ReadtableCaseModeInvert
	testCases := []struct {
		input string
		want  string
	}{
		{input: `foobar`, want: `FOOBAR`},
		{input: `FOOBAR`, want: `foobar`},
		{input: `FooBar`, want: `FooBar`},
		{input: `|FooBar|`, want: `FooBar`},
		{input: `\foobar`, want: `fOOBAR`},
		// Only unescaped letters decide whether to invert, escaped ones are kept
		{input: `\Foo`, want: `FOO`},
		{input: `|a|B`, want: `ab`},
		{input: `a\Bc`, want: `ABC`},
		{input: `A\bC`, want: `abc`},
		{input: `Fo\O`, want: `FoO`},
	}
	for _, tc := range testCases {
		i, err := NewInput(strings.NewReader(tc.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err)
		sym, ok := got.(*types.Symbol)
		assert.True(t, ok, "expected symbol for %q", tc.input)
		if ok {
			assert.Equal(t, tc.want, sym.Name(), "input %q", tc.input)
		}
	}
}

func TestReaderCasePreserve(t *testing.T) {
	r := newTestReader()
	r.rt.CaseMode = ReadtableCaseModePreserve
	testCases := []struct {
		input string
		want  string
	}{
		{input: `FooBar`, want: `FooBar`},
		{input: `foobar`, want: `foobar`},
		{input: `|FooBar|`, want: `FooBar`},
	}
	for _, tc := range testCases {
		i, err := NewInput(strings.NewReader(tc.input))
		assert.NoError(t, err)
		got, err := r.Read(i)
		assert.NoError(t, err)
		sym, ok := got.(*types.Symbol)
		assert.True(t, ok, "expected symbol for %q", tc.input)
		if ok {
			assert.Equal(t, tc.want, sym.Name(), "input %q", tc.input)
		}
	}
}

// Characters keep their case in every case mode, and names ignore it.
func TestReaderCharacterCase(t *testing.T) {
	for _, mode := range []ReadtableCaseMode{
		ReadtableCaseModeUpcase,
		ReadtableCaseModeDowncase,
		ReadtableCaseModePreserve,
		ReadtableCaseModeInvert,
	} {
		r := newTestReader()
		r.rt.CaseMode = mode
		testCases := []struct {
			input string
			want  rune
		}{
			{input: `#\a`, want: 'a'},
			{input: `#\A`, want: 'A'},
			{input: `#\é`, want: 'é'},
			{input: `#\SPACE`, want: ' '},
			{input: `#\space`, want: ' '},
			{input: `#\Space`, want: ' '},
			{input: `#\nEwLiNe`, want: '\n'},
		}
		for _, tc := range testCases {
			i, err := NewInput(strings.NewReader(tc.input))
			assert.NoError(t, err)
			got, err := r.Read(i)
			assert.NoError(t, err, "input %q, mode %d", tc.input, mode)
			assert.Equal(t, types.NewCharacter(tc.want), got, "input %q, mode %d", tc.input, mode)
		}
	}
}
