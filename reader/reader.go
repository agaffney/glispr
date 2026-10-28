package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/agaffney/glispr/types"
)

type Reader struct {
	rt Readtable
	// DefaultFloatFormat is the format of floats with no exponent marker or the
	// marker e. It is the equivalent of *read-default-float-format*, and must be
	// types.FloatFormatSingle or types.FloatFormatDouble: reading such a float
	// with any other value is an error.
	DefaultFloatFormat types.FloatFormat
}

func NewReader() *Reader {
	return &Reader{
		rt:                 StandardReadtable,
		DefaultFloatFormat: types.FloatFormatSingle,
	}
}

// errSkipped is returned by readOne when a macro (such as a comment) consumed
// input without producing an expression.
var errSkipped = errors.New("macro produced no expression")

// readNumberToken parses raw as a number, using parse. It returns nil without
// an error if raw is not a number.
//
// The parse functions in types have no access to the input, so the errors they
// return (such as division by zero) carry no position information. They must be
// wrapped in a ReaderError here so that callers get the line and offset of the
// problem.
func readNumberToken(i *Input, raw string, radix int, parse func(string, int) (*types.Number, error)) (*types.Number, error) {
	num, err := parse(raw, radix)
	if err != nil {
		return nil, newReaderError(i, "%v in %q", err, raw)
	}
	return num, nil
}

// readResult is what readOne produces: the expression, plus the token it came
// from when it was read from a token rather than by a macro function.
type readResult struct {
	expr types.Expression
	tok  *token
}

// isDot reports whether the result came from a token that is a single
// unescaped dot. Its expression is the symbol named ".", which must not be used
// as a value: only readList gives the dot a meaning.
func (res readResult) isDot() bool {
	return res.tok != nil && res.tok.isSingleDot()
}

// finishToken converts a completed token into a number or a symbol. A token
// made only of unescaped dots is an error, unless it is a single dot.
func (r *Reader) finishToken(i *Input, tok *token) (readResult, error) {
	if tok.allDots() && !tok.isSingleDot() {
		return readResult{}, newReaderError(i, "token consists only of dots: %q", tok.String())
	}
	if !tok.hasEscape() {
		tmpNum, err := readNumberToken(i, tok.String(), 10, func(val string, radix int) (*types.Number, error) {
			return types.ParseNumber(val, radix, r.DefaultFloatFormat)
		})
		if err != nil {
			return readResult{}, err
		}
		if tmpNum != nil {
			return readResult{expr: tmpNum, tok: tok}, nil
		}
	}
	return readResult{expr: types.NewSymbol(r.caseConvert(tok)), tok: tok}, nil
}

func (r *Reader) Read(i *Input) (types.Expression, error) {
	for {
		res, err := r.readOne(i)
		if errors.Is(err, errSkipped) {
			continue
		}
		if err == nil && res.isDot() {
			return nil, newReaderError(i, "dot context error: dot outside of a list")
		}
		return res.expr, err
	}
}

// readRequired reads an expression that must be present, so reaching the end
// of the input is an error.
func (r *Reader) readRequired(i *Input) (types.Expression, error) {
	expr, err := r.Read(i)
	if err != nil {
		return nil, err
	}
	if expr == nil {
		return nil, newReaderError(i, "unexpected end of file")
	}
	return expr, nil
}

func (r *Reader) readOne(i *Input) (readResult, error) {
	tok := &token{}
	multipleEscape := false
	for {
		ch, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if multipleEscape {
					return readResult{}, newReaderError(i, "unexpected end of file in multiple escape")
				}
				if !tok.started() {
					return readResult{}, nil
				}
				return r.finishToken(i, tok)
			}
			return readResult{}, err
		}
		st := r.rt.SyntaxTypeFunc(ch)
		// Treat valid characters inside multiple escape as constituent syntax type
		if multipleEscape && st != ReadtableSyntaxTypeInvalid && st != ReadtableSyntaxTypeMultipleEscape && st != ReadtableSyntaxTypeSingleEscape {
			st = ReadtableSyntaxTypeConstituent
		}
		switch st {
		case ReadtableSyntaxTypeInvalid:
			return readResult{}, newReaderError(i, "invalid character: %q (%#x)", ch, ch)
		case ReadtableSyntaxTypeWhitespace:
			if tok.started() {
				if err := i.UnreadRune(); err != nil {
					return readResult{}, err
				}
				return r.finishToken(i, tok)
			}
			continue
		case ReadtableSyntaxTypeMacroTerminating, ReadtableSyntaxTypeMacroNonTerminating:
			if tok.started() {
				if st == ReadtableSyntaxTypeMacroNonTerminating {
					// Non-terminating macro characters are constituents inside a token
					tok.add(ch, false)
					continue
				}
				if err := i.UnreadRune(); err != nil {
					return readResult{}, err
				}
				return r.finishToken(i, tok)
			}
			fn, ok := r.rt.MacroFuncs[ch]
			if !ok {
				// Write to token buffer if no macro function defined
				tok.add(ch, false)
				continue
			}
			expr, err := fn(r, i, ch)
			if err != nil {
				return readResult{}, err
			}
			if expr == nil {
				return readResult{}, errSkipped
			}
			return readResult{expr: expr}, nil
		case ReadtableSyntaxTypeSingleEscape:
			// Read next character and write to token buffer
			ch, _, err = i.ReadRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return readResult{}, newReaderError(i, "unexpected end of file after single escape")
				}
				return readResult{}, err
			}
			tok.add(ch, true)
		case ReadtableSyntaxTypeMultipleEscape:
			multipleEscape = !multipleEscape
			tok.sawEscape = true
			continue
		case ReadtableSyntaxTypeConstituent:
			tok.add(ch, multipleEscape)
		default:
			return readResult{}, fmt.Errorf("unknown syntax type %d, this is probably a bug", st)
		}
	}
}

