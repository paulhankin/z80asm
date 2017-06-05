package z80asm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/scanner"
)

// An Assembler can assemble Z80 instructions
// into RAM.
type Assembler struct {
	opener  func(string) (io.ReadCloser, error)
	pass    int
	p       uint16
	l       map[string]uint16
	m       []uint8
	scan    *scanner.Scanner
	scanErr error
}

func openFile(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	return f, err
}

func NewAssembler(ram []uint8) (*Assembler, error) {
	return &Assembler{
		opener: openFile,
		p:      0x8000,
		l:      make(map[string]uint16),
		m:      ram,
	}, nil
}

func (a *Assembler) AssembleFile(filename string) error {
	pc := a.p
	defer func() {
		a.p = pc
	}()
	for pass := 0; pass < 2; pass++ {
		a.p = pc
		a.pass = pass
		if err := a.assembleFile(filename); err != nil {
			return err
		}
	}
	return nil
}

func (a *Assembler) assembleFile(filename string) error {
	f, err := a.opener(filename)
	if err != nil {
		return fmt.Errorf("failed to assemble %q: %v", filename, err)
	}
	defer f.Close()

	var scan scanner.Scanner
	scan.Init(f)
	scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments | scanner.SkipComments
	scan.Whitespace = (1 << ' ') | (1 << '\t')
	scan.Position.Filename = filename
	scan.Error = func(s *scanner.Scanner, msg string) {
		a.scanErr = a.scanErrorf("%s", msg)
	}
	a.scan = &scan
	return a.assemble()
}

func (a *Assembler) scanErrorf(fs string, args ...interface{}) error {
	header := fmt.Sprintf("%s:%d.%d", a.scan.Position.Filename, a.scan.Position.Line, a.scan.Position.Column)
	rest := fmt.Sprintf(fs, args...)
	return errors.New(header + "\t" + rest)
}

func (a *Assembler) assemble() error {
	for a.scanErr == nil {
		t := a.scan.Scan()
		switch t {
		case scanner.EOF:
			return nil
		case scanner.Ident:
			if err := a.assembleCommand(a.scan.TokenText()); err != nil {
				return err
			}
		case ';':
			continue
		case '\n':
			continue
		case '.':
			if err := a.assembleLabel(); err != nil {
				return err
			}
		default:
			return a.scanErrorf("unexpected %s", scanner.TokenString(t))
		}
	}
	return a.scanErrorf("%v", a.scanErr)
}

func (a *Assembler) writeByte(u uint8) error {
	if int(a.p) >= len(a.m) {
		return a.scanErrorf("byte write out of range: %d", a.p)
	}
	a.m[a.p] = u
	a.p += 1
	return nil
}

func (a *Assembler) writeBytes(bs []byte) error {
	for _, b := range bs {
		if err := a.writeByte(b); err != nil {
			return err
		}
	}
	return nil
}

// GetLabel returns the value of the given label.
// It is only valid after the assembler has run.
func (a *Assembler) GetLabel(l string) (uint16, bool) {
	v, ok := a.l[l]
	return v, ok
}

func commandDB(a *Assembler) error {
	args, err := a.parseArgs()
	if err != nil {
		return err
	}
	for _, arg := range args {
		bs, ok, err := arg.evalAs(a, const8, false)
		if err != nil {
			return err
		}
		if !ok {
			return a.scanErrorf("bad 8-bit data value: %s", arg)
		}
		if err := a.writeBytes(bs); err != nil {
			return err
		}
	}
	return nil
}

func b(x ...byte) []byte {
	return x
}

