package z80asm

// A Machine describes the machine state
// of a 48k ZX Spectrum. Except for the ROM.
type Machine struct {
	AF, BC, DE, HL, IX, IY uint16
	AF2, BC2, DE2, HL2     uint16
	SP                     uint16
	PC                     uint16
	I                      uint8
	R                      uint8
	IntEnabled             bool
	IntMode                uint8 // 0, 1 or 2.
	BorderColor            uint8 // 0 to 7.
	RAM                    []uint8
}

// NewMachine returns a newly initialised Machine.
func NewMachine(RAM []uint8) (*Machine, error) {
	return &Machine{
		RAM: RAM,
	}, nil
}
