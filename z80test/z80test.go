// Package z80test allows you to write test cases for z80 code.
package z80test

import (
	"fmt"

	"github.com/paulhankin/z80asm"
	"github.com/paulhankin/z80asm/z80test/z80"
)

type NextMachine struct {
	// RAM stores the banks of RAM in bank order.
	// The default banks stored in slots 0 to 7 are
	// ROM, ROM, 10, 11, 4, 5, 0, 1
	RAM []byte

	af, bc, de, hl, ix, iy uint16
	bc_, de_, hl_          uint16
	pc, sp                 uint16

	// TODO: hardware registers, ports
}

type Config struct {
	Core            z80asm.Z80Core
	MaxInstructions int // Maximum number of instructions to execute.

	// StackTop is the location of the stack.
	// The value 0 means the stack grows backwards from the top of memory.
	StackTop    uint16
	NextMachine *NextMachine
}

// ErrorMaxInstructions is an error that is returned when the code reached
// the maximum number of instructions (as set in the config).
type ErrorMaxInstructions struct {
	MaxInstructions int
}

func (emi ErrorMaxInstructions) Error() string {
	return fmt.Sprintf("maximum number of instructions reached: %d", emi.MaxInstructions)
}

// ErrorPanic is returned when the interpreter panics (for example, when
// it executes an unknown instruction).
type ErrorPanic struct {
	Value interface{}
}

func (ep ErrorPanic) Error() string {
	return fmt.Sprintf("panic: %#v", ep.Value)
}

type ErrorHalt struct{}

func (ErrorHalt) Error() string {
	return "code halted"
}

// Call calls the code at the given address (pushing a dummy PC return
// address onto the stack first). It finishes naturally when the
// corresponding `ret` is executed.
// The final machine state (whenever it's not nil) can be read from the
// returned machine.
func Call(c *Config, address uint16) (rm *NextMachine, re error) {
	defer func() {
		if r := recover(); r != nil {
			if re == nil {
				re = ErrorPanic{Value: r}
			} else {
				panic(fmt.Sprintf("paniced when returning error!?\npanic: %v\nerror: %v", r, re))
			}
		}
	}()

	nm := c.NextMachine

	memory, err := NewMemory(2 * 1024 * 1024)
	if err != nil {
		return nil, err
	}
	copy(memory.RAM, nm.RAM)

	var ports z80.PortAccessor
	var registers z80.NextRegisterAccessor
	zm := z80.NewZ80(memory, ports, registers)

	zm.A = nm.A().Get()
	zm.F = nm.F().Get()
	zm.SetBC(nm.BC().Get())
	zm.SetDE(nm.DE().Get())
	zm.SetHL(nm.HL().Get())
	zm.SetBC_(nm.BC_().Get())
	zm.SetDE_(nm.DE_().Get())
	zm.SetHL_(nm.HL_().Get())
	zm.SetIX(nm.IX().Get())
	zm.SetIY(nm.IY().Get())

	halt := c.StackTop - 1
	sp := c.StackTop - 3

	// We put a halt instruction on the stack, then push its address onto the stack.
	memory.WriteByte(halt, 0x76)
	memory.WriteByte(sp, byte(halt&0xff))
	memory.WriteByte(sp+1, byte(halt>>8))

	zm.SetSP(sp)
	zm.SetPC(address)

	instructionCount := 0
	for (instructionCount < c.MaxInstructions) && !zm.Halted {
		zm.DoOpcode()
		instructionCount++
	}

	fm := &NextMachine{
		RAM: memory.RAM,

		af:  uint16(zm.A)<<8 | uint16(zm.F),
		bc:  zm.BC(),
		de:  zm.DE(),
		hl:  zm.HL(),
		ix:  zm.IX(),
		iy:  zm.IY(),
		bc_: zm.BC_(),
		de_: zm.DE_(),
		hl_: zm.HL_(),
		pc:  zm.PC(),
		sp:  zm.SP(),
	}

	if !zm.Halted {
		if instructionCount >= c.MaxInstructions {
			return fm, ErrorMaxInstructions{MaxInstructions: c.MaxInstructions}
		}
		panic("execution stopped without HALT or instruction limit reached")
	}
	if pc := zm.PC(); pc != halt {
		return fm, ErrorHalt{}
	}
	return fm, nil
}
