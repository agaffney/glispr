package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

type Reader struct {
	rt Readtable
}

func NewReader() *Reader {
	return &Reader{
		rt: StandardReadtable,
	}
}

func (r *Reader) Read(i *Input) (Expression, error) {
	tokenBuf := bytes.NewBuffer(nil)
	var multipleEscape bool
	for {
		ch, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if tokenBuf.Len() == 0 {
					return nil, nil
				}
				return &Symbol{
					name: tokenBuf.String(),
				}, nil
			}
			return nil, err
		}
		st := r.rt.SyntaxTypeFunc(ch)
		// Treat valid characters inside multiple escape as constituent syntax type
		if multipleEscape && st != ReadtableSyntaxTypeInvalid && st != ReadtableSyntaxTypeMultipleEscape && st != ReadtableSyntaxTypeSingleEscape {
			st = ReadtableSyntaxTypeConstituent
		}
		switch st {
		case ReadtableSyntaxTypeInvalid:
			return nil, InvalidCharError{
				ch:      ch,
				line:    i.line,
				linePos: i.linePos,
			}
		case ReadtableSyntaxTypeWhitespace:
			if tokenBuf.Len() > 0 {
				if err := i.UnreadRune(); err != nil {
					return nil, err
				}
				return &Symbol{
					name: tokenBuf.String(),
				}, nil
			}
			continue
		case ReadtableSyntaxTypeMacroTerminating, ReadtableSyntaxTypeMacroNonTerminating:
			if tokenBuf.Len() > 0 {
				if err := i.UnreadRune(); err != nil {
					return nil, err
				}
				return &Symbol{
					name: tokenBuf.String(),
				}, nil
			}
			fn, ok := r.rt.MacroFuncs[ch]
			if !ok {
				return nil, fmt.Errorf("no macro func defined for character %q", ch)
			}
			expr, err := fn(r, i, ch)
			if err != nil {
				return nil, err
			}
			if expr == nil {
				continue
			}
			return expr, nil
		case ReadtableSyntaxTypeSingleEscape:
			// Read next character and write to token buffer
			ch, err = i.ReadRune()
			if err != nil {
				return nil, err
			}
			_, _ = tokenBuf.WriteRune(ch)
		case ReadtableSyntaxTypeMultipleEscape:
			multipleEscape = !multipleEscape
			continue
		case ReadtableSyntaxTypeConstituent:
			if !multipleEscape {
				if r.rt.CaseMode == ReadtableCaseModeUpcase {
					ch = unicode.ToUpper(ch)
				} else if r.rt.CaseMode == ReadtableCaseModeDowncase {
					ch = unicode.ToLower(ch)
				} else if r.rt.CaseMode == ReadtableCaseModeInvert {
					// TODO: keep track of whether non-escaped chars are upper or lower case and invert if all the same
				}
			}
			_, _ = tokenBuf.WriteRune(ch)
		}
	}
	// TODO
	return nil, nil
}

func readString(r *Reader, i *Input, initialChar rune) (Expression, error) {
	buf := bytes.NewBuffer(nil)
	var escape bool
	var ch rune
	var err error
	var st ReadtableSyntaxType
	for {
		ch, err = i.ReadRune()
		if err != nil {
			return nil, err
		}
		st = r.rt.SyntaxTypeFunc(ch)
		switch st {
		case ReadtableSyntaxTypeSingleEscape:
			escape = true
			continue
		default:
			if !escape && ch == initialChar {
				return &String{
					val: buf.String(),
				}, nil
			}
			_, _ = buf.WriteRune(ch)
			escape = false
		}
	}
}

func readLiteral(r *Reader, i *Input, initialChar rune) (Expression, error) {
	item, err := r.Read(i)
	if err != nil {
		return nil, err
	}
	return &Literal{
		item: item,
	}, nil
}

func readList(r *Reader, i *Input, initialChar rune) (Expression, error) {
	items := []Expression{}
	var item Expression
	var ch rune
	var err error
	for {
		ch, err = i.ReadRune()
		if err != nil {
			return nil, err
		}
		if ch == ')' {
			break
		}
		if err = i.UnreadRune(); err != nil {
			return nil, err
		}
		item, err = r.Read(i)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &List{
		items: items,
	}, nil
}

func readComment(r *Reader, i *Input, initialChar rune) (Expression, error) {
	buf := bytes.NewBuffer(nil)
	_, _ = buf.WriteRune(initialChar)
	var ch rune
	var err error
	for {
		ch, err = i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if ch == '\n' {
			break
		}
		_, _ = buf.WriteRune(ch)
	}
	return nil, nil
}
