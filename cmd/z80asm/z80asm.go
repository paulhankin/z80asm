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
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/paulhankin/z80asm"
	"github.com/paulhankin/z80asm/z80io"
)

var (
	outFile = flag.String("o", "", "the sna filename to output")
	help    = flag.Bool("help", false, "show usage information about this command.")
)

func pf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
}

func usage() {
	pf("%s is a z80 assembler, which writes ZX Spectrum .sna files\n\n", os.Args[0])
	pf("Usage:\n\n")
	pf("%s <filename>: file to assemble\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Parse()
	if *help {
		usage()
	}
	if len(os.Args) == 1 {
		usage()
	}
	if len(os.Args) > 2 {
		pf("ERROR: too many command-line arguments: %s\n\n", os.Args[1:])
		usage()
	}
	m := z80asm.NewMachine()
	asm, err := z80asm.NewAssembler(m.RAM[:])
	if err != nil {
		pf("%s\n", err)
		os.Exit(1)
	}
	if err := asm.AssembleFile(os.Args[1]); err != nil {
		pf("%s\n", err)
		os.Exit(1)
	}

	value, ok := asm.GetLabel("main")
	if !ok {
		pf("ERROR: missing .main entrypoint in %s\n", os.Args[1:])
		os.Exit(3)
	}
	m.PC = value

	out := *outFile
	if out == "" {
		dir, base := path.Split(os.Args[1])
		ext := path.Ext(os.Args[1])
		out = path.Join(dir, base[:len(base)-len(ext)]+".sna")
	}
	if err := z80io.SaveSNA(out, m); err != nil {
		pf("failed to write .sna file %s: %v\n", out, err)
		os.Exit(3)
	}
}
