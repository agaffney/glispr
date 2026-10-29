package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestCharacter(t *testing.T) {
	c := types.NewCharacter('a')
	assert.Equal(t, types.ExpressionTypeCharacter, c.Type())
	assert.Equal(t, 'a', c.Rune())
	assert.Equal(t, types.NewCharacter('a'), c)
	assert.NotEqual(t, types.NewCharacter('A'), c)

	assert.Equal(t, rune(0x10ffff), types.NewCharacter(0x10ffff).Rune())

	// Below zero, the surrogates, and above the largest code point
	for _, r := range []rune{-1, 0xd800, 0xdfff, 0x110000} {
		assert.Panics(t, func() { types.NewCharacter(r) }, "NewCharacter(%#x)", r)
	}
}

func TestCharacterString(t *testing.T) {
	testDefs := []struct {
		r    rune
		want string
	}{
		{'a', `#\a`},
		{'A', `#\A`},
		{'1', `#\1`},
		{'(', `#\(`},
		{')', `#\)`},
		{'\\', `#\\`},
		{'"', `#\"`},
		{'é', `#\é`},
		{'λ', `#\λ`},
		{0x1f600, "#\\\U0001f600"},
		// Characters with names
		{' ', `#\Space`},
		{'\n', `#\Newline`},
		{'\t', `#\Tab`},
		{'\f', `#\Page`},
		{'\r', `#\Return`},
		{0x7f, `#\Rubout`},
		{'\b', `#\Backspace`},
		// No name and not printable, so the U+ form
		{0, `#\U+0000`},
		{1, `#\U+0001`},
		{0x1b, `#\U+001B`},
		{0x85, `#\U+0085`},
		{0xa0, `#\U+00A0`},
		{0x200b, `#\U+200B`},
		{0x200d, `#\U+200D`},
		{0x10ffff, `#\U+10FFFF`},
	}
	for _, testDef := range testDefs {
		assert.Equal(t, testDef.want, types.NewCharacter(testDef.r).String(), "character %#x", testDef.r)
	}
}

func TestCharacterName(t *testing.T) {
	assert.Equal(t, "Space", types.NewCharacter(' ').Name())
	// Newline is the name used when a character has two
	assert.Equal(t, "Newline", types.NewCharacter('\n').Name())
	assert.Equal(t, "", types.NewCharacter('a').Name())
	assert.Equal(t, "", types.NewCharacter(0).Name())
}

func TestCharacterFromName(t *testing.T) {
	valid := []struct {
		name string
		want rune
	}{
		{"Space", ' '},
		{"SPACE", ' '},
		{"space", ' '},
		{"sPaCe", ' '},
		{"Newline", '\n'},
		{"newline", '\n'},
		{"Linefeed", '\n'},
		{"LINEFEED", '\n'},
		{"Tab", '\t'},
		{"Page", '\f'},
		{"Return", '\r'},
		{"Rubout", 0x7f},
		{"Backspace", '\b'},
		{"U+41", 'A'},
		{"U+0041", 'A'},
		{"u+0041", 'A'},
		{"nEwLiNe", '\n'},
		{"U+e9", 'é'},
		{"U+E9", 'é'},
		{"U+0", 0},
		{"U+1F600", 0x1f600},
		{"U+10FFFF", 0x10ffff},
		{"U+000041", 'A'},
	}
	for _, tc := range valid {
		got, ok := types.CharacterFromName(tc.name)
		assert.True(t, ok, "CharacterFromName(%q)", tc.name)
		assert.Equal(t, tc.want, got, "CharacterFromName(%q)", tc.name)
	}

	invalid := []string{
		"",
		"a",
		"Foo",
		"Spac",
		"Spaces",
		" Space",
		"Space ",
		"Nul",
		"U",
		"U+",
		"U+G",
		"U+4G",
		"U+ 41",
		"U+-1",
		"U++41",
		"U+0x41",
		// Too large, surrogates, and too many digits
		"U+110000",
		"U+D800",
		"U+DFFF",
		"U+0000041",
		"U+FFFFFFFF",
		"V+41",
		"41",
		// Only ASCII letters ignore case
		"\u017fpace",     // long s
		"Bac\u212aspace", // Kelvin sign
	}
	for _, name := range invalid {
		got, ok := types.CharacterFromName(name)
		assert.False(t, ok, "CharacterFromName(%q) = %#x", name, got)
	}

	// Every printed name reads back as the same character
	for _, r := range []rune{' ', '\n', '\t', '\f', '\r', 0x7f, '\b'} {
		c := types.NewCharacter(r)
		got, ok := types.CharacterFromName(c.Name())
		assert.True(t, ok)
		assert.Equal(t, r, got)
	}
}
