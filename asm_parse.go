package z80asm

import (
	"fmt"
	"log"
	"strconv"
	"text/scanner"
)

type expr interface {
	evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error)
}

type exprInt struct {
	i int64
}

type exprString struct {
	s string
}

type exprUnaryOp struct {
	op rune
	e  expr
}

type exprBinaryOp struct {
	op     rune
	e1, e2 expr
}

func (e exprString) String() string {
	return fmt.Sprintf("%q", e.s)
}

func (e exprBinaryOp) String() string {
	return fmt.Sprintf("[%s %c %s]", e.e1, e.op, e.e2)
}

func (en exprUnaryOp) String() string {
	return fmt.Sprintf("%c%s", en.op, en.e)
}

type exprBracket struct {
	e expr
}

func (eb exprBracket) String() string {
	return fmt.Sprintf("(%s)", eb.e)
}

func indRegGetReg(a arg) arg {
	switch a {
	case indBC:
		return regBC
	case indHL:
		return regHL
	case indDE:
		return regDE
	case indSP:
		return regSP
	}
	log.Fatalf("passed %s to indRegGetReg", a)
	return void
}

func (es exprString) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	if a != argstring {
		return nil, false, nil
	}
	return []byte(es.s), true, nil
}

func (eb exprBracket) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeInt:
		if top {
			return nil, false, nil
		}
		return eb.e.evalAs(asm, a, false)
	case argTypeIndReg:
		_, ok, err := eb.e.evalAs(asm, indRegGetReg(a), false)
		return nil, ok, err
	case argTypeIndAddress:
		return eb.e.evalAs(asm, addr16, false)
	case argTypeIndRegPlusInt:
		/* TODO */
	case argTypePort:
		return eb.e.evalAs(asm, const8, false)
	case argTypePortC:
		return eb.e.evalAs(asm, regC, false)
	}
	return nil, false, nil
}

type exprIdent struct {
	id string
	r  arg // if non-zero, a register it matches
	cc arg // if non-zero a condition code it matches
}

func (ei exprIdent) String() string {
	return fmt.Sprintf("id:%s", ei.id)
}

func (ei exprIdent) getIntValue(asm *Assembler) (int64, bool, error) {
	if ei.r != 0 || ei.cc != 0 {
		return 0, false, nil
	}
	i, ok := asm.GetLabel(ei.id)
	if asm.pass > 0 && !ok {
		return 0, false, asm.scanErrorf("unknown label %q", ei.id)
	}
	return int64(i), true, nil
}

func (ei exprIdent) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeReg:
		return nil, ei.r == a, nil
	case argTypeCC:
		return nil, ei.cc == a, nil
	case argTypeInt, argTypeAddress, argTypeRelAddress:
		r, ok, err := ei.getIntValue(asm)
		if err != nil || !ok {
			return nil, ok, err
		}
		if argType(a) == argTypeRelAddress && ok {
			// 2 assumes that the length of the instruction is 2 bytes.
			// That happens to be true for all the z80 instructions
			// that take a relative offset.
			r -= int64(asm.p + 2)
		}
		return serializeIntArg(asm, r, a)
	}
	return nil, false, nil
}

type exprChar struct {
	r rune
}

func (ec exprChar) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeInt:
		return serializeIntArg(asm, int64(ec.r), a)
	}
	return nil, false, nil
}

func (ec exprChar) String() string {
	return fmt.Sprintf("%c", ec.r)
}

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

func serializeIntArg(asm *Assembler, i int64, a arg) ([]byte, bool, error) {
	min, max, size := argRange(a)
	if i < min || i > max {
		return nil, false, asm.scanErrorf("%x is out of range", i)
	}
	ui := uint16(i)
	switch size {
	case 1:
		return []byte{byte(ui)}, true, nil
	case 2:
		return []byte{byte(ui % 256), byte(ui / 256)}, true, nil
	default:
		log.Fatalf("weird size %d", size)
	}
	return nil, false, fmt.Errorf("internal error")

}

