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
	tokenEscaped := make([]bool, 0)
	multipleEscape := false
	seenEscape := false
	for {
		ch, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if tokenBuf.Len() == 0 {
					return nil, nil
				}
				rawToken := tokenBuf.String()
				if !seenEscape {
					if tmpNum := NewNumberFromString(rawToken); tmpNum != nil {
						return tmpNum, nil
					}
				}
				return &Symbol{
					name: r.caseConvert(rawToken, tokenEscaped),
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
				rawToken := tokenBuf.String()
				if !seenEscape {
					if tmpNum := NewNumberFromString(rawToken); tmpNum != nil {
						return tmpNum, nil
					}
				}
				return &Symbol{
					name: r.caseConvert(rawToken, tokenEscaped),
				}, nil
			}
			continue
		case ReadtableSyntaxTypeMacroTerminating, ReadtableSyntaxTypeMacroNonTerminating:
			if tokenBuf.Len() > 0 {
				if err := i.UnreadRune(); err != nil {
					return nil, err
				}
				rawToken := tokenBuf.String()
				if !seenEscape {
					if tmpNum := NewNumberFromString(rawToken); tmpNum != nil {
						return tmpNum, nil
					}
				}
				return &Symbol{
					name: r.caseConvert(rawToken, tokenEscaped),
				}, nil
			}
			fn, ok := r.rt.MacroFuncs[ch]
			if !ok {
				// Write to token buffer if no macro function defined
				_, _ = tokenBuf.WriteRune(ch)
				tokenEscaped = append(tokenEscaped, false)
				continue
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
			ch, _, err = i.ReadRune()
			if err != nil {
				return nil, err
			}
			_, _ = tokenBuf.WriteRune(ch)
			tokenEscaped = append(tokenEscaped, true)
			seenEscape = true
		case ReadtableSyntaxTypeMultipleEscape:
			multipleEscape = !multipleEscape
			seenEscape = true
			continue
		case ReadtableSyntaxTypeConstituent:
			_, _ = tokenBuf.WriteRune(ch)
			tokenEscaped = append(tokenEscaped, multipleEscape)
		default:
			return nil, fmt.Errorf("unknown syntax type %d, this is probably a bug", st)
		}
	}
}

func readString(r *Reader, i *Input, initialChar rune) (Expression, error) {
	buf := bytes.NewBuffer(nil)
	var escape bool
	var ch rune
	var err error
	var st ReadtableSyntaxType
	for {
		ch, _, err = i.ReadRune()
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
		ch, _, err = i.ReadRune()
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
		ch, _, err = i.ReadRune()
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

func (r *Reader) caseConvert(raw string, escaped []bool) string {
	runes := []rune(raw)
	out := append([]rune(nil), runes...)
	switch r.rt.CaseMode {
	case ReadtableCaseModeUpcase:
		for i := range out {
			if i < len(escaped) && !escaped[i] {
				out[i] = unicode.ToUpper(out[i])
			}
		}
	case ReadtableCaseModeDowncase:
		for i := range out {
			if i < len(escaped) && !escaped[i] {
				out[i] = unicode.ToLower(out[i])
			}
		}
	case ReadtableCaseModeInvert:
		// Only invert when all unescaped letters in the token share the same case.
		allUpper := true
		allLower := true
		hasLetter := false
		for i := range out {
			if i >= len(escaped) || escaped[i] {
				continue
			}
			if !unicode.IsLetter(out[i]) {
				continue
			}
			hasLetter = true
			if unicode.IsLower(out[i]) {
				allUpper = false
			}
			if unicode.IsUpper(out[i]) {
				allLower = false
			}
		}
		if !hasLetter {
			return string(out)
		}
		if allUpper && !allLower {
			for i := range out {
				if i < len(escaped) && !escaped[i] && unicode.IsLetter(out[i]) && unicode.IsUpper(out[i]) {
					out[i] = unicode.ToLower(out[i])
				}
			}
		} else if allLower && !allUpper {
			for i := range out {
				if i < len(escaped) && !escaped[i] && unicode.IsLetter(out[i]) && unicode.IsLower(out[i]) {
					out[i] = unicode.ToUpper(out[i])
				}
			}
		}
	}
	return string(out)
}

