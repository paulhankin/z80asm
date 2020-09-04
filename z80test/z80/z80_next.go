package z80

import "math/bits"

// This file extends the opcodes to include those of the Spectrum Next.
// https://wiki.specnext.dev/Extended_Z80_instruction_set

func initOpcodesNext() {
	OpcodesMap[SHIFT_0xED+0x23] = instrED__SWAPNIB
	OpcodesMap[SHIFT_0xED+0x24] = instrED__MIRROR_A
	OpcodesMap[SHIFT_0xED+0x27] = instrED__TEST_iNN

	OpcodesMap[SHIFT_0xED+0x28] = instrED__BSLA_DE_B
	OpcodesMap[SHIFT_0xED+0x29] = instrED__BSRA_DE_B
	OpcodesMap[SHIFT_0xED+0x2A] = instrED__BSRL_DE_B
	OpcodesMap[SHIFT_0xED+0x2B] = instrED__BSRF_DE_B
	OpcodesMap[SHIFT_0xED+0x2C] = instrED__BRLC_DE_B

	OpcodesMap[SHIFT_0xED+0x30] = instrED__MUL_D_E
	OpcodesMap[SHIFT_0xED+0x31] = instrED__ADD_HL_A
	OpcodesMap[SHIFT_0xED+0x32] = instrED__ADD_DE_A
	OpcodesMap[SHIFT_0xED+0x33] = instrED__ADD_BC_A
	OpcodesMap[SHIFT_0xED+0x34] = instrED__ADD_HL_iNNNN
	OpcodesMap[SHIFT_0xED+0x35] = instrED__ADD_DE_iNNNN
	OpcodesMap[SHIFT_0xED+0x36] = instrED__ADD_BC_iNNNN

	OpcodesMap[SHIFT_0xED+0x8A] = instrED__PUSH_iNNNN
	OpcodesMap[SHIFT_0xED+0x90] = instrED__OUTINB
	OpcodesMap[SHIFT_0xED+0x91] = instrED__NEXTREG_iNN_iNN
	OpcodesMap[SHIFT_0xED+0x92] = instrED__NEXTREG_iNN_A
	OpcodesMap[SHIFT_0xED+0x93] = instrED__PIXELDN
	OpcodesMap[SHIFT_0xED+0x94] = instrED__PIXELAD
	OpcodesMap[SHIFT_0xED+0x95] = instrED__SETAE
	OpcodesMap[SHIFT_0xED+0x98] = instrED__JP_iC
	OpcodesMap[SHIFT_0xED+0xA4] = instrED__LDIX
	OpcodesMap[SHIFT_0xED+0xA5] = instrED__LDWS
	OpcodesMap[SHIFT_0xED+0xAC] = instrED__LDDX
	OpcodesMap[SHIFT_0xED+0xB4] = instrED__LDIRX
	OpcodesMap[SHIFT_0xED+0xB7] = instrED__LDPIRX
	OpcodesMap[SHIFT_0xED+0xBC] = instrED__LDDRX
}

func notImplementedOpcode() {
	panic("not implemented next opcode")
}

func instrED__SWAPNIB(z80 *Z80) {
	a := z80.A
	z80.A = (a << 4) | (a >> 4)
}
func instrED__MIRROR_A(z80 *Z80) {
	z80.A = bits.Reverse8(z80.A)
}
func instrED__TEST_iNN(z80 *Z80) {
	notImplementedOpcode()
}

func instrED__BSLA_DE_B(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__BSRA_DE_B(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__BSRL_DE_B(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__BSRF_DE_B(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__BRLC_DE_B(z80 *Z80) {
	notImplementedOpcode()
}

func instrED__MUL_D_E(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_HL_A(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_DE_A(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_BC_A(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_HL_iNNNN(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_DE_iNNNN(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__ADD_BC_iNNNN(z80 *Z80) {
	notImplementedOpcode()
}

func instrED__PUSH_iNNNN(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__OUTINB(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__NEXTREG_iNN_iNN(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__NEXTREG_iNN_A(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__PIXELDN(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__PIXELAD(z80 *Z80) {
	d := uint16(z80.D)
	e := uint16(z80.E)
	hl := 0x4000 + ((d & 0xc0) << 5) + ((d & 0x7) << 8) + ((d & 0x38) << 2) + (e >> 3)
	z80.hl.set(hl)
}
func instrED__SETAE(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__JP_iC(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDIX(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDWS(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDDX(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDIRX(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDPIRX(z80 *Z80) {
	notImplementedOpcode()
}
func instrED__LDDRX(z80 *Z80) {
	notImplementedOpcode()
}
