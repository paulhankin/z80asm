// Package z80io can write z80 binary images.
// Currently, ZX Spectrum .sna files are supported.
package z80io

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/paulhankin/z80asm"
)

// SaveSNA writes the given machine to the named file.
// The documentation for WriteSNA contains more information.
func SaveSNA(filename string, m *z80asm.Machine) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}

	if err = WriteSNA(bufio.NewWriter(f), m); err != nil {
		if cerr := f.Close(); cerr != nil {
			log.Printf("Error closing file during failed write: %v", cerr)
		}
		return fmt.Errorf("failed to write SNA file %q: %v", filename, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close SNA file %q: %v", filename, err)
	}
	return nil
}

func pushpc(m *z80asm.Machine) func() {
	m.SP -= 1
	oldH := m.RAM[m.SP]
	m.RAM[m.SP] = uint8(m.PC >> 8)

	m.SP -= 1
	oldL := m.RAM[m.SP]
	m.RAM[m.SP] = uint8(m.PC)

	return func() {
		m.RAM[m.SP] = oldL
		m.SP += 1
		m.RAM[m.SP] = oldH
		m.SP += 1
	}
}

// WriteSNA writes the given machine as a SNA file.
// The writer is flushed before returning.
// The SNA format involves pushing PC onto the stack.
// Thus the written SP, and the two bytes of RAM before
// the given SP will not be the same as in the machine
// image.
// The Machine is modified during saving, but it restored
// before the function returns.
func WriteSNA(f *bufio.Writer, m *z80asm.Machine) error {
	var writeErr error

	undo := pushpc(m)
	defer undo()

	// write byte
	wb := func(b uint8) {
		if writeErr != nil {
			return
		}
		writeErr = f.WriteByte(b)
	}

	// write word little-endian
	ww := func(u uint16) {
		wb(uint8(u))
		wb(uint8(u >> 8))
	}

	wb(m.I)
	for _, reg := range []uint16{m.HL2, m.DE2, m.BC2, m.AF2, m.HL, m.DE, m.BC, m.IY, m.IX} {
		ww(reg)
	}
	var interrupt uint8
	if m.IntEnabled {
		interrupt |= 0x4
	}
	wb(interrupt)
	wb(m.R)
	ww(m.AF)
	ww(m.SP)
	wb(m.IntMode)
	wb(m.BorderColor)
	if writeErr != nil {
		return fmt.Errorf("failed to write header: %v", writeErr)
	}

	for i := 0; i < 16384; i++ {
		if m.RAM[i] != 0 {
			return fmt.Errorf("Non-zero ROM byte %02x found at address %04x", m.RAM[i], i)
		}
	}
	for i := 16384; i < 65536; i++ {
		wb(m.RAM[i])
	}
	if writeErr != nil {
		return fmt.Errorf("failed to write memory: %v", writeErr)
	}
	if err := f.Flush(); err != nil {
		return fmt.Errorf("failed to flush last few bytes: %v", err)
	}
	return nil
}
