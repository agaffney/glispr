package types

import (
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
	ExpressionTypeNull
	ExpressionTypeCons
	ExpressionTypeString
	ExpressionTypeNumber
	ExpressionTypeSymbol
	ExpressionTypeCharacter
	ExpressionTypeVector
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

func (c *Cons) String() string { return Format(c, nil) }

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

type String struct {
	val string
}

func NewString(val string) *String {
	return &String{
		val: val,
	}
}

func (String) Type() ExpressionType { return ExpressionTypeString }

// stringEscaper writes a \ before the two characters that mean something inside
// a string in reader syntax.
var stringEscaper = strings.NewReplacer(`"`, `\"`, `\`, `\\`)

// String returns the string in reader syntax.
func (s String) String() string {
	return `"` + stringEscaper.Replace(s.val) + `"`
}

func (s String) Value() string { return s.val }

type NumberType int

const (
	NumberTypeNone NumberType = iota
	NumberTypeInteger
	NumberTypeFloat
	NumberTypeRatio
)

// FloatFormat is the format of a float: single or double precision.
type FloatFormat int

const (
	FloatFormatNone FloatFormat = iota
	FloatFormatSingle
	FloatFormatDouble
)

// Precision returns the size of the mantissa in bits, or 0 for an unknown format.
func (f FloatFormat) Precision() uint {
	switch f {
	case FloatFormatSingle:
		return 24
	case FloatFormatDouble:
		return 53
	}
	return 0
}

func (f FloatFormat) String() string {
	switch f {
	case FloatFormatSingle:
		return "single-float"
	case FloatFormatDouble:
		return "double-float"
	}
	return "<unknown>"
}

type Number struct {
	typ      NumberType
	intVal   *big.Int
	floatVal *big.Float
	format   FloatFormat
	ratVal   *big.Rat
}

func NewNumberInteger(intVal *big.Int) *Number {
	return &Number{
		typ:    NumberTypeInteger,
		intVal: new(big.Int).Set(intVal),
	}
}

// NewNumberFloat returns a float in the given format. The value is rounded to
// the precision of the format. It panics if format is not single or double.
//
// The value is stored with the precision of the format even when it is a
// denormal, which has fewer significant bits. That doesn't change the value, but
// any arithmetic on floats should build its results with this function so that
// the format and precision stay consistent.
func NewNumberFloat(floatVal *big.Float, format FloatFormat) *Number {
	if format.Precision() == 0 {
		panic("types: NewNumberFloat called with unknown float format")
	}
	return &Number{
		typ:      NumberTypeFloat,
		floatVal: new(big.Float).SetPrec(format.Precision()).Set(floatVal),
		format:   format,
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

// Int returns a copy of the value of an integer, or nil for other numbers.
func (n Number) Int() *big.Int {
	if n.intVal == nil {
		return nil
	}
	return new(big.Int).Set(n.intVal)
}

// Float returns a copy of the value of a float, or nil for other numbers. The
// copy keeps the precision of the format.
func (n Number) Float() *big.Float {
	if n.floatVal == nil {
		return nil
	}
	return new(big.Float).Copy(n.floatVal)
}

// Format returns the format of a float, or FloatFormatNone for other numbers.
func (n Number) Format() FloatFormat { return n.format }

// Rat returns a copy of the value of a ratio, or nil for other numbers.
func (n Number) Rat() *big.Rat {
	if n.ratVal == nil {
		return nil
	}
	return new(big.Rat).Set(n.ratVal)
}

// Equal reports whether n and other are the same number, like eql in Common
// Lisp: they must have the same type, which for floats includes the format, and
// the same value. So 1.0f0 and 1.0d0 are not equal here, and neither are 1 and
// 1.0. It is not the numeric comparison done by =.
func (n Number) Equal(other *Number) bool {
	if other.typ != n.typ {
		return false
	}
	switch n.typ {
	case NumberTypeInteger:
		return n.intVal.Cmp(other.intVal) == 0
	case NumberTypeFloat:
		return n.format == other.format && n.floatVal.Cmp(other.floatVal) == 0
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
		return n.floatString()
	case NumberTypeRatio:
		return n.ratVal.String()
	}
	return "<unknown>"
}

// floatString formats a float with the fewest digits that read back as the same
// value, and always with an exponent marker for its format: f for single and d
// for double. Reading it back gives an equal number whatever the default format
// is.
func (n Number) floatString() string {
	f, _ := n.floatVal.Float64()
	bits, marker := 64, "d"
	if n.format == FloatFormatSingle {
		bits, marker = 32, "f"
	}
	text := strconv.FormatFloat(f, 'g', -1, bits)
	mantissa, exp, hasExp := strings.Cut(text, "e")
	if !strings.Contains(mantissa, ".") {
		mantissa += ".0"
	}
	if !hasExp {
		return mantissa + marker + "0"
	}
	// Drop the sign and leading zeros that strconv adds to a positive exponent
	expNum, _ := strconv.Atoi(exp)
	return mantissa + marker + strconv.Itoa(expNum)
}

type Symbol struct {
	name string
	// pkg is the home package, which is nil for an uninterned symbol
	pkg *Package
}

// NewSymbol returns a new symbol that isn't interned in any package, like
// make-symbol. To get the one symbol that a package has for a name, use
// Package.Intern.
func NewSymbol(name string) *Symbol {
	return &Symbol{
		name: name,
	}
}

func (Symbol) Type() ExpressionType { return ExpressionTypeSymbol }

// String returns the symbol in reader syntax, qualified with its package. Use
// Format to print it relative to a current package.
func (s Symbol) String() string { return Format(&s, nil) }

func (s Symbol) Name() string {
	return s.name
}

// Package returns the home package of the symbol, or nil if it is uninterned.
func (s Symbol) Package() *Package {
	return s.pkg
}
