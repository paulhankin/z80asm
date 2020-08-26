// Binary z80asm is a z80 assembler.
// Simple usage:
//   z80asm myfile.z80
//
// This assembles the code in the named file, and writes myfile.sna
// if everything is ok.
//
// The assembler file must define a .main label which is used as
// the entrypoint for the .sna file.
package main

import (
	"fmt"
	"os"

	"github.com/paulhankin/z80asm/cmd/z80asm/z80asmlib"
)

func main() {
	opts := z80asmlib.OptionsFromFlags(os.Args)
	if err := z80asmlib.Main(opts); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(2)
	}
}
