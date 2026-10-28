package types_test

import (
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestNewNumberFromString(t *testing.T) {
	testDefs := []struct {
		val         string
		radix       int
		expectedNum *types.Number
	}{
		{val: "123", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(123))},
		{val: "0", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(0))},
		{val: "-0", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(0))},
		{val: "+6", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(6))},
		{val: "+28", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(28))},
		{val: "-1", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(-1))},
		{val: "15511210043330985984000000", radix: 10, expectedNum: types.NewNumberInteger(new(big.Int).MulRange(1, 25))},
		{val: "11010101", radix: 2, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "325", radix: 8, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "D5", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "10.", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(10))},
		{val: "-7H", radix: 25, expectedNum: types.NewNumberInteger(big.NewInt(-192))},
		{val: "ACCEDED", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(181202413))},
		{val: "2/3", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(2, 3))},
		{val: "4/6", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(4, 6))},
		{val: "-17/23", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(-17, 23))},
		{val: "10/5", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(2))},
		{val: "-101/75", radix: 8, expectedNum: types.NewNumberRatio(big.NewRat(-65, 61))},
		{val: "120/21", radix: 3, expectedNum: types.NewNumberRatio(big.NewRat(15, 7))},
		{val: "bc/ad", radix: 16, expectedNum: types.NewNumberRatio(big.NewRat(188, 173))},
		{val: "FADED/FACADE", radix: 16, expectedNum: types.NewNumberRatio(big.NewRat(1027565, 16435934))},
		{val: "0.0", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(0.0))},
		{val: "0E0", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(0.0))},
		{val: "3.1415926535897932384", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(3.1415926535897932384e0))},
	}
	for _, testDef := range testDefs {
		tmp := types.NewNumberFromString(testDef.val, testDef.radix)
		if tmp == nil {
			t.Errorf("NewNumberFromString(%q, %d) returned nil", testDef.val, testDef.radix)
			continue
		}
		if !tmp.Equal(testDef.expectedNum) {
			t.Errorf("NewNumberFromString(%q, %d) does not match expected value\n\n  Expected: %s\n  Actual: %s", testDef.val, testDef.radix, testDef.expectedNum.String(), tmp.String())
		}
	}
}

func TestNewNumberFromStringZeroDenominator(t *testing.T) {
	for _, val := range []string{"1/0", "-1/0", "0/0"} {
		assert.NotPanics(t, func() {
			assert.Nil(t, types.NewNumberFromString(val, 10), "expected %q to not be a number", val)
		})
	}
}

func TestParseNumberIntegers(t *testing.T) {
	testDefs := []struct {
		val   string
		radix int
		want  int64
	}{
		{"1.", 10, 1},
		{"+1.", 10, 1},
		{"-0.", 10, 0},
		{"-12.", 10, -12},
		// A trailing decimal point is always decimal
		{"10.", 16, 10},
		{"10.", 2, 10},
		{"99.", 8, 99},
		// Ratios equal to whole numbers are integers
		{"4/2", 10, 2},
		{"-4/2", 10, -2},
		{"0/5", 10, 0},
		{"1/1", 10, 1},
		{"110/11", 2, 2},
		{"Z", 36, 35},
		// With a base above 14 an exponent marker is just a digit
		{"1e5", 16, 485},
		{"1D", 16, 29},
		{"1e5", 36, 1805},
	}
	for _, testDef := range testDefs {
		got, err := types.ParseNumber(testDef.val, testDef.radix)
		assert.NoError(t, err, "ParseNumber(%q, %d)", testDef.val, testDef.radix)
		if !assert.NotNil(t, got, "ParseNumber(%q, %d) returned nil", testDef.val, testDef.radix) {
			continue
		}
		assert.True(t, got.Equal(types.NewNumberInteger(big.NewInt(testDef.want))),
			"ParseNumber(%q, %d) = %s, want integer %d", testDef.val, testDef.radix, got.String(), testDef.want)
	}
}

func TestParseNumberRatios(t *testing.T) {
	testDefs := []struct {
		val   string
		radix int
		want  *big.Rat
	}{
		{"1/2", 10, big.NewRat(1, 2)},
		{"+1/2", 10, big.NewRat(1, 2)},
		{"-1/2", 10, big.NewRat(-1, 2)},
		{"-4/6", 10, big.NewRat(-2, 3)},
		{"11/10", 2, big.NewRat(3, 2)},
		{"Z/Y", 36, big.NewRat(35, 34)},
	}
	for _, testDef := range testDefs {
		got, err := types.ParseNumber(testDef.val, testDef.radix)
		assert.NoError(t, err, "ParseNumber(%q, %d)", testDef.val, testDef.radix)
		if !assert.NotNil(t, got, "ParseNumber(%q, %d) returned nil", testDef.val, testDef.radix) {
			continue
		}
		assert.True(t, got.Equal(types.NewNumberRatio(testDef.want)),
			"ParseNumber(%q, %d) = %s, want ratio %s", testDef.val, testDef.radix, got.String(), testDef.want)
	}
}