var commands0arg = map[string][]byte{
	"nop":  b(0),
	"di":   b(0xf3),
	"rlca": b(0x07),
	"rla":  b(0x17),
	"daa":  b(0x27),
	"scf":  b(0x37),
	"exx":  b(0xd9),
	"ei":   b(0xfb),
	"rrca": b(0x0f),
	"rra":  b(0x1f),
	"cpl":  b(0x2f),
	"ccf":  b(0x3f),
	"halt": b(0x76),
	"ldi":  b(0xed, 0xa0),
	"ldir": b(0xed, 0xb0),
	"cpi":  b(0xed, 0xa1),
	"cpir": b(0xed, 0xb1),
	"ini":  b(0xed, 0xa2),
	"inir": b(0xed, 0xb2),
	"outi": b(0xed, 0xa3),
	"otir": b(0xed, 0xb3),
	"neg":  b(0xed, 0x44),
	"reti": b(0xed, 0x4d),
	"retn": b(0xed, 0x45),
	"rrd":  b(0xed, 0x67),
	"ldd":  b(0xed, 0xa8),
	"lddr": b(0xed, 0xb8),
	"cpd":  b(0xed, 0xa9),
	"cpdr": b(0xed, 0xb9),
	"ind":  b(0xed, 0xaa),
	"indr": b(0xed, 0xba),
	"outd": b(0xed, 0xab),
	"otdr": b(0xed, 0xbb),
	"rld":  b(0xed, 0x6f),
}

func stdOpts(arg1 arg, base byte, prefix ...byte) args {
	r := args{}
	for i, reg := range []arg{
		regB, regC, regD, regE, regH, regL, indHL, regA,
	} {
		r[arg2(arg1, reg)] = append(b(prefix...), base+byte(i))
	}
	return r
}

func joinOpts(os ...args) args {
	r := args{}
	for _, o := range os {
		for k, v := range o {
			if _, ok := r[k]; ok {
				log.Fatalf("%s found in two args", k)
			}
			r[k] = v
		}
	}
	return r
}

func rmOpt(os args, o arg) args {
	r := args{}
	found := false
	for k, v := range os {
		if k == o {
			found = true
			continue
		}
		r[k] = v
	}
	if !found {
		log.Fatalf("asked to remove %s, but not found", o)
	}
	return r
}

type arg int

func arg2(arg1, arg2 arg) arg {
	return 1024*arg1 + arg2
}

const (
	void arg = iota
	regA
	regB
	regC
	regD
	regE
	regH
	regL
	regI
	regR
	regAF
	regAF_
	regBC
	regDE
	regHL
	regSP
	regIX
	regIXL
	regIXH
	regIY
	regIYH
	regIYL
	indBC
	indDE
	indHL
	indIXplus
	indIYplus
	indSP
	const8 // TODO: check these are used
	const16
	constS8
	addr16 // TODO: use this consistently
	reladdr8
	port8
	portC
	ind16
	ccNZ
	ccNC
	ccPO
	ccP
	ccZ
	ccC
	ccPE
	ccM
	val00h
	val01h
	val02h
	val03h
	val04h
	val05h
	val06h
	val07h
	val08h
	val10h
	val18h
	val20h
	val28h
	val30h
	val38h
)

var argMap = map[arg]string{
	regA:      "a",
	regB:      "b",
	regC:      "c",
	regD:      "d",
	regE:      "e",
	regH:      "h",
	regL:      "l",
	regI:      "i",
	regR:      "r",
	regAF:     "af",
	regAF_:    "af'",
	regBC:     "bc",
	regDE:     "de",
	regHL:     "hl",
	regSP:     "sp",
	regIX:     "ix",
	regIXL:    "ixl",
	regIXH:    "ixh",
	regIY:     "iy",
	regIYH:    "iyh",
	regIYL:    "iyl",
	indBC:     "(bc)",
	indDE:     "(de)",
	indHL:     "(hl)",
	indIXplus: "(ix+*)",
	indIYplus: "(iy+*)",
	indSP:     "(sp)",
	const8:    "*",
	const16:   "**",
	constS8:   "*",
	addr16:    "**",
	reladdr8:  "*",
	port8:     "(*)",
	portC:     "(c)",
	ind16:     "(**)",
	ccNZ:      "nz",
	ccNC:      "nc",
	ccPO:      "po",
	ccP:       "p",
	ccZ:       "z",
	ccC:       "c",
	ccPE:      "pe",
	ccM:       "m",
	val00h:    "0",
	val01h:    "1",
	val02h:    "2",
	val03h:    "3",
	val04h:    "4",
	val05h:    "5",
	val06h:    "6",
	val07h:    "7",
	val08h:    "0x08",
	val10h:    "0x10",
	val18h:    "0x18",
	val20h:    "0x20",
	val28h:    "0x28",
	val30h:    "0x30",
	val38h:    "0x38",
}