func bool2int(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func (euo exprUnaryOp) apply(n1 int64) int64 {
	switch euo.op {
	case '!':
		return bool2int(n1 == 0)
	case '^':
		return ^n1
	case '-':
		return -n1
	}
	log.Fatalf("Unknown unary op %c", euo.op)
	return 0
}

func (ebo exprBinaryOp) apply(asm *Assembler, n1 int64, e2 expr) (int64, error) {
	var n2 int64
	if ebo.op != tokAndAnd && ebo.op != tokOrOr {
		var err error
		var ok bool
		n2, ok, err = getIntValue(asm, e2)
		if err != nil {
			return 0, err
		} else if !ok {
			return 0, asm.scanErrorf("can't compute constant: %s", e2)
		}
	}
	switch ebo.op {
	case '+':
		return n1 + n2, nil
	case '-':
		return n1 - n2, nil
	case '*':
		return n1 * n2, nil
	case '/':
		if n2 == 0 {
			return 0, fmt.Errorf("divide by zero")
		}
		return n1 / n2, nil
	case '%':
		if n2 == 0 {
			return 0, fmt.Errorf("second arg of % must be non-zero")
		}
		return n1 % n2, nil
	case '&':
		return n1 & n2, nil
	case tokAndNot:
		return n1 &^ n2, nil
	case '|':
		return n1 | n2, nil
	case tokEqEq:
		return bool2int(n1 == n2), nil
	case tokNotEq:
		return bool2int(n1 != n2), nil
	case tokLTEq:
		return bool2int(n1 <= n2), nil
	case tokGTEq:
		return bool2int(n1 >= n2), nil
	case '<':
		return bool2int(n1 < n2), nil
	case '>':
		return bool2int(n1 > n2), nil
	case tokGTGT:
		if n2 < 0 {
			return 0, fmt.Errorf("shift must be positive")
		}
		return n1 >> uint64(n2), nil
	case tokLTLT:
		if n2 < 0 {
			return 0, fmt.Errorf("shift must be positive")
		}
		return n1 << uint64(n2), nil
	case tokAndAnd, tokOrOr:
		if n1 != 0 && ebo.op == tokOrOr || n1 == 0 && ebo.op == tokAndAnd {
			return n1, nil
		}
		n2, ok, err := getIntValue(asm, e2)
		if err != nil {
			return 0, err
		} else if !ok {
			return 0, asm.scanErrorf("can't compute constant: %s", e2)
		}
		return n2, nil
	}
	log.Fatalf("unknown binary op: %s", scanner.TokenString(ebo.op))
	return 0, nil
}

func getIntValue(asm *Assembler, e expr) (int64, bool, error) {
	switch v := e.(type) {
	case exprIdent:
		return v.getIntValue(asm)
	case exprBracket:
		return getIntValue(asm, v.e)
	case exprUnaryOp:
		n, ok, err := getIntValue(asm, v.e)
		if !ok || err != nil {
			return 0, ok, err
		}
		return v.apply(n), true, nil
	case exprInt:
		return v.i, true, nil
	case exprBinaryOp:
		n1, ok1, err1 := getIntValue(asm, v.e1)
		if err1 != nil || !ok1 {
			return 0, ok1, err1
		}
		n, err := v.apply(asm, n1, v.e2)
		if err != nil {
			return 0, false, err
		}
		return n, true, nil
	default:
		return 0, false, nil
	}
}

func (en exprUnaryOp) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	iv, ok, err := getIntValue(asm, en)
	if err != nil || !ok {
		return nil, ok, err
	}
	return exprInt{iv}.evalAs(asm, a, top)
}

func (ebo exprBinaryOp) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	iv, ok, err := getIntValue(asm, ebo)
	if err != nil || !ok {
		return nil, ok, err
	}
	return exprInt{iv}.evalAs(asm, a, false)
}

func (ei exprInt) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeInt, argTypeAddress:
		return serializeIntArg(asm, ei.i, a)
	case argTypeRelAddress:
		/* TODO: figure out what to do here! */
		return nil, false, nil
	case argTypeFixed:
		if !validFixedArgs[ei.i] {
			return nil, false, asm.scanErrorf("0x%x is not a valid argument", ei.i)
		}
		return nil, ei.i == argVals[a], nil
	}
	return nil, false, nil
}

func (ei exprInt) String() string {
	return fmt.Sprintf("%d", ei.i)
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

var opPrecedence = map[rune]int{
	'*':       5,
	'/':       5,
	'%':       5,
	tokLTLT:   5,
	tokGTGT:   5,
	'&':       5,
	tokAndNot: 5,
	'+':       4,
	'-':       4,
	'|':       4,
	'^':       4,
	tokEqEq:   3,
	tokGTEq:   3,
	tokLTEq:   3,
	'<':       3,
	'>':       3,
	tokNotEq:  3,
	tokAndAnd: 2,
	tokOrOr:   1,
}

func (a *Assembler) continueExpr(pri int, ex expr, tok token, err error) (expr, token, error) {
	for err == nil && opPrecedence[tok.t] > 0 && opPrecedence[tok.t] >= pri {
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
			x, tok, err := a.parseExpression(6, false)
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
		case scanner.String:
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
		switch tok.t {
		case ';', '\n', scanner.EOF:
			if comma && !trailingOK {
				return nil, a.scanErrorf("unexpected trailing ,")
			}
			return r, nil
		case ',':
			comma = true
			continue
		default:
			return nil, a.scanErrorf("unexpected %s", tok)
		}
	}
}
