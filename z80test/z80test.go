// Package z80test allows you to write test cases for z80 code.
package z80test

import (
	"github.com/paulhankin/z80asm"
)

type TestConfig struct {
	Core            z80asm.Z80Core
	MaxInstructions int // Maximum number of instructions to execute.
}

func RunTest(asm *z80asm.Assembler, tc *TestConfig) error {

}
