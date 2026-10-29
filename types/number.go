package types

import (
	"cmp"
	"errors"
	"math/big"
	"strconv"
	"strings"
)

// ErrDivideByZero is returned when a ratio has a zero denominator.
var ErrDivideByZero = errors.New("division by zero")

// ErrFloatRange is returned when a float is too large or too small to represent.
var ErrFloatRange = errors.New("floating point number out of range")

// ErrUnknownFloatFormat is returned when a float needs the default format and
// that is not single or double.
var ErrUnknownFloatFormat = errors.New("unknown default float format")

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
//     an exponent. The exponent is a marker with an optional sign and digits.
//     The marker picks the format: s and f are single, d and l are double, and e
//     uses defaultFormat, as does a float with no exponent. A float that needs
//     defaultFormat when it is not single or double is an error
//     (ErrUnknownFloatFormat).
func ParseNumber(val string, radix int, defaultFormat FloatFormat) (*Number, error) {
	if num, err := ParseRational(val, radix); num != nil || err != nil {
		return num, err
	}
	sign, rest, ok := splitSign(val)
	if !ok || radix < 2 || radix > 36 {
		return nil, nil
	}
	return parseFloat(sign, rest, defaultFormat)
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
func parseFloat(sign, rest string, defaultFormat FloatFormat) (*Number, error) {
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
	format := defaultFormat
	if pos < len(rest) && strings.ContainsRune("esfdlESFDL", rune(rest[pos])) {
		hasExp = true
		switch rest[pos] {
		case 's', 'S', 'f', 'F':
			format = FloatFormatSingle
		case 'd', 'D', 'l', 'L':
			format = FloatFormatDouble
		}
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
	text := sign + cmp.Or(intPart, "0") + "." + cmp.Or(fracPart, "0")
	if hasExp {
		text += "e" + expPart
	}
	if format.Precision() == 0 {
		return nil, ErrUnknownFloatFormat
	}
	bits := 64
	if format == FloatFormatSingle {
		bits = 32
	}
	// strconv rounds once, to the nearest float of this size
	f, err := strconv.ParseFloat(text, bits)
	if err != nil {
		// The syntax is valid by now, so this is a magnitude too large
		return nil, ErrFloatRange
	}
	// A nonzero number that rounds to zero is too small to represent
	if f == 0 && strings.Trim(intPart+fracPart, "0") != "" {
		return nil, ErrFloatRange
	}
	return NewNumberFloat(new(big.Float).SetFloat64(f), format), nil
}
