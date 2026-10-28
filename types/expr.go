package types

import (
	"fmt"
	"math/big"
	"slices"
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

func NewLiteral(item Expression) *Literal {
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
