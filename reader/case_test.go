package reader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestReaderCaseUpcaseEscapes(t *testing.T) {
	r := NewReader()
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
	r := NewReader()
	r.rt.CaseMode = ReadtableCaseModeDowncase
	testCases := []struct {
		input string
		want  string
	}{
		{input: `FROBBOZ`, want: `frobboz`},
		{input: `frobboz`, want: `frobboz`},
		{input: `|FrobBoz|`, want: `FrobBoz`},
		{input: `\Frobboz`, want: `Frobboz`},
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
	r := NewReader()
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
	r := NewReader()
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
