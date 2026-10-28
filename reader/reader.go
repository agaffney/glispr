package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
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

// errSkipped is returned by readOne when a macro (such as a comment) consumed
// input without producing an expression.
var errSkipped = errors.New("macro produced no expression")

// readNumberToken parses raw as a number. It returns nil without an error if
// raw is not a number.
//
// parseNumber has no access to the input, so the errors it returns (such as
// division by zero) carry no position information. They must be wrapped in a
// ReaderError here so that callers get the line and offset of the problem.
func readNumberToken(i *Input, raw string, radix int) (*Number, error) {
	num, err := parseNumber(raw, radix)
	if err != nil {
		return nil, newReaderError(i, "%v in %q", err, raw)
	}
	return num, nil
}

func (r *Reader) Read(i *Input) (Expression, error) {
	for {
		expr, err := r.readOne(i)
		if errors.Is(err, errSkipped) {
			continue
		}
		return expr, err
	}
}

// readRequired reads an expression that must be present, so reaching the end
// of the input is an error.
func (r *Reader) readRequired(i *Input) (Expression, error) {
	expr, err := r.Read(i)
	if err != nil {
		return nil, err
	}
	if expr == nil {
		return nil, newReaderError(i, "unexpected end of file")
	}
	return expr, nil
}

func (r *Reader) readOne(i *Input) (Expression, error) {
	tokenBuf := bytes.NewBuffer(nil)
	tokenEscaped := make([]bool, 0)
	multipleEscape := false
	seenEscape := false
	for {
		ch, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if multipleEscape {
					return nil, newReaderError(i, "unexpected end of file in multiple escape")
				}
				if tokenBuf.Len() == 0 {
					return nil, nil
				}
				rawToken := tokenBuf.String()
				if !seenEscape {
					tmpNum, err := readNumberToken(i, rawToken, 10)
					if err != nil {
						return nil, err
					}
					if tmpNum != nil {
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
			return nil, newReaderError(i, "invalid character: %q (%#x)", ch, ch)
		case ReadtableSyntaxTypeWhitespace:
			if tokenBuf.Len() > 0 {
				if err := i.UnreadRune(); err != nil {
					return nil, err
				}
				rawToken := tokenBuf.String()
				if !seenEscape {
					tmpNum, err := readNumberToken(i, rawToken, 10)
					if err != nil {
						return nil, err
					}
					if tmpNum != nil {
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
				if st == ReadtableSyntaxTypeMacroNonTerminating {
					// Non-terminating macro characters are constituents inside a token
					_, _ = tokenBuf.WriteRune(ch)
					tokenEscaped = append(tokenEscaped, false)
					continue
				}
				if err := i.UnreadRune(); err != nil {
					return nil, err
				}
				rawToken := tokenBuf.String()
				if !seenEscape {
					tmpNum, err := readNumberToken(i, rawToken, 10)
					if err != nil {
						return nil, err
					}
					if tmpNum != nil {
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
				return nil, errSkipped
			}
			return expr, nil
		case ReadtableSyntaxTypeSingleEscape:
			// Read next character and write to token buffer
			ch, _, err = i.ReadRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, newReaderError(i, "unexpected end of file after single escape")
				}
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
			if errors.Is(err, io.EOF) {
				return nil, newReaderError(i, "unexpected end of file in string")
			}
			return nil, err
		}
		st = r.rt.SyntaxTypeFunc(ch)
		if !escape {
			if st == ReadtableSyntaxTypeSingleEscape {
				escape = true
				continue
			}
			if ch == initialChar {
				return &String{
					val: buf.String(),
				}, nil
			}
		}
		_, _ = buf.WriteRune(ch)
		escape = false
	}
}

func readLiteral(r *Reader, i *Input, initialChar rune) (Expression, error) {
	item, err := r.readRequired(i)
	if err != nil {
		return nil, err
	}
	return &Literal{
		item: item,
	}, nil
}

func readCloseParen(r *Reader, i *Input, initialChar rune) (Expression, error) {
	return nil, newReaderError(i, "unmatched close parenthesis")
}

func readDispatch(r *Reader, i *Input, initialChar rune) (Expression, error) {
	// initialChar is '#'; the next rune is the dispatch character.
	dispatch, _, err := i.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return &Symbol{name: "#"}, nil
		}
		return nil, err
	}

	switch dispatch {
	case '|':
		if err := readBlockComment(r, i); err != nil {
			return nil, err
		}
		return nil, nil
	case 'b', 'B', 'o', 'O', 'x', 'X':
		// Simple radix number forms: #b, #o, #x
		return readDispatchRadixNumber(r, i, dispatch)
	default:
		// Complex radix number forms: #<radix>r<number>
		if unicode.IsDigit(dispatch) {
			return readDispatchRadixNumber(r, i, dispatch)
		}
	}

	return nil, newReaderError(i, "unknown dispatch macro #%c", dispatch)
}

func readBlockComment(r *Reader, i *Input) error {
	depth := 1
	for depth > 0 {
		ch, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return newReaderError(i, "unterminated block comment")
			}
			return err
		}
		switch ch {
		case '#':
			next, _, err := i.ReadRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return newReaderError(i, "unterminated block comment")
				}
				return err
			}
			if next == '|' {
				depth++
			} else if err := i.UnreadRune(); err != nil {
				return err
			}
		case '|':
			next, _, err := i.ReadRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return newReaderError(i, "unterminated block comment")
				}
				return err
			}
			if next == '#' {
				depth--
			} else if err := i.UnreadRune(); err != nil {
				return err
			}
		}
	}
	return nil
}