func TestParseNumberFloats(t *testing.T) {
	// Values that are exact in binary, so rounding doesn't matter
	testDefs := []struct {
		val   string
		radix int
		want  float64
	}{
		{"1.5", 10, 1.5},
		{"+1.5", 10, 1.5},
		{"-1.5", 10, -1.5},
		{".5", 10, 0.5},
		{"+.5", 10, 0.5},
		{"-.5", 10, -0.5},
		{"0.25", 10, 0.25},
		{"1e5", 10, 100000},
		{"1E5", 10, 100000},
		{"1e+5", 10, 100000},
		{"1e-2", 10, 0.01},
		{"1.e5", 10, 100000},
		{"1.5e1", 10, 15},
		{".5e1", 10, 5},
		{"2.5E-1", 10, 0.25},
		// Every exponent marker, either case
		{"1s2", 10, 100},
		{"1f2", 10, 100},
		{"1d2", 10, 100},
		{"1l2", 10, 100},
		{"1S2", 10, 100},
		{"1F2", 10, 100},
		{"1D2", 10, 100},
		{"1L2", 10, 100},
		// Floats are always decimal
		{"1.5", 16, 1.5},
		{"1.5", 2, 1.5},
	}
	for _, testDef := range testDefs {
		got, err := types.ParseNumber(testDef.val, testDef.radix)
		assert.NoError(t, err, "ParseNumber(%q, %d)", testDef.val, testDef.radix)
		if !assert.NotNil(t, got, "ParseNumber(%q, %d) returned nil", testDef.val, testDef.radix) {
			continue
		}
		assert.Equal(t, types.NumberTypeFloat, got.NumType(), "ParseNumber(%q, %d) is not a float", testDef.val, testDef.radix)
		f, _ := got.Float().Float64()
		assert.InDelta(t, testDef.want, f, 1e-12, "ParseNumber(%q, %d) = %s", testDef.val, testDef.radix, got.String())
	}
}

func TestParseNumberNotANumber(t *testing.T) {
	testDefs := []struct {
		val   string
		radix int
	}{
		{"", 10},
		{"+", 10},
		{"-", 10},
		{".", 10},
		{"+.", 10},
		{"-.", 10},
		{"..", 10},
		{"1.2.3", 10},
		{"1..", 10},
		{"1/", 10},
		{"/2", 10},
		{"/", 10},
		{"1/2/3", 10},
		{"1/-2", 10},
		{"1/+2", 10},
		{"--1", 10},
		{"+-1", 10},
		{"1+", 10},
		{"1-", 10},
		{"1-2", 10},
		{"1+2", 10},
		// A ratio has no decimal point, and a float has no slash
		{"1/2.", 10},
		{"1./2", 10},
		{"1.5/2", 10},
		{"1/2e3", 10},
		{"1e3/2", 10},
		// Exponents need digits, and floats need digits around the point or an exponent
		{"e", 10},
		{"e5", 10},
		{".e5", 10},
		{"1e", 10},
		{"1e+", 10},
		{"1e-", 10},
		{"1.5e", 10},
		{"1.5ee5", 10},
		{"1.5e5.5", 10},
		{"1e5e5", 10},
		{"1.e", 10},
		// Other characters
		{"abc", 10},
		{"1a", 10},
		{"1.5x", 10},
		{" 1", 10},
		{"1 ", 10},
		{"\u0663", 10},
		// Digits out of range for the radix
		{"2", 2},
		{"1/2", 2},
		{"G", 16},
		{"1.5.", 16},
		// A trailing decimal point only applies to decimal digits
		{"FF.", 16},
		{"1F.", 16},
		{"-A.", 16},
		{"1/2.", 16},
		// Radix out of range
		{"1", 0},
		{"1", 1},
		{"1", 37},
		{"1", -1},
	}
	for _, testDef := range testDefs {
		assert.NotPanics(t, func() {
			got, err := types.ParseNumber(testDef.val, testDef.radix)
			assert.NoError(t, err, "ParseNumber(%q, %d)", testDef.val, testDef.radix)
			assert.Nil(t, got, "expected %q (radix %d) to not be a number, got %v", testDef.val, testDef.radix, got)
		})
	}
}

func TestParseNumberDivideByZero(t *testing.T) {
	for _, val := range []string{"1/0", "-1/0", "0/0", "+5/00"} {
		got, err := types.ParseNumber(val, 10)
		assert.ErrorIs(t, err, types.ErrDivideByZero, "ParseNumber(%q)", val)
		assert.Nil(t, got)
	}
	got, err := types.ParseNumber("1/0", 2)
	assert.ErrorIs(t, err, types.ErrDivideByZero)
	assert.Nil(t, got)
}

