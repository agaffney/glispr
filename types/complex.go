package types

import (
	"math"
	"math/big"
	"strings"
)

// NewNumberComplex returns the complex number with the given real and imaginary
// parts, which must be real numbers: integers, ratios or floats. It panics if
// either is nil or a complex number.
//
// If a part is a float, both are made floats, in the larger of the formats of
// the parts: the other part is converted if it is a rational or a smaller float.
// The error is ErrFloatRange if that conversion can't be represented.
//
// A complex number with rational parts and an imaginary part of zero is the
// real part, which is returned as it is. With float parts it stays complex, so
// the result of NewNumberComplex(1.0, 0.0) is a complex number.
func NewNumberComplex(re, im *Number) (*Number, error) {
	if re == nil || im == nil || re.typ == NumberTypeComplex || im.typ == NumberTypeComplex {
		panic("types: NewNumberComplex called with a part that is nil or complex")
	}
	if re.typ == NumberTypeFloat || im.typ == NumberTypeFloat {
		format := FloatFormatSingle
		if re.format == FloatFormatDouble || im.format == FloatFormatDouble {
			format = FloatFormatDouble
		}
		var err error
		if re, err = toFloat(re, format); err != nil {
			return nil, err
		}
		if im, err = toFloat(im, format); err != nil {
			return nil, err
		}
	} else if im.typ == NumberTypeInteger && im.intVal.Sign() == 0 {
		return re, nil
	}
	return &Number{
		typ: NumberTypeComplex,
		re:  re,
		im:  im,
	}, nil
}

// toFloat returns n as a float in format, which is at least as large as the
// format n has if it is a float.
func toFloat(n *Number, format FloatFormat) (*Number, error) {
	if n.typ == NumberTypeFloat {
		// A larger format holds every value of a smaller one
		return NewNumberFloat(n.floatVal, format), nil
	}
	rat := n.ratVal
	if n.typ == NumberTypeInteger {
		rat = new(big.Rat).SetInt(n.intVal)
	}
	var f float64
	if format == FloatFormatSingle {
		f32, _ := rat.Float32()
		f = float64(f32)
	} else {
		f, _ = rat.Float64()
	}
	// Too large to represent, or a value that isn't zero but rounds to it
	if math.IsInf(f, 0) || (f == 0 && rat.Sign() != 0) {
		return nil, ErrFloatRange
	}
	return NewNumberFloat(new(big.Float).SetFloat64(f), format), nil
}

// writeComplex writes a complex number in reader syntax, with writePart writing
// each of the parts, so the printer can write them for its read base.
func writeComplex(sb *strings.Builder, re, im *Number, writePart func(*Number)) {
	sb.WriteString("#C(")
	writePart(re)
	sb.WriteString(" ")
	writePart(im)
	sb.WriteString(")")
}

// Real returns the real part of a complex number, or nil for other numbers.
func (n Number) Real() *Number { return n.re }

// Imag returns the imaginary part of a complex number, or nil for other
// numbers.
func (n Number) Imag() *Number { return n.im }