// readDispatchRadixNumber reads and parses a radix-prefixed number (#b, #o, #x,
// or #<digits>r<number>) after the initial '#' and dispatch character have
// been consumed. It returns the parsed number or nil, and an optional error
func readDispatchRadixNumber(r *Reader, i *Input, dispatch rune) (Expression, error) {
	radix := 10
	var radixDigits []rune

	if unicode.IsDigit(dispatch) {
		// #<digits>r<number> form
		radixDigits = []rune{dispatch}
		for {
			next, _, err := i.ReadRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
			if unicode.IsDigit(next) {
				radixDigits = append(radixDigits, next)
				continue
			}
			if err := i.UnreadRune(); err != nil {
				return nil, err
			}
			break
		}
		next, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, newReaderError(i, "invalid radix specifier: missing 'r'")
			}
			return nil, err
		}
		if next != 'r' && next != 'R' {
			return nil, newReaderError(i, "invalid radix specifier: expected 'r'")
		}
		radix, err = strconv.Atoi(string(radixDigits))
		if err != nil {
			return nil, newReaderError(i, "invalid radix specifier: %v", err)
		}
		if radix < 2 || radix > 36 {
			return nil, newReaderError(i, "invalid radix specifier: radix must be between 2 and 36")
		}
	} else {
		switch dispatch {
		case 'b', 'B':
			radix = 2
		case 'o', 'O':
			radix = 8
		case 'x', 'X':
			radix = 16
		}
	}

	// Read number digits (without the #<radix> prefix)
	var numDigits strings.Builder
	for {
		next, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		st := r.rt.SyntaxTypeFunc(next)
		if st == ReadtableSyntaxTypeWhitespace ||
			st == ReadtableSyntaxTypeMacroTerminating ||
			st == ReadtableSyntaxTypeMacroNonTerminating ||
			st == ReadtableSyntaxTypeSingleEscape ||
			st == ReadtableSyntaxTypeMultipleEscape ||
			st == ReadtableSyntaxTypeInvalid {
			if err := i.UnreadRune(); err != nil {
				return nil, err
			}
			break
		}
		numDigits.WriteRune(next)
	}

	tmpNum, err := readNumberToken(i, numDigits.String(), radix)
	if err != nil {
		return nil, err
	}
	if tmpNum != nil {
		return tmpNum, nil
	}
	return nil, newReaderError(i, "invalid number %q for radix %d", numDigits.String(), radix)
}

func readList(r *Reader, i *Input, initialChar rune) (Expression, error) {
	items := []Expression{}
	var item Expression
	var ch rune
	var err error
	for {
		ch, _, err = i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, newReaderError(i, "unexpected end of file in list")
			}
			return nil, err
		}
		if r.rt.SyntaxTypeFunc(ch) == ReadtableSyntaxTypeWhitespace {
			continue
		}
		if ch == ')' {
			break
		}
		if err = i.UnreadRune(); err != nil {
			return nil, err
		}
		item, err = r.readOne(i)
		if err != nil {
			if errors.Is(err, errSkipped) {
				continue
			}
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
