package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type Token struct {
	Type    TokenType
	Value   string
	Line    int
	LinePos int
}

type TokenType int

const (
	TokenTypeNone TokenType = iota
	TokenTypeEOF
	TokenTypeLParen
	TokenTypeRParen
	TokenTypeQuote
	TokenTypeAtom
	TokenTypeString
	TokenTypeMinus
	TokenTypeNumber
	TokenTypeComment
)

type Lexer struct {
	r           *bufio.Reader
	line        int
	linePos     int
	prevLinePos int
}

func NewLexer(r io.Reader) (*Lexer, error) {
	if r == nil {
		return nil, errors.New("nil io.Reader")
	}
	return &Lexer{
		r: bufio.NewReader(r),
	}, nil
}

func (l *Lexer) readRune() (rune, error) {
	// Reset line number if zero
	l.line = max(l.line, 1)
	c, _, err := l.r.ReadRune()
	if err != nil {
		return 0, err
	}
	// Increment line position
	l.linePos++
	// Check for newline
	if c == '\n' {
		l.line++
		l.prevLinePos = l.linePos
		l.linePos = 0
	}
	return c, nil
}

func (l *Lexer) unreadRune() error {
	err := l.r.UnreadRune()
	if err != nil {
		return err
	}
	l.linePos--
	if l.linePos < 1 {
		l.line--
		l.linePos = l.prevLinePos
	}
	return err
}

func (l *Lexer) newToken(typ TokenType, val string, nums ...int) *Token {
	line := l.line
	linePos := l.linePos
	if len(nums) > 0 {
		line = nums[0]
	}
	if len(nums) > 1 {
		linePos = nums[1]
	}
	return &Token{
		Type:    typ,
		Value:   val,
		Line:    line,
		LinePos: linePos,
	}
}

func (l *Lexer) newError(msg string) error {
	return fmt.Errorf(
		"error at line %d, position %d: %s",
		l.line,
		l.linePos,
		msg,
	)
}

func (l *Lexer) NextToken() (*Token, error) {
	c, err := l.readRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return l.newToken(TokenTypeEOF, ""), nil
		}
		return nil, err
	}
	switch c {
	case '(':
		return l.newToken(TokenTypeLParen, string(c)), nil
	case ')':
		return l.newToken(TokenTypeRParen, string(c)), nil
	case '\'':
		return l.newToken(TokenTypeQuote, string(c)), nil
	case '-':
		return l.newToken(TokenTypeMinus, string(c)), nil
	case '"':
		if err := l.unreadRune(); err != nil {
			return nil, err
		}
		return l.readString()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if err := l.unreadRune(); err != nil {
			return nil, err
		}
		return l.readNumber()
	case ';':
		if err := l.unreadRune(); err != nil {
			return nil, err
		}
		return l.readComment()
	case ' ', '\t', '\n':
		// Skip whitespace/newline
		return l.NextToken()
	default:
		if err := l.unreadRune(); err != nil {
			return nil, err
		}
		return l.readAtom()
	}
}

func (l *Lexer) readString() (*Token, error) {
	buf := bytes.NewBuffer(nil)
	var c rune
	var err error
	var startLine, startLinePos int
	for {
		for {
			c, err = l.readRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, l.newError("unexpected EOF while reading string")
				}
				return nil, err
			}
			if startLine == 0 {
				startLine = l.line
				startLinePos = l.linePos
			}
			if c == '\\' {
				continue
			}
			break
		}
		_, _ = buf.WriteRune(c)
		if c == '"' {
			break
		}
	}
	return l.newToken(TokenTypeString, buf.String(), startLine, startLinePos), nil
}

func (l *Lexer) readNumber() (*Token, error) {
	buf := bytes.NewBuffer(nil)
	var c rune
	var err error
	var startLine, startLinePos int
	for {
		c, err = l.readRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if startLine == 0 {
			startLine = l.line
			startLinePos = l.linePos
		}
		if c >= '0' && c <= '9' {
			_, _ = buf.WriteRune(c)
			continue
		} else if c == '.' {
			// Check if next character is numeric for floating point
			c, err = l.readRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
			if c >= '0' && c <= '9' {
				_, _ = buf.WriteRune('.')
				_, _ = buf.WriteRune(c)
				continue
			}
		}
		// Put back last character if we didn't consume it above
		if err := l.unreadRune(); err != nil {
			return nil, err
		}
		break
	}
	return l.newToken(TokenTypeNumber, buf.String(), startLine, startLinePos), nil
}

func (l *Lexer) readComment() (*Token, error) {
	buf := bytes.NewBuffer(nil)
	var c rune
	var err error
	var startLine, startLinePos int
	for {
		c, err = l.readRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if startLine == 0 {
			startLine = l.line
			startLinePos = l.linePos
		}
		if c == '\n' {
			break
		}
		_, _ = buf.WriteRune(c)
	}
	return l.newToken(TokenTypeComment, buf.String(), startLine, startLinePos), nil
}

func (l *Lexer) readAtom() (*Token, error) {
	buf := bytes.NewBuffer(nil)
	var c rune
	var err error
	var startLine, startLinePos int
	for {
		c, err = l.readRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, l.newError("unexpected EOF while reading atom")
			}
			return nil, err
		}
		if startLine == 0 {
			startLine = l.line
			startLinePos = l.linePos
		}
		if c == '(' || c == ')' || c == '"' || c == ' ' || c == '\t' || c == '\n' {
			if err := l.unreadRune(); err != nil {
				return nil, err
			}
			break
		}
		_, _ = buf.WriteRune(c)
	}
	return l.newToken(TokenTypeAtom, buf.String(), startLine, startLinePos), nil
}
