package z80

// NextRegisterAccessor provides an interface to read and write
// ZX spectrum next hardware registers.
type NextRegisterAccessor interface {
	ReadRegister(reg uint8) byte
	WriteRegister(reg uint8, b byte)
}
