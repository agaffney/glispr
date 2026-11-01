package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/agaffney/glispr/types"
)

// A Reader interns symbols, which changes its package set, so readers that
// share a set must not be used concurrently.
type Reader struct {
	rt Readtable
	// Packages is the set of packages that symbols are interned in, and Package
	// is the current package, the equivalent of *package*. It must belong to
	// Packages: reading a symbol with any other value is an error.
	Packages *types.Packages
	Package  *types.Package
	// DefaultFloatFormat is the format of floats with no exponent marker or the
	// marker e. It is the equivalent of *read-default-float-format*, and must be
	// types.FloatFormatSingle or types.FloatFormatDouble: reading such a float
	// with any other value is an error.
	DefaultFloatFormat types.FloatFormat
	// backquoteDepth is the number of backquotes that the form being read is
	// inside, less the commas between it and them.
	backquoteDepth int
}

// NewReader returns a reader that interns symbols in ps, with COMMON-LISP-USER
// as the current package.
func NewReader(ps *types.Packages) *Reader {
	return &Reader{
		rt:                 StandardReadtable,
		Packages:           ps,
		Package:            ps.User(),
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
	expr, err := r.readSymbol(i, tok)
	if err != nil {
		return readResult{}, err
	}
	return readResult{expr: expr, tok: tok}, nil
}

// readSymbol converts a token that is not a number into a symbol (CLHS 2.3.5).
// Its unescaped package markers pick the package:
//
//   - name interns in the current package.
//   - :name interns in KEYWORD.
//   - pkg:name is the external symbol name of pkg, which is an error if there is
//     none. That includes KEYWORD, so keyword:name must already exist, unlike
//     the shorthand :name.
//   - pkg::name interns in pkg.
//
// The package name and the symbol name are case converted separately.
func (r *Reader) readSymbol(i *Input, tok *token) (types.Expression, error) {
	if err := r.checkPackage(i); err != nil {
		return nil, err
	}
	// The markers must be one run of one or two characters, found by the
	// traits so that an escaped colon is not one
	start, end := -1, -1
	for k, c := range tok.chars {
		if c.traits&traitPackageMarker == 0 {
			continue
		}
		switch {
		case start < 0:
			start, end = k, k+1
		case k == end:
			end++
		default:
			return nil, newReaderError(i, "too many package markers in token: %q", tok.String())
		}
	}
	if start < 0 {
		// Use the current package if none is specified
		sym, _ := r.Package.Intern(r.caseConvert(tok))
		return r.symbolExpr(sym), nil
	}
	if end-start > 2 {
		return nil, newReaderError(i, "too many package markers in token: %q", tok.String())
	}
	// Tokens for each part, so case conversion sees only that part's characters
	pkgTok := &token{chars: tok.chars[:start]}
	nameTok := &token{chars: tok.chars[end:]}
	// An empty escape (||) gives an empty name or package name with no characters
	if len(nameTok.chars) == 0 && !tok.hasEscapeAt(end) {
		return nil, newReaderError(i, "symbol name missing after package marker in token: %q", tok.String())
	}
	// The check in finishToken only saw the whole token, which has the marker
	if nameTok.allDots() {
		return nil, newReaderError(i, "token consists only of dots: %q", nameTok.String())
	}
	name := r.caseConvert(nameTok)
	internal := end-start == 2
	shorthand := len(pkgTok.chars) == 0 && !tok.hasEscapeAt(0)
	var pkg *types.Package
	if shorthand {
		// The keyword form has no package name, and :: needs one
		if internal {
			return nil, newReaderError(i, "package name missing before package marker in token: %q", tok.String())
		}
		pkg = r.Packages.Keyword()
	} else {
		pkgName := r.caseConvert(pkgTok)
		if pkg = r.Packages.Find(pkgName); pkg == nil {
			return nil, newReaderError(i, "package %q not found", pkgName)
		}
	}
	// Only the :name shorthand interns with one colon, so that it works for a
	// name that is not yet present
	if !internal && !shorthand {
		// Find also reports symbols the package merely inherits, which are
		// accessible but not external
		sym, status := pkg.Find(name)
		if status == types.SymbolStatusInherited {
			return nil, newReaderError(i, "symbol %q is inherited, not external, in package %q", name, pkg.Name())
		}
		if status != types.SymbolStatusExternal {
			return nil, newReaderError(i, "symbol %q is not external in package %q", name, pkg.Name())
		}
		return r.symbolExpr(sym), nil
	}
	sym, _ := pkg.Intern(name)
	return r.symbolExpr(sym), nil
}

// intern returns the symbol with the given name in the current package (see
// symbolExpr for NIL).
func (r *Reader) intern(i *Input, name string) (types.Expression, error) {
	if err := r.checkPackage(i); err != nil {
		return nil, err
	}
	sym, _ := r.Package.Intern(name)
	return r.symbolExpr(sym), nil
}

// checkPackage returns an error if the current package is not in the package
// set.
func (r *Reader) checkPackage(i *Input) error {
	if r.Packages == nil || r.Package == nil || r.Packages.Find(r.Package.Name()) != r.Package {
		return newReaderError(i, "current package is not in the package set")
	}
	return nil
}

// symbolExpr returns sym, except that COMMON-LISP:NIL is the object Nil.
func (r *Reader) symbolExpr(sym *types.Symbol) types.Expression {
	if sym.Package() == r.Packages.CommonLisp() && sym.Name() == "NIL" {
		return types.Nil
	}
	return sym
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
	return r.readItem(i, &token{}, r.finishToken)
}

// readItem reads from the input until the end of an expression. It starts with
// the characters already in tok, and when the token ends it is converted to a
// result by finish. If there are no characters yet, it can instead read an
// expression using a macro function.
func (r *Reader) readItem(i *Input, tok *token, finish func(*Input, *token) (readResult, error)) (readResult, error) {
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
				return finish(i, tok)
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
				return finish(i, tok)
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
				return finish(i, tok)
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
			tok.markEscape()
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

// readQuote reads the expression after ' and returns it in a list with
// COMMON-LISP:QUOTE.
func readQuote(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	return readQuoteForm(r, i, types.NameQuote)
}

// readQuoteForm reads an expression and returns it in a list with the symbol
// with the given name in COMMON-LISP, which is how 'x and #'x are read.
func readQuoteForm(r *Reader, i *Input, name string) (types.Expression, error) {
	return r.readPrefixForm(i, types.NameCommonLisp, name, 0)
}

// readBackquote reads the expression after ` and returns it in a list with
// GLISPR:QUASIQUOTE.
func readBackquote(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	return r.readPrefixForm(i, types.NameGlispr, types.NameQuasiquote, 1)
}

// readComma reads the expression after , and returns it in a list with
// GLISPR:UNQUOTE, or after ,@ or ,. with UNQUOTE-SPLICING or UNQUOTE-NSPLICING.
// A comma is only allowed inside a backquote, and each comma ends one level of
// it.
func readComma(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	if r.backquoteDepth == 0 {
		return nil, newReaderError(i, "comma is not inside a backquote")
	}
	name := types.NameUnquote
	next, _, err := i.ReadRune()
	switch {
	case err == nil && next == '@':
		name = types.NameUnquoteSplicing
	case err == nil && next == '.':
		name = types.NameUnquoteNSplicing
	case err == nil:
		if err := i.UnreadRune(); err != nil {
			return nil, err
		}
	case !errors.Is(err, io.EOF):
		return nil, err
	}
	return r.readPrefixForm(i, types.NameGlispr, name, -1)
}

// readPrefixForm reads an expression and returns it in a list with the symbol
// with the given name in the package with pkgName. While the expression is read
// the backquote depth changes by depthDelta.
func (r *Reader) readPrefixForm(i *Input, pkgName, name string, depthDelta int) (types.Expression, error) {
	// The package set is needed to find the symbol, but the current package is not
	if r.Packages == nil {
		return nil, newReaderError(i, "no package set")
	}
	r.backquoteDepth += depthDelta
	defer func() { r.backquoteDepth -= depthDelta }()
	item, err := r.readRequired(i)
	if err != nil {
		return nil, err
	}
	// The symbol is always there, as NewPackages exports it
	sym, _ := r.Packages.Find(pkgName).Find(name)
	return types.NewList([]types.Expression{sym, item}), nil
}

func readCloseParen(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	return nil, newReaderError(i, "unmatched close parenthesis")
}

// readCharacter reads the rest of a character after #\. The backslash makes the
// next character part of the token no matter what it is, so #\( is a character.
// A token of one character is that character, with its case as written, and a
// longer one is the name of a character.
func readCharacter(r *Reader, i *Input) (types.Expression, error) {
	ch, _, err := i.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, newReaderError(i, `unexpected end of file after #\`)
		}
		return nil, err
	}
	tok := &token{}
	tok.add(ch, true)
	res, err := r.readItem(i, tok, finishCharacter)
	if err != nil {
		return nil, err
	}
	return res.expr, nil
}

// finishCharacter converts the token after #\ to a character.
func finishCharacter(i *Input, tok *token) (readResult, error) {
	if len(tok.chars) == 1 {
		return readResult{expr: types.NewCharacter(tok.chars[0].r), tok: tok}, nil
	}
	name := tok.String()
	ch, ok := types.CharacterFromName(name)
	if !ok {
		return readResult{}, newReaderError(i, "unrecognized character name %q", name)
	}
	return readResult{expr: types.NewCharacter(ch), tok: tok}, nil
}

// readUninterned reads the token after #: and returns a new symbol with that
// name, which is in no package. Each read makes a different symbol. The token
// is only a name, so it can't be a number or have package markers.
func readUninterned(r *Reader, i *Input) (types.Expression, error) {
	ch, _, err := i.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, newReaderError(i, "symbol name missing after #:")
		}
		return nil, err
	}
	tok := &token{}
	switch r.rt.SyntaxTypeFunc(ch) {
	case ReadtableSyntaxTypeWhitespace, ReadtableSyntaxTypeMacroTerminating:
		return nil, newReaderError(i, "symbol name missing after #:")
	case ReadtableSyntaxTypeMacroNonTerminating:
		// readItem would run its macro function at the start of a token, but
		// here it is the first character of the name
		tok.add(ch, false)
	default:
		if err := i.UnreadRune(); err != nil {
			return nil, err
		}
	}
	res, err := r.readItem(i, tok, r.finishUninterned)
	if err != nil {
		return nil, err
	}
	return res.expr, nil
}

// finishUninterned converts the token after #: to an uninterned symbol.
func (r *Reader) finishUninterned(i *Input, tok *token) (readResult, error) {
	if tok.allDots() {
		return readResult{}, newReaderError(i, "token consists only of dots: %q", tok.String())
	}
	for _, c := range tok.chars {
		if c.traits&traitPackageMarker != 0 {
			return readResult{}, newReaderError(i, "package marker in uninterned symbol: %q", tok.String())
		}
	}
	return readResult{expr: types.NewSymbol(r.caseConvert(tok))}, nil
}

// isASCIIDigit reports whether r is one of 0-9. Digits from other scripts are
// not digits for a dispatch macro's numeric argument.
func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

func readDispatch(r *Reader, i *Input, initialChar rune) (types.Expression, error) {
	// initialChar is '#'; the next rune is the dispatch character.
	dispatch, _, err := i.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return r.intern(i, "#")
		}
		return nil, err
	}

	switch dispatch {
	case '|':
		if err := readBlockComment(r, i); err != nil {
			return nil, err
		}
		return nil, nil
	case '\\':
		return readCharacter(r, i)
	case '(':
		return readVector(r, i, -1)
	case '\'':
		return readQuoteForm(r, i, types.NameFunction)
	case ':':
		return readUninterned(r, i)
	case 'b', 'B':
		return readDispatchRadixNumber(r, i, 2)
	case 'o', 'O':
		return readDispatchRadixNumber(r, i, 8)
	case 'x', 'X':
		return readDispatchRadixNumber(r, i, 16)
	default:
		// Forms with a number argument: #<radix>r<number> and #<length>(...)
		if isASCIIDigit(dispatch) {
			return readDispatchArgument(r, i, dispatch)
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

// readDispatchArgument reads the rest of a dispatch macro that has a number
// argument, such as #16rFF or #3(a b c), once the first digit of it has been
// read.
func readDispatchArgument(r *Reader, i *Input, firstDigit rune) (types.Expression, error) {
	digits := []rune{firstDigit}
	for {
		next, _, err := i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, newReaderError(i, "unexpected end of file after dispatch argument")
			}
			return nil, err
		}
		if isASCIIDigit(next) {
			digits = append(digits, next)
			continue
		}
		arg, err := strconv.Atoi(string(digits))
		switch next {
		case 'r', 'R':
			if err != nil || arg < 2 || arg > 36 {
				return nil, newReaderError(i, "invalid radix specifier: radix must be between 2 and 36")
			}
			return readDispatchRadixNumber(r, i, arg)
		case '(':
			if err != nil || arg > maxVectorLength {
				return nil, newReaderError(i, "invalid vector length: the limit is %d", maxVectorLength)
			}
			return readVector(r, i, arg)
		}
		return nil, newReaderError(i, "invalid dispatch argument: expected 'r' or '('")
	}
}

// readDispatchRadixNumber reads the digits of a number in the given radix, once
// the prefix such as #x or #16r has been read.
func readDispatchRadixNumber(r *Reader, i *Input, radix int) (types.Expression, error) {
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
	items, tail, err := readListItems(r, i, "list", true)
	if err != nil {
		return nil, err
	}
	if tail != nil {
		return types.NewListTail(items, tail), nil
	}
	return types.NewList(items), nil
}

// readListItems reads expressions up to the close paren that ends a list or
// vector, which is named by what in errors. If allowDot is set, a dot before the
// last expression makes that expression the tail, which is returned. The tail is
// nil otherwise.
func readListItems(r *Reader, i *Input, what string, allowDot bool) (items []types.Expression, tail types.Expression, err error) {
	items = []types.Expression{}
	dotSeen := false
	var res readResult
	var ch rune
	for {
		ch, _, err = i.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, nil, newReaderError(i, "unexpected end of file in %s", what)
			}
			return nil, nil, err
		}
		if r.rt.SyntaxTypeFunc(ch) == ReadtableSyntaxTypeWhitespace {
			continue
		}
		if ch == ')' {
			if dotSeen && tail == nil {
				return nil, nil, newReaderError(i, "dot context error: nothing after dot in %s", what)
			}
			return items, tail, nil
		}
		if err = i.UnreadRune(); err != nil {
			return nil, nil, err
		}
		res, err = r.readOne(i)
		if err != nil {
			if errors.Is(err, errSkipped) {
				continue
			}
			return nil, nil, err
		}
		switch {
		case res.isDot():
			if !allowDot {
				return nil, nil, newReaderError(i, "dot context error: dot in %s", what)
			}
			if len(items) == 0 {
				return nil, nil, newReaderError(i, "dot context error: nothing before dot in %s", what)
			}
			if dotSeen {
				return nil, nil, newReaderError(i, "dot context error: more than one dot in %s", what)
			}
			dotSeen = true
		case dotSeen && tail != nil:
			return nil, nil, newReaderError(i, "dot context error: more than one object after dot in %s", what)
		case dotSeen:
			tail = res.expr
		default:
			items = append(items, res.expr)
		}
	}
}

// maxVectorLength is the largest length a vector can be given with #n(, so that
// a short token can't use up all the memory by asking for a huge vector.
const maxVectorLength = 1 << 20

// readVector reads the rest of a vector after #( or #n(. A length of -1 means
// that none was given. With a length, the vector is filled out to it by
// repeating the last element, and having more elements than that is an error.
func readVector(r *Reader, i *Input, length int) (types.Expression, error) {
	items, _, err := readListItems(r, i, "vector", false)
	if err != nil {
		return nil, err
	}
	if length >= 0 {
		if len(items) > length {
			return nil, newReaderError(i, "vector of length %d has too many elements (%d)", length, len(items))
		}
		if len(items) == 0 && length > 0 {
			return nil, newReaderError(i, "vector of length %d has no elements to fill it with", length)
		}
		items = slices.Grow(items, length-len(items))
		for len(items) < length {
			items = append(items, items[len(items)-1])
		}
	}
	return types.NewVector(items), nil
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
