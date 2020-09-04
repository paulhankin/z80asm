// Package z80test allows you to write test cases for z80 code.
package z80test

import (
	"fmt"

	"github.com/paulhankin/z80asm"
)

type NextMachine struct {
	RAM []byte

	af, bc, de, hl, ix, iy uint16
	bc_, de_, hl_          uint16
	pc, sp                 uint16

	// TODO: hardware registers, ports
}

type Config struct {
	Core            z80asm.Z80Core
	MaxInstructions int // Maximum number of instructions to execute.

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
	return nil, fmt.Errorf("not implemented yet")
}
