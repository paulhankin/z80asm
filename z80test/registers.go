package z80test

type Register16 struct {
	value *uint16
}

func (r Register16) High() Register8 {
	return Register8{value: r.value, high: true}
}

func (r Register16) Low() Register8 {
	return Register8{value: r.value, high: false}
}

func (r Register16) Set(x int) {
	*r.value = uint16(x)
}

func (r Register16) Get() uint16 {
	return *r.value
}

func (r Register8) Set(x int) {
	if r.high {
		*r.value = (*r.value)&0xff + uint16(x<<8)
	} else {
		*r.value = (*r.value)&0xff00 + uint16(x&0xff)
	}
}

func (r Register8) Get() uint8 {
	if r.high {
		return uint8((*r.value) >> 8)
	} else {
		return uint8(*r.value & 0xff)
	}
}

type Register8 struct {
	value *uint16
	high  bool
}

func (tc *NextMachine) AF() Register16 {
	return Register16{value: &tc.af}
}
func (tc *NextMachine) BC() Register16 {
	return Register16{value: &tc.bc}
}
func (tc *NextMachine) DE() Register16 {
	return Register16{value: &tc.de}
}
func (tc *NextMachine) HL() Register16 {
	return Register16{value: &tc.hl}
}
func (tc *NextMachine) BC_() Register16 {
	return Register16{value: &tc.bc_}
}
func (tc *NextMachine) DE_() Register16 {
	return Register16{value: &tc.de_}
}
func (tc *NextMachine) HL_() Register16 {
	return Register16{value: &tc.hl_}
}
func (tc *NextMachine) IX() Register16 {
	return Register16{value: &tc.ix}
}
func (tc *NextMachine) IY() Register16 {
	return Register16{value: &tc.iy}
}
func (tc *NextMachine) PC() Register16 {
	return Register16{value: &tc.pc}
}
func (tc *NextMachine) SP() Register16 {
	return Register16{value: &tc.sp}
}

func (tc *NextMachine) A() Register8 {
	return tc.AF().High()
}
func (tc *NextMachine) F() Register8 {
	return tc.AF().Low()
}

func (tc *NextMachine) B() Register8 {
	return tc.BC().High()
}
func (tc *NextMachine) C() Register8 {
	return tc.BC().Low()
}

func (tc *NextMachine) H() Register8 {
	return tc.HL().High()
}
func (tc *NextMachine) L() Register8 {
	return tc.HL().Low()
}

func (tc *NextMachine) D() Register8 {
	return tc.DE().High()
}
func (tc *NextMachine) E() Register8 {
	return tc.DE().Low()
}

func (tc *NextMachine) B_() Register8 {
	return tc.BC_().High()
}
func (tc *NextMachine) C_() Register8 {
	return tc.BC_().Low()
}

func (tc *NextMachine) H_() Register8 {
	return tc.HL_().High()
}
func (tc *NextMachine) L_() Register8 {
	return tc.HL_().Low()
}

func (tc *NextMachine) D_() Register8 {
	return tc.DE_().High()
}
func (tc *NextMachine) E_() Register8 {
	return tc.DE_().Low()
}
