package z80test

import "fmt"

type Memory struct {
	RAM     []byte // 8k banks 0 to (up to) 223.
	ROM     [0x10000]byte
	Layer2  [256 * 192]byte
	Layer2_ [256 * 192]byte

	// Next memory is managed with 8 8k slots.
	// We store read and write slots separately, although
	// mostly they are the same.
	// But Layer2 paging can allow writes to slots 0 and 1
	// to go to layer2 (or shadow layer2) banks.
	// Also, we set the write slot for a ROM bank to nil.
	ReadSlots  [8][]byte
	WriteSlots [8][]byte
}

func (mem *Memory) Bank(n int) []byte {
	return mem.RAM[n*1024*8 : (n+1)*1024*8]
}

func NewMemory(sizeKB int) (*Memory, error) {
	ramBytes := sizeKB * 1024
	if ramBytes%(1024*8) != 0 {
		return nil, fmt.Errorf("RAM must be a multiple of 8kb (got %dkb)", sizeKB)
	}
	mem := &Memory{
		RAM: make([]byte, ramBytes),
	}
	mem.ReadSlots = [8][]byte{
		0: mem.ROM[:1024*8],
		1: mem.ROM[1024*8 : 2*1024*8],
		2: mem.Bank(10),
		3: mem.Bank(11),
		4: mem.Bank(4),
		5: mem.Bank(5),
		6: mem.Bank(0),
		7: mem.Bank(1),
	}
	mem.WriteSlots = [8][]byte{
		2: mem.Bank(10),
		3: mem.Bank(11),
		4: mem.Bank(4),
		5: mem.Bank(5),
		6: mem.Bank(0),
		7: mem.Bank(1),
	}
	return mem, nil
}

func (mem *Memory) CopyBank(n int, bank *[1024 * 8]byte) error {
	if n < 0 || n*1024*8 >= len(mem.RAM) {
		return fmt.Errorf("bank %d out of range (want less than %d)", n, len(mem.RAM)/8/1024)
	}
	copy(mem.Bank(n), bank[:])
	return nil
}

func (mem *Memory) ReadByte(address uint16) byte {
	return mem.ReadByteInternal(address)
}

func (mem *Memory) ReadByteInternal(address uint16) byte {
	return mem.ReadSlots[address>>13][address&0x1fff]
}

func (mem *Memory) WriteByte(address uint16, value byte) {
	mem.WriteByteInternal(address, value)
}

func (mem *Memory) WriteByteInternal(address uint16, value byte) {
	mem.WriteSlots[address>>13][address&0x1fff] = value
}

func (mem *Memory) ContendRead(address uint16, time int)                         {}
func (mem *Memory) ContendReadNoMreq(address uint16, time int)                   {}
func (mem *Memory) ContendReadNoMreq_loop(address uint16, time int, count uint)  {}
func (mem *Memory) ContendWriteNoMreq(address uint16, time int)                  {}
func (mem *Memory) ContendWriteNoMreq_loop(address uint16, time int, count uint) {}

func (mem *Memory) Read(address uint16) byte {
	return mem.ReadByte(address)
}

func (mem *Memory) Write(address uint16, value byte, protectROM bool) {
	mem.WriteByte(address, value)
}

func (mem *Memory) Data() []byte {
	return nil
}
