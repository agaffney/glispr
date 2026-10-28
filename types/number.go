package types

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
)

// ErrDivideByZero is returned when a ratio has a zero denominator.
var ErrDivideByZero = errors.New("division by zero")

// NewNumberFromString parses val as a number in the given radix. It returns
// nil if val is not a number.
func NewNumberFromString(val string, radix int) *Number {
	num, _ := ParseNumber(val, radix)
	return num
}

// ParseNumber returns nil without an error if val is not a number. An error is
// returned for input that has number syntax but cannot be represented, such as
// a ratio with a zero denominator.
func ParseNumber(val string, radix int) (*Number, error) {
	foundRatio := false
	foundFloat := false
	buf := bytes.NewBuffer(nil)
	for i, ch := range val {
		switch {
		case ch >= '0' && ch <= min('0'+rune(radix-1), '9'):
			_, _ = buf.WriteRune(ch)
		case radix > 10 && ((ch >= 'a' && ch < 'a'+rune(radix-10)) || (ch >= 'A' && ch < 'A'+rune(radix-10))):
			// Write any alphabetic character that would be valid
			// for the current radix to the buffer
			_, _ = buf.WriteRune(ch)
		case ch == '-' || ch == '+':
			if foundRatio {
				// No sign allowed on denominator
				return nil, nil
			}
			_, _ = buf.WriteRune(ch)
		case ch == '.':
			if radix != 10 && i != len(val)-1 {
				return nil, nil
			}
			_, _ = buf.WriteRune(ch)
			if i != len(val)-1 {
				foundFloat = true
			}
		case radix == 10 && (ch == 'e' || ch == 'E' || ch == 's' || ch == 'S' || ch == 'f' || ch == 'F' || ch == 'd' || ch == 'D' || ch == 'l' || ch == 'L'):
			// Replace various exponent/size markers with 'e'
			// TODO: record/use size markers
			_, _ = buf.WriteRune('e')
			foundFloat = true
		case ch == '/':
			_, _ = buf.WriteRune(ch)
			foundRatio = true
		default:
			// Non-number character
			return nil, nil
		}
	}
	bufStr := buf.String()
	// Treat trailing decimal point as integer
	if strings.HasSuffix(bufStr, `.`) {
		foundFloat = false
		bufStr = strings.TrimSuffix(bufStr, `.`)
	}
	if foundFloat {
		// Parse number using big.Float
		tmpFloat := new(big.Float)
		if _, ok := tmpFloat.SetString(bufStr); !ok {
			return nil, nil
		}
		return NewNumberFloat(tmpFloat.SetPrec(53)), nil // This precision is the same you get with big.NewFloat(n)
	} else if foundRatio {
		// Parse number as ratio
		parts := strings.Split(bufStr, `/`)
		if len(parts) != 2 {
			return nil, nil
		}
		tmpNum := new(big.Int)
		if _, ok := tmpNum.SetString(parts[0], radix); !ok {
			return nil, nil
		}
		tmpDenom := new(big.Int)
		if _, ok := tmpDenom.SetString(parts[1], radix); !ok {
			return nil, nil
		}
		if tmpDenom.Sign() == 0 {
			return nil, ErrDivideByZero
		}
		tmpRat := new(big.Rat).SetFrac(tmpNum, tmpDenom)
		return NewNumberRatio(tmpRat), nil
	} else {
		// Parse number as integer
		tmpInt := new(big.Int)
		if _, ok := tmpInt.SetString(bufStr, radix); !ok {
			return nil, nil
		}
		return NewNumberInteger(tmpInt), nil
	}
}