func (o arg) String() string {
	if o >= 1024 {
		return fmt.Sprintf("%s, %s", o/1024, o%1024)
	}
	r, ok := argMap[o]
	if !ok {
		return fmt.Sprintf("arg[%d]", int(o))
	}
	return r
}

type args map[arg][]byte

var commandsArgs = map[string]args{
	"inc": args{
		regA:  b(0x3c),
		regB:  b(0x04),
		regC:  b(0x0c),
		regD:  b(0x14),
		regE:  b(0x1c),
		regH:  b(0x24),
		regL:  b(0x2c),
		regBC: b(0x03),
		regDE: b(0x13),
		regHL: b(0x23),
		regSP: b(0x33),
		indHL: b(0x34),
	},
	"dec": args{
		regA:  b(0x3d),
		regB:  b(0x05),
		regC:  b(0x0d),
		regD:  b(0x15),
		regE:  b(0x1d),
		regH:  b(0x25),
		regL:  b(0x2d),
		regBC: b(0x0b),
		regDE: b(0x1b),
		regHL: b(0x2b),
		regSP: b(0x3b),
		indHL: b(0x35),
	},
	"djnz": args{reladdr8: b(0x10)},
	"sub":  joinOpts(stdOpts(0, 0x90), args{const8: b(0xd6)}),
	"and":  joinOpts(stdOpts(0, 0xa0), args{const8: b(0xe6)}),
	"xor":  joinOpts(stdOpts(0, 0xa8), args{const8: b(0xee)}),
	"or":   joinOpts(stdOpts(0, 0xb0), args{const8: b(0xf6)}),
	"cp":   joinOpts(stdOpts(0, 0xb8), args{const8: b(0xfe)}),
	"rlc":  stdOpts(0, 0x00, 0xcb),
	"rrc":  stdOpts(0, 0x08, 0xcb),
	"rl":   stdOpts(0, 0x10, 0xcb),
	"rr":   stdOpts(0, 0x18, 0xcb),
	"sla":  stdOpts(0, 0x20, 0xcb),
	"sra":  stdOpts(0, 0x28, 0xcb),
	"sll":  stdOpts(0, 0x30, 0xcb),
	"srl":  stdOpts(0, 0x38, 0xcb),
	"ld": joinOpts(
		args{
			arg2(regBC, const16): b(0x01),
			arg2(regDE, const16): b(0x11),
			arg2(regHL, const16): b(0x21),
			arg2(regSP, const16): b(0x31),
			arg2(indBC, regA):    b(0x02),
			arg2(indDE, regA):    b(0x12),
			arg2(ind16, regHL):   b(0x22),
			arg2(ind16, regA):    b(0x32),
			arg2(regB, const8):   b(0x06),
			arg2(regD, const8):   b(0x16),
			arg2(regH, const8):   b(0x26),
			arg2(indHL, const8):  b(0x36),
			arg2(regA, indBC):    b(0x0a),
			arg2(regA, indDE):    b(0x1a),
			arg2(regHL, ind16):   b(0x2a),
			arg2(regA, ind16):    b(0x3a),
			arg2(regC, const8):   b(0x0e),
			arg2(regE, const8):   b(0x1e),
			arg2(regL, const8):   b(0x2e),
			arg2(regA, const8):   b(0x3e),
			arg2(regSP, regHL):   b(0xf9),
			arg2(ind16, regBC):   b(0xed, 0x43),
			arg2(ind16, regDE):   b(0xed, 0x53),
			arg2(ind16, regSP):   b(0xed, 0x73),
			arg2(regI, regA):     b(0xed, 0x47),
			arg2(regA, regI):     b(0xed, 0x57),
			arg2(regBC, ind16):   b(0xed, 0x4b),
			arg2(regDE, ind16):   b(0xed, 0x5b),
			arg2(regSP, ind16):   b(0xed, 0x7b),
			arg2(regR, regA):     b(0xed, 0x4f),
			arg2(regA, regR):     b(0xed, 0x5f),
		},
		stdOpts(regB, 0x40),
		stdOpts(regD, 0x50),
		stdOpts(regH, 0x60),
		rmOpt(stdOpts(indHL, 0x70), arg2(indHL, indHL)),
		stdOpts(regC, 0x48),
		stdOpts(regE, 0x58),
		stdOpts(regL, 0x68),
		stdOpts(regA, 0x78)),
	"ex": args{
		arg2(regAF, regAF_): b(0x08),
		arg2(indSP, regHL):  b(0xe3),
		arg2(regDE, regHL):  b(0xeb),
	},
	"push": args{
		regBC: b(0xc5),
		regDE: b(0xd5),
		regHL: b(0xe5),
		regAF: b(0xf5),
	},
	"pop": args{
		regBC: b(0xc1),
		regDE: b(0xd1),
		regHL: b(0xe1),
		regAF: b(0xf1),
	},
	"add": joinOpts(
		args{
			arg2(regHL, regBC): b(0x09),
			arg2(regHL, regDE): b(0x19),
			arg2(regHL, regHL): b(0x29),
			arg2(regHL, regSP): b(0x39),
			arg2(regA, const8): b(0xc6),
		},
		stdOpts(regA, 0x80),
	),
	"adc": joinOpts(
		args{
			arg2(regA, const8): b(0xce),
			arg2(regHL, regBC): b(0xed, 0x4a),
			arg2(regHL, regDE): b(0xed, 0x5a),
			arg2(regHL, regHL): b(0xed, 0x6a),
			arg2(regHL, regSP): b(0xed, 0x7a),
		},
		stdOpts(regA, 0x88),
	),
	"sbc": joinOpts(
		args{
			arg2(regA, const8): b(0xde),
			arg2(regHL, regBC): b(0xed, 0x42),
			arg2(regHL, regDE): b(0xed, 0x52),
			arg2(regHL, regHL): b(0xed, 0x62),
			arg2(regHL, regSP): b(0xed, 0x72),
		},
		stdOpts(regA, 0x98),
	),
	"call": args{
		arg2(ccNZ, addr16): b(0xc4),
		arg2(ccNC, addr16): b(0xd4),
		arg2(ccPO, addr16): b(0xe4),
		arg2(ccP, addr16):  b(0xf4),
		arg2(ccZ, addr16):  b(0xcc),
		arg2(ccC, addr16):  b(0xdc),
		arg2(ccPE, addr16): b(0xec),
		arg2(ccM, addr16):  b(0xfc),
		addr16:             b(0xcd),
	},
	"jp": args{
		arg2(ccNZ, addr16): b(0xc2),
		arg2(ccNC, addr16): b(0xd2),
		arg2(ccPO, addr16): b(0xe2),
		arg2(ccP, addr16):  b(0xf2),
		addr16:             b(0xc3),
		arg2(ccZ, addr16):  b(0xca),
		arg2(ccC, addr16):  b(0xda),
		arg2(ccPE, addr16): b(0xea),
		arg2(ccM, addr16):  b(0xfa),
		indHL:              b(0xe9),
	},
	"jr": args{
		reladdr8:             b(0x18),
		arg2(ccZ, reladdr8):  b(0x28),
		arg2(ccC, reladdr8):  b(0x38),
		arg2(ccNZ, reladdr8): b(0x20),
		arg2(ccNC, reladdr8): b(0x30),
	},
	"ret": args{
		void: b(0xc9),
		ccNZ: b(0xc0),
		ccNC: b(0xd0),
		ccPO: b(0xe0),
		ccP:  b(0xf0),
		ccZ:  b(0xc8),
		ccC:  b(0xd8),
		ccPE: b(0xe8),
		ccM:  b(0xf8),
	},
	"rst": args{
		val00h: b(0xc7),
		val10h: b(0xd7),
		val20h: b(0xe7),
		val30h: b(0xf7),
		val08h: b(0xcf),
		val18h: b(0xdf),
		val28h: b(0xef),
		val38h: b(0xff),
	},
	"bit": joinOpts(
		stdOpts(val00h, 0x40, 0xcb),
		stdOpts(val01h, 0x48, 0xcb),
		stdOpts(val02h, 0x50, 0xcb),
		stdOpts(val03h, 0x58, 0xcb),
		stdOpts(val04h, 0x60, 0xcb),
		stdOpts(val05h, 0x68, 0xcb),
		stdOpts(val06h, 0x70, 0xcb),
		stdOpts(val07h, 0x78, 0xcb),
	),
	"res": joinOpts(
		stdOpts(val00h, 0x80, 0xcb),
		stdOpts(val01h, 0x88, 0xcb),
		stdOpts(val02h, 0x90, 0xcb),
		stdOpts(val03h, 0x98, 0xcb),
		stdOpts(val04h, 0xa0, 0xcb),
		stdOpts(val05h, 0xa8, 0xcb),
		stdOpts(val06h, 0xb0, 0xcb),
		stdOpts(val07h, 0xb8, 0xcb),
	),
	"set": joinOpts(
		stdOpts(val00h, 0xc0, 0xcb),
		stdOpts(val01h, 0xc8, 0xcb),
		stdOpts(val02h, 0xd0, 0xcb),
		stdOpts(val03h, 0xd8, 0xcb),
		stdOpts(val04h, 0xe0, 0xcb),
		stdOpts(val05h, 0xe8, 0xcb),
		stdOpts(val06h, 0xf0, 0xcb),
		stdOpts(val07h, 0xf8, 0xcb),
	),
	"in": args{
		arg2(regA, port8): b(0xdb),
		arg2(regB, portC): b(0xed, 0x40),
		arg2(regD, portC): b(0xed, 0x50),
		arg2(regH, portC): b(0xed, 0x60),
		arg2(regC, portC): b(0xed, 0x48),
		arg2(regE, portC): b(0xed, 0x58),
		arg2(regL, portC): b(0xed, 0x68),
		arg2(regA, portC): b(0xed, 0x78),
	},
	"out": args{
		arg2(port8, regA): b(0xd3),
		arg2(portC, regB): b(0xed, 0x41),
		arg2(portC, regD): b(0xed, 0x51),
		arg2(portC, regH): b(0xed, 0x61),
		arg2(portC, regC): b(0xed, 0x49),
		arg2(portC, regE): b(0xed, 0x59),
		arg2(portC, regL): b(0xed, 0x69),
		arg2(portC, regA): b(0xed, 0x79),
	},
	"im": args{
		val00h: b(0xed, 0x46),
		val01h: b(0xed, 0x56),
		val02h: b(0xed, 0x5e),
	},
}

