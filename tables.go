package z80asm

import (
	"fmt"
	"log"
)

func b(x ...byte) []byte {
	return x
}

type arg int

func arg2(arg1, arg2 arg) arg {
	return 1024*arg1 + arg2
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
	argTypeString
	argTypeUnknown // oops
)

func argType(a arg) argumentType {
	switch a {
	case void:
		return argTypeVoid
	case regA, regB, regC, regD, regE, regH, regL, regI, regR, regAF, regAF2, regBC, regDE, regHL, regSP, regIX, regIXL, regIXH, regIY, regIYH, regIYL:
		return argTypeReg
	case indBC, indDE, indHL, indSP, indIX, indIY:
		return argTypeIndReg
	case indIXplus, indIYplus:
		return argTypeIndRegPlusInt
	case const8, const16, const16be, constS8:
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
	case argstring:
		return argTypeString
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
	regAF2
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
	indIX
	indIY
	indIXplus
	indIYplus
	indSP
	const8
	const16
	const16be
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
	argstring // used for directives (eg: ds), not for any z80 instruction
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
	regAF2:    "af'",
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
	indIX:     "(ix)",
	indIY:     "(iy)",
	indIXplus: "(ix+*)",
	indIYplus: "(iy+*)",
	indSP:     "(sp)",
	const8:    "*",
	const16:   "**",
	const16be: "**",
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
	argstring: "...",
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

// extra commands0arg for Z80N on the spectrum next.
var commands0argNext = map[string][]byte{
	"ldix":    b(0xed, 0xa4),
	"ldws":    b(0xed, 0xa5),
	"ldirx":   b(0xed, 0xb4),
	"lddx":    b(0xed, 0xac),
	"lddrx":   b(0xed, 0xbc),
	"ldpirx":  b(0xed, 0xb7),
	"outinb":  b(0xed, 0x90),
	"swapnib": b(0xed, 0x23),
	"pixeldn": b(0xed, 0x93),
	"pixelan": b(0xed, 0x94),
	"setae":   b(0xed, 0x95),
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
	//"sll":  stdOpts(0, 0x30, 0xcb),
	"srl": stdOpts(0, 0x38, 0xcb),
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
		arg2(regAF, regAF2): b(0x08),
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

var commandsArgsNext = map[string]args{
	"add": args{
		arg2(regHL, regA):    b(0xed, 0x31),
		arg2(regDE, regA):    b(0xed, 0x32),
		arg2(regBC, regA):    b(0xed, 0x33),
		arg2(regHL, const16): b(0xed, 0x34),
		arg2(regDE, const16): b(0xed, 0x35),
		arg2(regBC, const16): b(0xed, 0x36),
	},
	"push": args{
		const16be: b(0xed, 0x8a),
	},
}

var (
	ixMap = map[arg]arg{
		regHL: regIX,
		indHL: indIXplus,
	}

	iyMap = map[arg]arg{
		regHL: regIY,
		indHL: indIYplus,
	}

	ixyExcludes = map[string]map[arg]bool{
		"ex":  map[arg]bool{arg2(regDE, regHL): true},
		"jp":  map[arg]bool{indHL: true},
		"sll": map[arg]bool{indHL: true},
	}

	ixCommands = joinCommands(
		replaceCommands(commandsArgs, ixMap, 0xdd, ixyExcludes),
		map[string]args{
			"jp": map[arg][]byte{
				indIX: []byte{0xdd, 0xe9},
			},
		})
	iyCommands = joinCommands(
		replaceCommands(commandsArgs, iyMap, 0xfd, ixyExcludes),
		map[string]args{
			"jp": map[arg][]byte{
				indIY: []byte{0xfd, 0xe9},
			},
		})
)

func doRename(a arg, rename map[arg]arg) arg {
	a0, a1 := a/1024, a%1024
	if rename[a0] != 0 {
		a0 = rename[a0]
	}
	if rename[a1] != 0 {
		a1 = rename[a1]
	}
	return a0*1024 + a1
}

func replaceCommands(cmds map[string]args, rename map[arg]arg, prefix byte, exclude map[string]map[arg]bool) map[string]args {
	result := map[string]args{}
	for k, variants := range cmds {
		for as, bs := range variants {
			if exclude[k][as] {
				continue
			}
			rnas := doRename(as, rename)
			if rnas == as {
				continue
			}
			if result[k] == nil {
				result[k] = map[arg][]byte{}
			}
			result[k][rnas] = append([]byte{prefix}, bs...)
		}
	}
	return result
}
