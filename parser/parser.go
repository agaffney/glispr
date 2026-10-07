package parser

import (
	"fmt"
	"io"
)

type Expression interface {
	Type() ExpressionType
	String() string
}

type ExpressionType int

const (
	ExpressionTypeNone ExpressionType = iota
	ExpressionTypeList
	ExpressionTypeListQuote
	ExpressionTypeString
	ExpressionTypeNumber
	ExpressionTypeSymbol
)

type Parser struct {
	l *Lexer
}

func NewParser(r io.Reader) (*Parser, error) {
	l, err := NewLexer(r)
	if err != nil {
		return nil, fmt.Errorf("create lexer: %w", err)
	}
	return &Parser{
		l: l,
	}, nil
}

func (p *Parser) Parse() (Expression, error) {
	return p.nextExpression()
}

func (p *Parser) nextExpression() (Expression, error) {
	// TODO
	return nil, nil
}
