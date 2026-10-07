package reader

import (
	"fmt"
	"math/big"
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

func (List) Type() ExpressionType { return ExpressionTypeList }

func (l List) String() string {
	// TODO
	return ""
}

type Literal struct {
	item Expression
}

func (Literal) Type() ExpressionType { return ExpressionTypeLiteral }

func (l Literal) String() string {
	return fmt.Sprintf("'%s", l.item.String())
}

type String struct {
	val string
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
}

func (Number) Type() ExpressionType { return ExpressionTypeNumber }

func (n Number) String() string {
	// TODO
	return ""
}

type Symbol struct {
	name string
}

func (Symbol) Type() ExpressionType { return ExpressionTypeSymbol }

func (s Symbol) String() string {
	// TODO
	return fmt.Sprintf(`|%s|`, s.name)
}

func (s Symbol) Name() string {
	return s.name
}
