package reader

import (
	"bytes"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

type Expression interface {
	Type() ExpressionType
	String() string
}

type ExpressionType int

const (
	ExpressionTypeNone ExpressionType = iota
	ExpressionTypeList
	ExpressionTypeLiteral
	ExpressionTypeString
	ExpressionTypeNumber
	ExpressionTypeSymbol
)

type List struct {
	items []Expression
}

func NewList(items []Expression) *List {
	return &List{
		items: slices.Clone(items),
	}
}

func (List) Type() ExpressionType { return ExpressionTypeList }

func (l List) String() string {
	ret := `(`
	for i, item := range l.items {
		ret += item.String()
		if i < len(l.items)-1 {
			ret += ` `
		}
	}
	ret += `)`
	return ret
}

func (l List) Items() []Expression {
	return slices.Clone(l.items)
}

type Literal struct {
	item Expression
}

func NewListeral(item Expression) *Literal {
	return &Literal{
		item: item,
	}
}

func (Literal) Type() ExpressionType { return ExpressionTypeLiteral }

func (l Literal) String() string {
	return fmt.Sprintf("'%s", l.item.String())
}

func (l Literal) Item() Expression {
	return l.item
}

type String struct {
	val string
}

func NewString(val string) *String {
	return &String{
		val: val,
	}
}

func (String) Type() ExpressionType { return ExpressionTypeString }

func (s String) String() string {
	return fmt.Sprintf(`"%s"`, s.val)
}

func (s String) Value() string { return s.val }

type NumberType int

const (
	NumberTypeNone NumberType = iota
	NumberTypeInteger
	NumberTypeFloat
	NumberTypeRatio
)

type Number struct {
	typ      NumberType
	intVal   *big.Int
	floatVal *big.Float
	ratVal   *big.Rat
}

func NewNumber(typ NumberType, intVal *big.Int, floatVal *big.Float, ratVal *big.Rat) *Number {
	return &Number{
		typ:      typ,
		intVal:   intVal,
		floatVal: floatVal,
		ratVal:   ratVal,
	}
}

func NewNumberInteger(intVal *big.Int) *Number {
	return &Number{
		typ:    NumberTypeInteger,
		intVal: new(big.Int).Set(intVal),
	}
}

func NewNumberFloat(floatVal *big.Float) *Number {
	return &Number{
		typ:      NumberTypeFloat,
		floatVal: new(big.Float).Set(floatVal),
	}
}

func NewNumberRatio(ratVal *big.Rat) *Number {
	return &Number{
		typ:    NumberTypeRatio,
		ratVal: new(big.Rat).Set(ratVal),
	}
}

func NewNumberFromString(val string) *Number {
	var err error
	radix := 10
	foundNum := false
	foundRatio := false
	foundFloat := false
	foundRadix := false
	buf := bytes.NewBuffer(nil)
	for i, ch := range val {
		if i == 0 {
			if ch == '#' {
				foundRadix = true
				continue
			}
			foundNum = true
		}
		if foundNum {
			switch {
			case ch >= '0' && ch <= '9':
				_, _ = buf.WriteRune(ch)
			case radix > 10 && (ch >= 'a' && ch < 'a'+rune(radix-10)) || (ch >= 'A' && ch < 'A'+rune(radix-10)):
				// Write any alphabetic character that would be valid
				// for the current radix to the buffer
				_, _ = buf.WriteRune(ch)
			case ch == '-' || ch == '+':
				if foundRatio {
					// No sign allowed on denominator
					return nil
				}
				_, _ = buf.WriteRune(ch)
			case ch == '.':
				_, _ = buf.WriteRune(ch)
				foundFloat = true
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
				return nil
			}
		} else if foundRadix {
			if i == 1 && (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				switch ch {
				case 'b', 'B':
					radix = 2
					foundNum = true
					continue
				case 'o', 'O':
					radix = 8
					foundNum = true
					continue
				case 'x', 'X':
					radix = 16
					foundNum = true
					continue
				}
			}
			switch {
			case ch >= '0' && ch <= '9':
				_, _ = buf.WriteRune(ch)
			case ch == 'r' || ch == 'R':
				radix, err = strconv.Atoi(buf.String())
				if err != nil {
					return nil
				}
				buf.Reset()
				foundNum = true
			default:
				// This isn't a valid number, so return
				return nil
			}
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
			return nil
		}
		return &Number{
			typ:      NumberTypeFloat,
			floatVal: tmpFloat.SetPrec(53), // This precision is the same you get with big.NewFloat(n)
		}
	} else if foundRatio {
		// Parse number as ratio
		parts := strings.Split(bufStr, `/`)
		if len(parts) != 2 {
			return nil
		}
		tmpNum := new(big.Int)
		if _, ok := tmpNum.SetString(parts[0], radix); !ok {
			return nil
		}
		tmpDenom := new(big.Int)
		if _, ok := tmpDenom.SetString(parts[1], radix); !ok {
			return nil
		}
		tmpRat := new(big.Rat).SetFrac(tmpNum, tmpDenom)
		return &Number{
			typ:    NumberTypeRatio,
			ratVal: tmpRat,
		}
	} else {
		// Parse number as integer
		tmpInt := new(big.Int)
		if _, ok := tmpInt.SetString(bufStr, radix); !ok {
			return nil
		}
		return &Number{
			typ:    NumberTypeInteger,
			intVal: tmpInt,
		}
	}
}

func (Number) Type() ExpressionType { return ExpressionTypeNumber }

func (n Number) NumType() NumberType { return n.typ }

func (n Number) Int() *big.Int { return n.intVal }

func (n Number) Float() *big.Float { return n.floatVal }

func (n Number) Rat() *big.Rat { return n.ratVal }

func (n Number) Equal(other *Number) bool {
	if other.typ != n.typ {
		return false
	}
	switch n.typ {
	case NumberTypeInteger:
		return n.intVal.Cmp(other.intVal) == 0
	case NumberTypeFloat:
		return n.floatVal.Cmp(other.floatVal) == 0
	case NumberTypeRatio:
		return n.ratVal.Cmp(other.ratVal) == 0
	}
	return false
}

func (n Number) String() string {
	switch n.typ {
	case NumberTypeInteger:
		return n.intVal.String()
	case NumberTypeFloat:
		return n.floatVal.String()
	case NumberTypeRatio:
		return n.ratVal.String()
	}
	return "<unknown>"
}

type Symbol struct {
	name string
}

func NewSymbol(name string) *Symbol {
	return &Symbol{
		name: name,
	}
}

func (Symbol) Type() ExpressionType { return ExpressionTypeSymbol }

func (s Symbol) String() string {
	// TODO
	return fmt.Sprintf(`|%s|`, s.name)
}

func (s Symbol) Name() string {
	return s.name
}