func TestParseNumberFloatRange(t *testing.T) {
	for _, val := range []string{
		"1e999999999",
		"-1e999999999",
		"1e2147483647",
		"1e2147483648",
		"1e99999999999999999999",
		"-1.5e99999999999999999999",
		// Too small to be anything but zero, which is not the same as zero
		"1e-2147483648",
		"1e-99999999999999999999",
		"0.5e-99999999999999999999",
	} {
		got, err := types.ParseNumber(val, 10)
		assert.ErrorIs(t, err, types.ErrFloatRange, "ParseNumber(%q)", val)
		assert.Nil(t, got, "ParseNumber(%q)", val)
	}

	// A zero mantissa is zero for any exponent
	for _, val := range []string{"0e0", "0.0e999999999", "0e99999999999999999999", "-0.0e-99999999999999999999", ".0e5"} {
		got, err := types.ParseNumber(val, 10)
		assert.NoError(t, err, "ParseNumber(%q)", val)
		if assert.NotNil(t, got, "ParseNumber(%q)", val) {
			assert.Equal(t, 0, got.Float().Sign(), "ParseNumber(%q)", val)
			assert.Equal(t, strings.HasPrefix(val, "-"), got.Float().Signbit(), "sign of ParseNumber(%q)", val)
		}
	}

	// Leading zeros in the exponent don't matter
	got, err := types.ParseNumber("1e"+strings.Repeat("0", 100000)+"5", 10)
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		f, _ := got.Float().Float64()
		assert.Equal(t, 100000.0, f)
	}
}

func TestParseNumberLongDigits(t *testing.T) {
	digits := strings.Repeat("9", 100000)

	got, err := types.ParseNumber(digits, 10)
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, types.NumberTypeInteger, got.NumType())
		assert.Len(t, got.Int().String(), 100000)
	}

	got, err = types.ParseNumber(digits+".5", 10)
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, types.NumberTypeFloat, got.NumType())
	}

	got, err = types.ParseNumber("1."+digits+"e5", 10)
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, types.NumberTypeFloat, got.NumType())
	}
}

// Floats are rounded once, to the nearest 53-bit value, so they must match what
// strconv produces for a float64.
func TestParseNumberFloatRounding(t *testing.T) {
	for _, val := range []string{"0.1", "0.2", "0.3", "1e-5", "6.02e23", "3.141592653589793", "2.2250738585072014e-308", "1.7976931348623157e308"} {
		got, err := types.ParseNumber(val, 10)
		assert.NoError(t, err, "ParseNumber(%q)", val)
		want, _ := strconv.ParseFloat(val, 64)
		if assert.NotNil(t, got, "ParseNumber(%q)", val) {
			f, _ := got.Float().Float64()
			assert.Equal(t, want, f, "ParseNumber(%q)", val)
		}
	}

	// Long fractions land near rounding boundaries often, which exposes
	// rounding twice
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 20000; n++ {
		var sb strings.Builder
		sb.WriteString("0.")
		for k := 17 + rng.Intn(10); k > 0; k-- {
			sb.WriteByte(byte('0' + rng.Intn(10)))
		}
		val := sb.String()
		got, err := types.ParseNumber(val, 10)
		if !assert.NoError(t, err, "ParseNumber(%q)", val) || !assert.NotNil(t, got, "ParseNumber(%q)", val) {
			continue
		}
		want, _ := strconv.ParseFloat(val, 64)
		f, _ := got.Float().Float64()
		if f != want {
			t.Errorf("ParseNumber(%q) = %v, want %v", val, f, want)
		}
	}
}

func TestParseRational(t *testing.T) {
	two := types.NewNumberInteger(big.NewInt(2))
	for _, tc := range []struct {
		val   string
		radix int
		want  *types.Number
	}{
		{"2", 10, two},
		{"2.", 10, two},
		{"4/2", 10, two},
		{"10", 2, two},
		{"1/2", 10, types.NewNumberRatio(big.NewRat(1, 2))},
		{"-1/2", 10, types.NewNumberRatio(big.NewRat(-1, 2))},
		// Floats are not rational
		{"1.5", 10, nil},
		{"1e5", 10, nil},
		{".5", 10, nil},
		{"1.e5", 10, nil},
		{"abc", 10, nil},
		{"1", 1, nil},
	} {
		got, err := types.ParseRational(tc.val, tc.radix)
		assert.NoError(t, err, "ParseRational(%q, %d)", tc.val, tc.radix)
		if tc.want == nil {
			assert.Nil(t, got, "ParseRational(%q, %d)", tc.val, tc.radix)
			continue
		}
		if assert.NotNil(t, got, "ParseRational(%q, %d)", tc.val, tc.radix) {
			assert.True(t, got.Equal(tc.want), "ParseRational(%q, %d) = %s", tc.val, tc.radix, got)
		}
	}

	got, err := types.ParseRational("1/0", 10)
	assert.ErrorIs(t, err, types.ErrDivideByZero)
	assert.Nil(t, got)
}
