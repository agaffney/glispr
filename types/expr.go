package types

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
)

type Expression interface {
	Type() ExpressionType
	String() string
}

type ExpressionType int

const (
	ExpressionTypeNone ExpressionType = iota
	ExpressionTypeNull
	ExpressionTypeCons
	ExpressionTypeLiteral
	ExpressionTypeString
	ExpressionTypeNumber
	ExpressionTypeSymbol
)

// null is the type of the empty list. It is unexported so that Nil is the only
// instance and external code can't construct a second "empty list" that would
// be mistaken for Nil. Compare against Nil or check Type() to detect it.
type null struct{}

// Nil is the empty list. It must be treated as read-only: reassigning it would
// break every comparison against it, including ListItems.
var Nil Expression = &null{}

func (*null) Type() ExpressionType { return ExpressionTypeNull }

func (*null) String() string { return "()" }

type Cons struct {
	car Expression
	cdr Expression
}

// isNilCons reports whether expr holds a nil *Cons, which is not == nil when
// stored in an Expression.
func isNilCons(expr Expression) bool {
	c, ok := expr.(*Cons)
	return ok && c == nil
}

// NewCons builds a cons cell. It panics if car or cdr is nil, including a nil
// *Cons stored in an Expression.
func NewCons(car, cdr Expression) *Cons {
	if car == nil || cdr == nil || isNilCons(car) || isNilCons(cdr) {
		panic("types: NewCons called with nil car or cdr")
	}
	return &Cons{
		car: car,
		cdr: cdr,
	}
}

func (*Cons) Type() ExpressionType { return ExpressionTypeCons }

// Car returns the first element of the cell. It panics on a nil *Cons.
func (c *Cons) Car() Expression {
	if c == nil {
		panic("types: Car called on nil *Cons")
	}
	return c.car
}

// Cdr returns the rest of the cell. It panics on a nil *Cons.
func (c *Cons) Cdr() Expression {
	if c == nil {
		panic("types: Cdr called on nil *Cons")
	}
	return c.cdr
}

func (c *Cons) String() string {
	if c == nil {
		return "<nil>"
	}
	var sb strings.Builder
	sb.WriteString(`(`)
	sb.WriteString(c.car.String())
	rest := c.cdr
	for {
		next, ok := rest.(*Cons)
		if !ok {
			break
		}
		sb.WriteString(` `)
		sb.WriteString(next.car.String())
		rest = next.cdr
	}
	if rest != Nil {
		sb.WriteString(` . `)
		sb.WriteString(rest.String())
	}
	sb.WriteString(`)`)
	return sb.String()
}

// NewList builds a proper list from items. It returns Nil if there are none.
// The returned list does not share storage with items. It panics if any item
// is nil, as NewCons does.
func NewList(items []Expression) Expression {
	return NewListTail(items, Nil)
}

// NewListTail builds a list of items whose final cdr is tail. The result is
// dotted unless tail is Nil or itself a proper list, in which case the result
// is a proper list, so (a . (b c)) is (a b c). With no items it returns tail
// itself. It panics if tail or any item is nil, as NewCons does.
func NewListTail(items []Expression, tail Expression) Expression {
	if tail == nil || isNilCons(tail) {
		panic("types: NewListTail called with nil tail")
	}
	ret := tail
	for _, item := range slices.Backward(items) {
		ret = NewCons(item, ret)
	}
	return ret
}

// ListItems returns the elements of a proper list, or an empty non-nil slice
// for Nil. The bool is false (and the slice nil) if expr is not a proper list,
// meaning it is neither Nil nor a chain of conses ending in Nil.
func ListItems(expr Expression) ([]Expression, bool) {
	items := []Expression{}
	for {
		// Compare against Nil rather than a concrete type so that Nil is the
		// only terminator, however it is represented
		if expr == Nil {
			return items, true
		}
		switch e := expr.(type) {
		case *Cons:
			if e == nil {
				return nil, false
			}
			items = append(items, e.car)
			expr = e.cdr
		default:
			return nil, false
		}
	}
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
