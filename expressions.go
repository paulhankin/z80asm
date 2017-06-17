package z80asm

import (
	"fmt"
	"log"
	"text/scanner"
)

type expr interface {
	evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error)
	stringPri(pri int) string
}

type exprInt struct {
	i int64
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

func (ei exprInt) stringPri(int) string {
	return ei.String()
}

type exprString struct {
	s string
}

func (es exprString) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	if a != argstring {
		return nil, false, nil
	}
	return []byte(es.s), true, nil
}

func (e exprString) String() string {
	return fmt.Sprintf("%q", e.s)
}

func (e exprString) stringPri(int) string {
	return e.String()
}

type exprUnaryOp struct {
	op rune
	e  expr
}

func (en exprUnaryOp) String() string {
	return en.stringPri(0)
}

func (en exprUnaryOp) stringPri(pri int) string {
	return fmt.Sprintf("%c%s", en.op, en.e.stringPri(precUnary))
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

type exprBinaryOp struct {
	op     rune
	e1, e2 expr
}

func (e exprBinaryOp) String() string {
	return e.stringPri(0)
}

func (e exprBinaryOp) stringPri(pri int) string {
	myPri := opPrecedence[e.op]
	left := e.e1.stringPri(myPri)
	right := e.e2.stringPri(myPri + 1)
	result := fmt.Sprintf("%s %c %s", left, e.op, right)
	if myPri < pri {
		return "(" + result + ")"
	}
	return result
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

func (ebo exprBinaryOp) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	iv, ok, err := getIntValue(asm, ebo)
	if err != nil || !ok {
		return nil, ok, err
	}
	return exprInt{iv}.evalAs(asm, a, false)
}

type exprBracket struct {
	e expr
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
	case indIXplus:
		return regIX
	case indIYplus:
		return regIY
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
		switch ex := eb.e.(type) {
		case exprIdent:
			_, ok, err := ex.evalAs(asm, indRegGetReg(a), false)
			if !ok || err != nil {
				return nil, ok, err
			}
			return []byte{0}, true, nil
		case exprBinaryOp:
			_, ok, err := ex.e1.evalAs(asm, indRegGetReg(a), false)
			if !ok || err != nil {
				return nil, ok, err
			}
			if ex.op != '+' && ex.op != '-' {
				return nil, false, asm.scanErrorf("expected %s+n or %s-n, got %c", a, ex.op)
			}
			n, ok, err := getIntValue(asm, ex.e2)
			if !ok {
				return nil, false, asm.scanErrorf("(%s+n) right hand side must be int", a)
			}
			if ex.op == '-' {
				n = -n
			}
			if n < -128 || n > 127 {
				return nil, false, asm.scanErrorf("(%s%+d) out of range -128 to 127", a, n)
			}
			return serializeIntArg(asm, n, const8)
		}
		return nil, false, nil
	case argTypePort:
		return eb.e.evalAs(asm, const8, false)
	case argTypePortC:
		return eb.e.evalAs(asm, regC, false)
	}
	return nil, false, nil
}

func (eb exprBracket) String() string {
	return fmt.Sprintf("(%s)", eb.e)
}

func (eb exprBracket) stringPri(pri int) string {
	if pri > 0 {
		return eb.e.stringPri(pri)
	}
	return eb.String()
}

type exprIdent struct {
	id string
	r  arg // if non-zero, a register it matches
	cc arg // if non-zero a condition code it matches
}

func (ei exprIdent) String() string {
	return fmt.Sprintf("%s", ei.id)
}

func (ei exprIdent) stringPri(int) string {
	return ei.String()
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
			if asm.pass == 0 {
				// We may not have the label defined in pass 0.
				// So we set the relative jump to 0 to make
				// sure it's in range.
				// If it's out of range, pass 1 will catch it.
				r = 0
			} else {
				// 2 assumes that the length of the instruction is 2 bytes.
				// That happens to be true for all the z80 instructions
				// that take a relative offset.
				r -= int64(asm.p + 2)
			}
		}
		return serializeIntArg(asm, r, a)
	}
	return nil, false, nil
}

type exprChar struct {
	r rune
}

func (ec exprChar) String() string {
	return fmt.Sprintf("%c", ec.r)
}

func (ec exprChar) stringPri(int) string {
	return ec.String()
}

func (ec exprChar) evalAs(asm *Assembler, a arg, top bool) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeInt:
		return serializeIntArg(asm, int64(ec.r), a)
	}
	return nil, false, nil
}
