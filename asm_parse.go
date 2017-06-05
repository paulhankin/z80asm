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

type exprNeg struct {
	e expr
}

type exprBinaryOp struct {
	op     rune
	e1, e2 expr
}

func (e exprBinaryOp) String() string {
	return fmt.Sprintf("[%s %c %s]", e.e1, e.op, e.e2)
}

func (en exprNeg) String() string {
	return fmt.Sprintf("-%s", en.e)
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

func (ebo exprBinaryOp) apply(n1, n2 int64) (int64, error) {
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
	}
	log.Fatalf("unknown binary op: %c", ebo.op)
	return 0, nil
}

func getIntValue(asm *Assembler, e expr) (int64, bool, error) {
	switch v := e.(type) {
	case exprIdent:
		return v.getIntValue(asm)
	case exprBracket:
		return getIntValue(asm, v.e)
	case exprNeg:
		n, ok, err := getIntValue(asm, v.e)
		if !ok || err != nil {
			return 0, ok, err
		}
		return -n, true, nil
	case exprInt:
		return v.i, true, nil
	case exprBinaryOp:
		n1, ok1, err1 := getIntValue(asm, v.e1)
		n2, ok2, err2 := getIntValue(asm, v.e2)
		if err1 != nil {
			return 0, false, err1
		}
		if err2 != nil {
			return 0, false, err2
		}
		if !ok1 || !ok2 {
			return 0, false, nil
		}
		n, err := v.apply(n1, n2)
		if err != nil {
			return 0, false, asm.scanErrorf("error evaluating constant: %v", err)
		}
		return n, true, nil
	default:
		return 0, false, nil
	}
}

func (en exprNeg) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
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

// statementEnd scan meaningless tokens until the next ; EOF or newline.
// Anything meaningful is an error.
func (a *Assembler) statementEnd() error {
	for a.scanErr == nil {
		t := a.scan.Scan()
		switch t {
		case ';', '\n', scanner.EOF:
			return nil
		default:
			return a.scanErrorf("expected end of statement, found %q", scanner.TokenString(t))
		}
	}
	return a.scanErr
}

func (a *Assembler) scanNumber() (int64, error) {
	for a.scanErr == nil {
		t := a.scan.Scan()
		switch t {
		case scanner.Int:
			i, err := strconv.ParseInt(a.scan.TokenText(), 0, 64)
			if err != nil {
				return 0, a.scanErrorf("bad number %q: %v", a.scan.TokenText(), err)
			}
			return i, nil
		default:
			return 0, a.scanErrorf("expected number, found %s", scanner.TokenString(t))
		}
	}
	return 0, a.scanErrorf("expected number, but got error: %v", a.scanErr)
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
	'+': 4,
	'-': 4,
	'*': 5,
	'/': 5,
}

func (a *Assembler) continueExpr(pri int, ex expr, t rune, err error) (expr, rune, error) {
	for err == nil && opPrecedence[t] > 0 && opPrecedence[t] >= pri {
		ex2, t2, err2 := a.parseExpression(opPrecedence[t])
		if err2 != nil {
			return nil, 0, err2
		}
		ex, t, err = exprBinaryOp{t, ex, ex2}, t2, err2
	}
	if err != nil {
		return nil, 0, err
	}
	return ex, t, a.scanErr
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
func (a *Assembler) parseExpression(pri int) (expr, rune, error) {
	for a.scanErr == nil {
		t := a.scan.Scan()
		switch t {
		case ';', '\n', scanner.EOF:
			return nil, t, nil
		case '-':
			x, t, err := a.parseExpression(6)
			if err != nil {
				return nil, 0, err
			}
			return a.continueExpr(pri, exprNeg{x}, t, a.scanErr)
		case '(':
			ex, t, err := a.parseExpression(0)
			if err != nil {
				return nil, 0, err
			}
			if t != ')' {
				return nil, 0, a.scanErrorf("found: %c, expected )", t)
			}
			ex = exprBracket{ex}
			return a.continueExpr(0, ex, a.scan.Scan(), a.scanErr)
		case scanner.Int:
			i, err := strconv.ParseInt(a.scan.TokenText(), 0, 64)
			if err != nil {
				return nil, 0, a.scanErrorf("bad number %q: %v", a.scan.TokenText(), err)
			}
			return a.continueExpr(pri, exprInt{i}, a.scan.Scan(), a.scanErr)
		case scanner.Char:
			r, _, _, err := strconv.UnquoteChar(a.scan.TokenText()[1:], '\'')
			if err != nil {
				return nil, 0, a.scanErrorf("bad char %q: %v", a.scan.TokenText(), err)
			}
			return exprChar{r}, a.scan.Scan(), a.scanErr
		case scanner.Ident:
			expr := exprIdent{
				id: a.scan.TokenText(),
				r:  regFromString[a.scan.TokenText()],
				cc: ccFromString[a.scan.TokenText()],
			}
			return a.continueExpr(pri, expr, a.scan.Scan(), a.scanErr)
		default:
			return nil, 0, a.scanErrorf("unexpected token %q", a.scan.TokenText())
		}
	}
	return nil, 0, a.scanErr
}

func (a *Assembler) parseArgs() ([]expr, error) {
	var r []expr
	for a.scanErr == nil {
		e, t, err := a.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if e != nil {
			r = append(r, e)
		}
		switch t {
		case ';', '\n', scanner.EOF:
			return r, nil
		case ',':
			continue
		default:
			return nil, a.scanErrorf("unexpected character %c", t)
		}
	}
	return nil, a.scanErr
}