type instrAssembler interface {
	W(a *Assembler) error
}

type cmdFunc func(a *Assembler) error

func (cf cmdFunc) W(a *Assembler) error {
	return cf(a)
}

var commandTable = map[string]instrAssembler{
	"org": cmdFunc(commandOrg),
	"db":  cmdFunc(commandDB),
	//"ds":  commandDS,
}

type command0assembler struct {
	bs []byte
}

func (c0a command0assembler) W(a *Assembler) error {
	/* TODO: merge this code with commandAssembler */
	if err := a.writeBytes(c0a.bs); err != nil {
		return err
	}
	return a.statementEnd()
}

type commandAssembler struct {
	cmd  string
	args map[arg][]byte
}

type argumentType int

const (
	argTypeVoid = iota
	argTypeReg
	argTypeCC
	argTypeIndReg
	argTypeIndAddress
	argTypeAddress
	argTypeIndRegPlusInt // (ix+*)
	argTypeRelAddress
	argTypeFixed // a particular number that's not data. Eg: "rst 0x10"
	argTypeInt
	argTypePort
	argTypePortC
	argTypeUnknown // oops
)

func argType(a arg) argumentType {
	switch a {
	case void:
		return argTypeVoid
	case regA, regB, regC, regD, regE, regH, regL, regI, regR, regAF, regAF_, regBC, regDE, regHL, regSP, regIX, regIXL, regIXH, regIY, regIYH, regIYL:
		return argTypeReg
	case indBC, indDE, indHL, indSP:
		return argTypeIndReg
	case indIXplus, indIYplus:
		return argTypeIndRegPlusInt
	case const8, const16, constS8:
		return argTypeInt
	case addr16:
		return argTypeAddress
	case reladdr8:
		return argTypeRelAddress
	case port8:
		return argTypePort
	case portC:
		return argTypePortC
	case ind16:
		return argTypeIndAddress
	case ccNZ, ccNC, ccPO, ccP, ccZ, ccC, ccPE, ccM:
		return argTypeCC
	case val00h, val01h, val02h, val03h, val04h, val05h, val06h, val07h, val08h, val10h, val18h, val20h, val28h, val30h, val38h:
		return argTypeFixed
	}
	return argTypeUnknown
}

