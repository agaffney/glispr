package types

import "fmt"

// Placeholder stands in for a form that is still being read, so that the form
// can contain a reference to itself, as in #1=(a . #1#). The reader replaces
// every placeholder with the form once it is complete, so a placeholder is never
// part of an expression that it returns.
type Placeholder struct {
	label int
}

// NewPlaceholder returns a placeholder for the form labeled with the number
// label.
func NewPlaceholder(label int) *Placeholder {
	return &Placeholder{
		label: label,
	}
}

func (*Placeholder) Type() ExpressionType { return ExpressionTypePlaceholder }

// String returns the reference to the label, such as #1#.
func (p *Placeholder) String() string { return fmt.Sprintf("#%d#", p.label) }

// Substitute replaces every reference to ph in root, which is the car or cdr of
// a cons or an item of a vector in it, with value. It goes through the whole of
// root, including cells that are reached more than once or in a cycle, once
// each.
//
// This changes conses and vectors that are already built, which nothing else in
// this package does, so it must only be used on structure that no one else has
// seen yet, as the reader does while it builds a labeled form. It panics if ph
// or value is nil.
func Substitute(root Expression, ph *Placeholder, value Expression) {
	if ph == nil || value == nil || isNilCons(value) {
		panic("types: Substitute called with a nil placeholder or value")
	}
	seen := map[Expression]bool{}
	stack := []Expression{root}
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch v := e.(type) {
		case *Cons:
			if v == nil || seen[v] {
				continue
			}
			seen[v] = true
			stack = substituteIn(stack, &v.car, ph, value)
			stack = substituteIn(stack, &v.cdr, ph, value)
		case *Vector:
			if seen[v] {
				continue
			}
			seen[v] = true
			for k := range v.items {
				stack = substituteIn(stack, &v.items[k], ph, value)
			}
		}
	}
}

// substituteIn replaces *slot with value if it is ph, and otherwise adds it to
// stack to be searched.
func substituteIn(stack []Expression, slot *Expression, ph *Placeholder, value Expression) []Expression {
	if *slot == Expression(ph) {
		*slot = value
		return stack
	}
	return append(stack, *slot)
}
