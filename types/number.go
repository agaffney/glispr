package types

import (
	"cmp"
	"errors"
	"math/big"
	"strings"
)

// ErrDivideByZero is returned when a ratio has a zero denominator.
var ErrDivideByZero = errors.New("division by zero")

// ErrFloatRange is returned when a float is too large or too small to represent.
var ErrFloatRange = errors.New("floating point number out of range")

// NewNumberFromString parses val as a number in the given radix. It returns
// nil if val is not a number.
func NewNumberFromString(val string, radix int) *Number {
	num, _ := ParseNumber(val, radix)
	return num
}

// ParseNumber parses val as a number following the syntax of CLHS 2.3.1.1, with
// integers and ratios in the given radix (2 to 36). It returns nil without an
// error if val is not a number, including when radix is out of range. An error
// is returned for input that has number syntax but cannot be represented, such
// as a ratio with a zero denominator (ErrDivideByZero) or a float whose
// magnitude is too large or too small (ErrFloatRange).
//
//   - An integer is an optional sign and digits in radix, or an optional sign and
//     decimal digits followed by a decimal point, which are always decimal.
//   - A ratio is an optional sign, digits in radix, a slash, and digits in radix.
//     It has no trailing decimal point, and a ratio equal to a whole number, such
//     as 4/2, is an integer.
//   - A float is always decimal. It is either digits with a decimal point and
//     digits after it, or digits with an optional decimal point and fraction and
//     an exponent. The exponent is a marker (e, s, f, d, or l) with an optional
//     sign and digits.
func ParseNumber(val string, radix int) (*Number, error) {
	if num, err := ParseRational(val, radix); num != nil || err != nil {
		return num, err
	}
	sign, rest, ok := splitSign(val)
	if !ok || radix < 2 || radix > 36 {
		return nil, nil
	}
	return parseFloat(sign, rest)
}

// ParseRational is like ParseNumber, but only accepts integers and ratios.
func ParseRational(val string, radix int) (*Number, error) {
	if radix < 2 || radix > 36 {
		return nil, nil
	}
	sign, rest, ok := splitSign(val)
	if !ok {
		return nil, nil
	}

	// Integer in the current radix
	if isDigits(rest, radix) {
		return parseInteger(sign+rest, radix), nil
	}

	// Ratio in the current radix
	if numer, denom, ok := strings.Cut(rest, "/"); ok {
		if !isDigits(numer, radix) || !isDigits(denom, radix) {
			return nil, nil
		}
		return parseRatio(sign+numer, denom, radix)
	}

	// Integer with a trailing decimal point, which is decimal in any radix
	if digits, ok := strings.CutSuffix(rest, "."); ok && isDigits(digits, 10) {
		return parseInteger(sign+digits, 10), nil
	}

	return nil, nil
}

// splitSign splits an optional leading sign from val. The sign is "-" or "",
// and ok is false if nothing follows it.
func splitSign(val string) (sign, rest string, ok bool) {
	rest = val
	if rest != "" && (rest[0] == '+' || rest[0] == '-') {
		if rest[0] == '-' {
			sign = "-"
		}
		rest = rest[1:]
	}
	return sign, rest, rest != ""
}

// digitValue returns the value of ch as a digit, or 36 if it is not one.
func digitValue(ch byte) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'z':
		return int(ch-'a') + 10
	case ch >= 'A' && ch <= 'Z':
		return int(ch-'A') + 10
	}
	return 36
}

// isDigits reports whether s is one or more ASCII digits valid in radix.
func isDigits(s string, radix int) bool {
	if s == "" {
		return false
	}
	for k := 0; k < len(s); k++ {
		if digitValue(s[k]) >= radix {
			return false
		}
	}
	return true
}

func parseInteger(s string, radix int) *Number {
	n, _ := new(big.Int).SetString(s, radix)
	return NewNumberInteger(n)
}

func parseRatio(numer, denom string, radix int) (*Number, error) {
	n, _ := new(big.Int).SetString(numer, radix)
	d, _ := new(big.Int).SetString(denom, radix)
	if d.Sign() == 0 {
		return nil, ErrDivideByZero
	}
	rat := new(big.Rat).SetFrac(n, d)
	if rat.IsInt() {
		return NewNumberInteger(rat.Num()), nil
	}
	return NewNumberRatio(rat), nil
}

// parseFloat parses the unsigned decimal float syntax in rest, or returns nil
// if rest is not a float.
func parseFloat(sign, rest string) (*Number, error) {
	// Scan decimal digits, starting at pos
	scan := func(pos int) int {
		for pos < len(rest) && rest[pos] >= '0' && rest[pos] <= '9' {
			pos++
		}
		return pos
	}
	pos := scan(0)
	intPart := rest[:pos]
	hasDot := false
	fracPart := ""
	if pos < len(rest) && rest[pos] == '.' {
		hasDot = true
		start := pos + 1
		pos = scan(start)
		fracPart = rest[start:pos]
	}
	hasExp := false
	expPart := ""
	if pos < len(rest) && strings.ContainsRune("esfdlESFDL", rune(rest[pos])) {
		hasExp = true
		pos++
		start := pos
		if pos < len(rest) && (rest[pos] == '+' || rest[pos] == '-') {
			pos++
		}
		digitsStart := pos
		pos = scan(pos)
		if pos == digitsStart {
			return nil, nil
		}
		expPart = rest[start:pos]
	}
	if pos != len(rest) {
		return nil, nil
	}
	// Either digits, a dot, and digits after the dot, or digits with an exponent
	if (!hasDot || fracPart == "") && (intPart == "" || !hasExp) {
		return nil, nil
	}
	// TODO: use the exponent marker to pick the float format
	text := sign + cmp.Or(intPart, "0") + "." + cmp.Or(fracPart, "0")
	if hasExp {
		text += "e" + expPart
	}
	// Zero is zero no matter how large the exponent is
	zeroMantissa := strings.Trim(intPart+fracPart, "0") == ""
	if zeroMantissa {
		f := new(big.Float).SetPrec(53)
		if sign == "-" {
			f.Neg(f)
		}
		return NewNumberFloat(f), nil
	}
	// Set the precision first so the value is rounded once, to the nearest
	// float with this precision. The syntax is valid by now, so the only way for
	// this to fail is an exponent too large for big.Float.
	f, ok := new(big.Float).SetPrec(53).SetString(text) // This precision is the same you get with big.NewFloat(n)
	if !ok || f.IsInf() {
		return nil, ErrFloatRange
	}
	// A nonzero number that rounds to zero is too small to represent
	if f.Sign() == 0 {
		return nil, ErrFloatRange
	}
	return NewNumberFloat(f), nil
}