func readString(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
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
				return types.NewString(buf.String()), nil
			}
		}
		_, _ = buf.WriteRune(ch)
		escape = false
	}
}

func readLiteral(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	item, err := r.readRequired(i)
	if err != nil {
		return nil, err
	}
	return types.NewLiteral(item), nil
}

func readCloseParen(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	return nil, newReaderError(i, "unmatched close parenthesis")
}

func readDispatch(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	// initialChar is '#'; the next rune is the dispatch character.
	dispatch, _, err := i.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return types.NewSymbol("#"), nil
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
func readDispatchRadixNumber(r *Reader, i *Input, dispatch rune) (types.Expression, error) {
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

	// Only integers and ratios are allowed, not floats
	tmpNum, err := readNumberToken(i, numDigits.String(), radix, types.ParseRational)
	if err != nil {
		return nil, err
	}
	if tmpNum != nil {
		return tmpNum, nil
	}
	return nil, newReaderError(i, "invalid number %q for radix %d", numDigits.String(), radix)
}

func readList(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	items := []types.Expression{}
	// tail is the object after the dot, once it has been read
	var tail types.Expression
	dotSeen := false
	var res readResult
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
			if dotSeen && tail == nil {
				return nil, newReaderError(i, "dot context error: nothing after dot in list")
			}
			break
		}
		if err = i.UnreadRune(); err != nil {
			return nil, err
		}
		res, err = r.readOne(i)
		if err != nil {
			if errors.Is(err, errSkipped) {
				continue
			}
			return nil, err
		}
		switch {
		case res.isDot():
			if len(items) == 0 {
				return nil, newReaderError(i, "dot context error: nothing before dot in list")
			}
			if dotSeen {
				return nil, newReaderError(i, "dot context error: more than one dot in list")
			}
			dotSeen = true
		case dotSeen && tail != nil:
			return nil, newReaderError(i, "dot context error: more than one object after dot in list")
		case dotSeen:
			tail = res.expr
		default:
			items = append(items, res.expr)
		}
	}
	if tail != nil {
		return types.NewListTail(items, tail), nil
	}
	return types.NewList(items), nil
}

func readComment(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
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

func (r *Reader) caseConvert(tok *token) string {
	out := make([]rune, len(tok.chars))
	for i, c := range tok.chars {
		out[i] = c.r
	}
	switch r.rt.CaseMode {
	case ReadtableCaseModeUpcase:
		for i, c := range tok.chars {
			if !c.escaped {
				out[i] = unicode.ToUpper(out[i])
			}
		}
	case ReadtableCaseModeDowncase:
		for i, c := range tok.chars {
			if !c.escaped {
				out[i] = unicode.ToLower(out[i])
			}
		}
	case ReadtableCaseModeInvert:
		// Only invert when all unescaped letters in the token share the same case.
		allUpper := true
		allLower := true
		hasLetter := false
		for i, c := range tok.chars {
			if c.escaped || !unicode.IsLetter(out[i]) {
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
			for i, c := range tok.chars {
				if !c.escaped && unicode.IsLetter(out[i]) && unicode.IsUpper(out[i]) {
					out[i] = unicode.ToLower(out[i])
				}
			}
		} else if allLower && !allUpper {
			for i, c := range tok.chars {
				if !c.escaped && unicode.IsLetter(out[i]) && unicode.IsLower(out[i]) {
					out[i] = unicode.ToUpper(out[i])
				}
			}
		}
	}
	return string(out)
}
