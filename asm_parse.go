package z80asm

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/scanner"
)

type expr interface {
	evalAs(asm *Assembler, a arg) ([]byte, bool, error)
}

type exprInt struct {
	i int64
}

type exprReg struct {
	r arg // always a register
}

func (er exprReg) String() string {
	return fmt.Sprintf("reg:%s", er.r.String())
}

func (er exprReg) evalAs(asm *Assembler, a arg) ([]byte, bool, error) {
	return nil, er.r == a, nil
}

type exprIdent struct {
	id string
}

func (ei exprIdent) String() string {
	return fmt.Sprintf("id:%s", ei.id)
}

func (ei exprIdent) evalAs(asm *Assembler, a arg) ([]byte, bool, error) {
	switch argType(a) {
	case argTypeInt, argTypeAddress, argTypeRelAddress:
		i, ok := asm.GetLabel(ei.id)
		if asm.pass > 0 && !ok {
			return nil, false, asm.scanErrorf("unknown label %q", ei.id)
		}
		r := int64(i)
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

func (ec exprChar) evalAs(asm *Assembler, a arg) ([]byte, bool, error) {
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
	switch size {
	case 1:
		return []byte{byte(i)}, true, nil
	case 2:
		return []byte{byte(i % 256), byte(i / 256)}, true, nil
	default:
		log.Fatalf("weird size %d", size)
	}
	return nil, false, fmt.Errorf("internal error")

}

func (ei exprInt) evalAs(asm *Assembler, a arg) ([]byte, bool, error) {
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

var regFromString map[string]arg = getRegArgs()

func getRegArgs() map[string]arg {
	r := map[string]arg{}
	for a := arg(0); a < 1024; a++ {
		if argType(a) == argTypeReg {
			r[a.String()] = a
		}
	}
	return r
}

// parseExpression parses an expression from the scanner.
// After parsing the expression, the scanner is advanced
// to the token after the expression.
func (a *Assembler) parseExpression() (expr, rune, error) {
	for a.scanErr == nil {
		t := a.scan.Scan()
		switch t {
		case ';', '\n', scanner.EOF:
			/* Return nil for an empty expression */
			return nil, t, nil
		case '-':
			i, err := a.scanNumber()
			if err != nil {
				return nil, 0, err
			}
			return exprInt{-i}, a.scan.Scan(), a.scanErr
		case scanner.Int:
			i, err := strconv.ParseInt(a.scan.TokenText(), 0, 64)
			if err != nil {
				return nil, 0, a.scanErrorf("bad number %q: %v", a.scan.TokenText(), err)
			}
			return exprInt{i}, a.scan.Scan(), a.scanErr
		case scanner.Char:
			r, _, _, err := strconv.UnquoteChar(a.scan.TokenText()[1:], '\'')
			if err != nil {
				return nil, 0, a.scanErrorf("bad char %q: %v", a.scan.TokenText(), err)
			}
			return exprChar{r}, a.scan.Scan(), a.scanErr
		case scanner.Ident:
			if r, ok := regFromString[strings.ToLower(a.scan.TokenText())]; ok {
				return exprReg{r}, a.scan.Scan(), a.scanErr
			}
			return exprIdent{a.scan.TokenText()}, a.scan.Scan(), a.scanErr
		default:
			return nil, 0, a.scanErrorf("unexpected token %q", a.scan.TokenText())
		}
	}
	return nil, 0, a.scanErr
}

func (a *Assembler) parseArgs() ([]expr, error) {
	var r []expr
	for a.scanErr == nil {
		e, t, err := a.parseExpression()
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