func argLen(a arg) int {
	if a == 0 {
		return 0
	}
	if a < 1024 {
		return 1
	}
	return 2
}

func (asm *Assembler) argsCompatible(vals []expr, a arg) ([]byte, bool, error) {
	if len(vals) != argLen(a) {
		return nil, false, nil
	}
	if a == 0 {
		return nil, true, nil
	}
	if a >= 1024 {
		a0, ok, err := vals[0].evalAs(asm, a/1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		a1, ok, err := vals[1].evalAs(asm, a%1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		return append(a0, a1...), true, nil
	}
	return vals[0].evalAs(asm, a, true)
}

func (ca commandAssembler) W(a *Assembler) error {
	vals, err := a.parseArgs()
	if err != nil {
		return err
	}
	found := false
	for argVariant, bs := range ca.args {
		argData, ok, err := a.argsCompatible(vals, argVariant)
		if err != nil {
			return err
		}
		if ok {
			if found {
				log.Fatalf("more than one variant of %s possible: args %#v, found alt variant %s", ca.cmd, vals, argVariant)
			}
			found = true
			if err := a.writeBytes(bs); err != nil {
				return err
			}
			if err := a.writeBytes(argData); err != nil {
				return err
			}
		}
	}
	if !found {
		return a.scanErrorf("no suitable form of %s found for args: %v", ca.cmd, vals)
	}

	return nil
}

func init() {
	for c0, bs := range commands0arg {
		if _, ok := commandTable[c0]; ok {
			panic("duplicate command: " + c0)
		}
		commandTable[c0] = command0assembler{bs}
	}
	for c0, os := range commandsArgs {
		if _, ok := commandTable[c0]; ok {
			panic("duplicate command: " + c0)
		}
		commandTable[c0] = commandAssembler{c0, os}
	}
}

func commandOrg(a *Assembler) error {
	n, err := a.scanNumber()
	if err != nil {
		return a.scanErrorf("org wants address: %v", err)
	}
	if n < 16384 || n > 65535 {
		return a.scanErrorf("org %x out of range", n)
	}
	if err := a.statementEnd(); err != nil {
		return err
	}
	a.p = uint16(n)
	return nil
}

func (a *Assembler) assembleCommand(cmdOrig string) error {
	cmd := strings.ToLower(cmdOrig)
	if f, ok := commandTable[cmd]; ok {
		return f.W(a)
	}
	return a.scanErrorf("unknown command %v", cmdOrig)
}

func (a *Assembler) setLabel(label string) error {
	if _, ok := a.l[label]; a.pass == 0 && ok {
		return a.scanErrorf("label %q already set", label)
	}
	a.l[label] = a.p
	return nil
}

func (a *Assembler) assembleLabel() error {
	t := a.scan.Scan()
	switch t {
	case scanner.Ident:
		return a.setLabel(a.scan.TokenText())
	default:
		return a.scanErrorf("unexpected %s", scanner.TokenString(t))
	}
}

func getByte(prefix, bs []byte) (byte, bool) {
	n := len(bs)
	if !bytes.HasPrefix(bs, prefix) || n != len(prefix)+1 {
		return 0, false
	}
	return bs[n-1], true
}

// GetPlane returns a map of instructions with the given prefix.
func GetPlane(prefix []byte) []string {
	result := make([]string, 256)
	collisions := make([][]string, 256)
	for cmd, asm := range commandTable {
		switch v := asm.(type) {
		case command0assembler:
			if b, ok := getByte(prefix, v.bs); ok {
				result[b] = cmd
				collisions[b] = append(collisions[b], result[b])
			}
		case commandAssembler:
			for o, bs := range v.args {
				if b, ok := getByte(prefix, bs); ok {
					s := fmt.Sprintf("%s %s", cmd, o)
					if o == void {
						s = cmd
					}
					result[b] = s
					collisions[b] = append(collisions[b], result[b])
				}
			}
		}
	}
	failed := false
	for i, c := range collisions {
		if len(c) > 1 {
			fmt.Printf("collisions at 0x%02x: %s\n", i, strings.Join(c, "; "))
			failed = true
		}
	}
	if failed {
		log.Fatalf("found collisions!")
	}
	return result
}
