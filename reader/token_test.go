package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstituentTraits(t *testing.T) {
	testCases := []struct {
		r    rune
		want traitSet
	}{
		{'a', traitAlphabetic | traitDigit},
		{'Z', traitAlphabetic | traitDigit},
		{'e', traitAlphabetic | traitDigit | traitExponentMarker},
		{'D', traitAlphabetic | traitDigit | traitExponentMarker},
		{'0', traitAlphabetic | traitDigit},
		{'9', traitAlphabetic | traitDigit},
		{'+', traitAlphabetic | traitPlusSign},
		{'-', traitAlphabetic | traitMinusSign},
		{'.', traitAlphabetic | traitDot | traitDecimalPoint},
		{'/', traitAlphabetic | traitRatioMarker},
		{':', traitPackageMarker},
		{'!', traitAlphabetic},
		{'~', traitAlphabetic},
		{'é', traitAlphabetic},
		{'\b', traitInvalid},
		{0x7f, traitInvalid},
		{' ', traitInvalid},
		{'\n', traitInvalid},
		{'\t', traitInvalid},
		{'\r', traitInvalid},
		{'\f', traitInvalid},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.want, constituentTraits(tc.r), "wrong traits for %q", tc.r)
	}
}

func TestTokenAdd(t *testing.T) {
	tok := &token{}
	tok.add('.', false)
	tok.add('.', true)
	tok.add('1', true)
	tok.add('e', false)

	assert.Equal(t, 4, len(tok.chars))
	assert.Equal(t, "..1e", tok.String())
	assert.Equal(t, []tokenChar{
		{r: '.', escaped: false, traits: traitAlphabetic | traitDot | traitDecimalPoint},
		// Escaped characters only have the alphabetic trait
		{r: '.', escaped: true, traits: traitAlphabetic},
		{r: '1', escaped: true, traits: traitAlphabetic},
		{r: 'e', escaped: false, traits: traitAlphabetic | traitDigit | traitExponentMarker},
	}, tok.chars)
}

func TestTokenHasEscape(t *testing.T) {
	tok := &token{}
	tok.add('a', false)
	assert.False(t, tok.hasEscape())

	// Adding an escaped character records the escape
	tok.add('b', true)
	assert.True(t, tok.hasEscape())

	// An escape counts even if it contributed no characters
	empty := &token{}
	assert.False(t, empty.hasEscape())
	empty.sawEscape = true
	assert.True(t, empty.hasEscape())
}

func TestTokenStarted(t *testing.T) {
	tok := &token{}
	assert.False(t, tok.started())

	// An empty multiple escape starts a token without adding characters
	tok.sawEscape = true
	assert.True(t, tok.started())

	tok = &token{}
	tok.add('a', false)
	assert.True(t, tok.started())
}

func TestTokenDots(t *testing.T) {
	build := func(escapedFlags string) *token {
		tok := &token{}
		for _, f := range escapedFlags {
			tok.add('.', f == 'e')
		}
		return tok
	}

	// u = unescaped dot, e = escaped dot
	assert.True(t, build("u").allDots())
	assert.True(t, build("u").isSingleDot())
	assert.True(t, build("uu").allDots())
	assert.False(t, build("uu").isSingleDot())

	// An escaped dot is only alphabetic, so it is neither
	assert.False(t, build("e").allDots())
	assert.False(t, build("e").isSingleDot())
	assert.False(t, build("ue").allDots())

	// Other characters and empty tokens are not dots
	other := &token{}
	other.add('.', false)
	other.add('a', false)
	assert.False(t, other.allDots())
	assert.False(t, (&token{}).allDots())
	assert.False(t, (&token{}).isSingleDot())
}
