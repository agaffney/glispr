package types

import "strings"

// BitVector is a one-dimensional array of bits.
type BitVector struct {
	bits []byte
}

// NewBitVector returns a bit vector of bits, each of which is 0 or 1. It panics
// if any of them is not. The vector does not share storage with bits.
func NewBitVector(bits []byte) *BitVector {
	for _, bit := range bits {
		if bit > 1 {
			panic("types: NewBitVector called with a bit that is not 0 or 1")
		}
	}
	// Never nil, so that equal vectors have equal contents however they were made
	return &BitVector{
		bits: append(make([]byte, 0, len(bits)), bits...),
	}
}

func (BitVector) Type() ExpressionType { return ExpressionTypeBitVector }

// Len returns the number of bits in the vector.
func (v BitVector) Len() int { return len(v.bits) }

// At returns the bit at index i, which is 0 or 1. It panics if i is out of
// range.
func (v BitVector) At(i int) byte { return v.bits[i] }

// Bits returns a copy of the bits in the vector.
func (v BitVector) Bits() []byte { return append([]byte{}, v.bits...) }

// String returns the vector in reader syntax: #* and its bits.
func (v BitVector) String() string {
	var sb strings.Builder
	sb.WriteString("#*")
	for _, bit := range v.bits {
		sb.WriteByte('0' + bit)
	}
	return sb.String()
}
