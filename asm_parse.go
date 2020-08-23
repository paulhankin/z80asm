package z80asm

import (
	"fmt"
	"log"
	"strconv"
	"text/scanner"
)

var argVals = map[arg]int64{
	val00h: 0,
	val01h: 1,
	val02h: 2,
	val03h: 3,
	val04h: 4,
	val05h: 5,
	val06h: 6,
	val07h: 7,
	val08h: 8,
	val10h: 0x10,
	val18h: 0x18,
	val20h: 0x20,
	val28h: 0x28,
	val30h: 0x30,
	val38h: 0x38,
}

var validFixedArgs = getValidFixedArgs(argVals)

func getValidFixedArgs(m map[arg]int64) map[int64]bool {
	r := make(map[int64]bool)
	for _, v := range m {
		r[v] = true
	}
	return r
}

func argRange(a arg) (min, max, size int64) {
	switch a {
	case const8:
		return -128, 255, 1 // sloppily allow signed or unsigned bytes
	case const16:
		return -32768, 65535, 2
	case const16be:
		return -32768, 65535, 2
	case constS8:
		return -128, 127, 1
	case addr16:
		return 0, 65535, 2
	case reladdr8:
		return -128, 127, 1
	case port8:
		return 0, 255, 1
	case ind16:
		return 0, 65535, 2
	}
	log.Fatalf("argRange(%s)", a)
	return 0, 0, 0
}

func writesBigEndian(a arg) bool {
	_, _, sz := argRange(a)
	if sz != 2 {
		log.Fatalf("got unexpected request for big-endianness for non-2-byte arg %v", a)
	}
	return a == const16be
}

func serializeIntArg(asm *Assembler, i int64, a arg) ([]byte, bool, error) {
	min, max, size := argRange(a)
	if i < min || i > max {
		return nil, false, asm.scanErrorf("%d is not in the range %d...%d", i, min, max)
	}
	ui := uint16(i)
	switch size {
	case 1:
		return []byte{byte(ui)}, true, nil
	case 2:
		if writesBigEndian(a) {
			return []byte{byte(ui / 256), byte(ui % 256)}, true, nil
		} else {
			return []byte{byte(ui % 256), byte(ui / 256)}, true, nil
		}
	default:
		log.Fatalf("weird size %d", size)
	}
	return nil, false, fmt.Errorf("internal error")

}

var (
	regFromString = getMatchingArgs(argTypeReg)
	ccFromString  = getMatchingArgs(argTypeCC)
)

func getMatchingArgs(at argumentType) map[string]arg {
	r := map[string]arg{}
	for a := arg(0); a < 1024; a++ {
		if argType(a) == at {
			r[a.String()] = a
		}
	}
	return r
}

var (
	precUnary = 12

	opPrecedence = map[rune]int{
		'*':       10,
		'/':       10,
		'%':       10,
		tokLTLT:   10,
		tokGTGT:   10,
		'&':       10,
		tokAndNot: 10,
		'+':       8,
		'-':       8,
		'|':       8,
		'^':       8,
		tokEqEq:   6,
		tokGTEq:   6,
		tokLTEq:   6,
		'<':       6,
		'>':       6,
		tokNotEq:  6,
		tokAndAnd: 4,
		tokOrOr:   2,
	}
)

func (a *Assembler) continueExpr(pri int, ex expr, tok token, err error) (expr, token, error) {
	for err == nil && opPrecedence[tok.t] > 0 && opPrecedence[tok.t] > pri {
		ex2, tok2, err2 := a.parseExpression(opPrecedence[tok.t], false)
		if err2 != nil {
			return nil, token{}, err2
		}
		ex, tok, err = exprBinaryOp{tok.t, ex, ex2}, tok2, err2
	}
	return ex, tok, err
}

// parseExpression parses an expression from the scanner.
// After parsing the expression, the scanner is advanced
// to the token after the expression.
// pri is the parsing priority (same as go).
// 6             unary operators
// 5             *  /  %  <<  >>  &  &^
// 4             +  -  |  ^
// 3             ==  !=  <  <=  >  >=
// 2             &&
// 1             ||
func (a *Assembler) parseExpression(pri int, emptyOK bool) (expr, token, error) {
	for {
		tok, err := a.nextToken()
		if err != nil {
			return nil, token{}, err
		}
		switch tok.t {
		case ';', '\n', scanner.EOF:
			if !emptyOK {
				return nil, token{}, a.scanErrorf("unexpected %s", tok)
			}
			return nil, tok, nil
		case '-', '^', '!':
			op := tok.t
			x, tok, err := a.parseExpression(precUnary, false)
			return a.continueExpr(pri, exprUnaryOp{op, x}, tok, err)
		case '(':
			ex, tok, err := a.parseExpression(0, false)
			if err != nil {
				return nil, token{}, err
			}
			if tok.t != ')' {
				return nil, token{}, a.scanErrorf("found: %s, expected )", tok)
			}
			ex = exprBracket{ex}
			nt, err := a.nextToken()
			return a.continueExpr(0, ex, nt, err)
		case scanner.Int:
			i, err := strconv.ParseInt(tok.s, 0, 64)
			if err != nil {
				return nil, token{}, a.scanErrorf("bad number %q: %v", tok, err)
			}
			nt, err := a.nextToken()
			return a.continueExpr(pri, exprInt{i}, nt, err)
		case scanner.String, scanner.RawString:
			r, err := strconv.Unquote(tok.s)
			if err != nil {
				return nil, token{}, a.scanErrorf("bad string %q: %v", tok.s, err)
			}
			nt, err := a.nextToken()
			return a.continueExpr(pri, exprString{r}, nt, err)
		case scanner.Char:
			r, _, _, err := strconv.UnquoteChar(tok.s[1:], '\'')
			if err != nil {
				return nil, token{}, a.scanErrorf("bad char %q: %v", tok, err)
			}
			nt, err := a.nextToken()
			return exprChar{r}, nt, err
		case scanner.Ident:
			expr := exprIdent{
				id: tok.s,
				r:  regFromString[tok.s],
				cc: ccFromString[tok.s],
			}
			nt, err := a.nextToken()
			return a.continueExpr(pri, expr, nt, err)
		default:
			return nil, token{}, a.scanErrorf("unexpected token %s", tok)
		}
	}
}

func (a *Assembler) parseArgs(trailingOK bool) ([]expr, error) {
	return a.parseSepArgs(',', trailingOK)
}

func (a *Assembler) parseSepArgs(sep rune, trailingOK bool) ([]expr, error) {
	var r []expr
	comma := false
	for {
		e, tok, err := a.parseExpression(0, true)
		if err != nil {
			return nil, err
		}
		if e != nil {
			comma = false
			r = append(r, e)
		}
		if tok.t == sep {
			comma = true
			continue
		}
		switch tok.t {
		case ';', '\n', scanner.EOF:
			if comma && !trailingOK {
				return nil, a.scanErrorf("unexpected trailing %c", sep)
			}
			return r, nil
		default:
			return nil, a.scanErrorf("unexpected %s", tok)
		}
	}
}
