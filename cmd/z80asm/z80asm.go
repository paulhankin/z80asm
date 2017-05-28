// Binary z80asm is a z80 assembler.
package main

import (
	"log"
	"os"
	"path"

	"github.com/paulhankin/z80asm"
	"github.com/paulhankin/z80asm/z80io"
)

func main() {
	m := z80asm.NewMachine()
	asm, err := z80asm.NewAssembler(m.RAM[:])
	if err != nil {
		log.Fatal(err)
	}
	if err := asm.AssembleFile(os.Args[1]); err != nil {
		log.Fatal(err)
	}

	value, ok := asm.GetLabel("main")
	if !ok {
		log.Fatalf("no .main entrypoint")
	}
	m.PC = value

	dir, base := path.Split(os.Args[1])
	ext := path.Ext(os.Args[1])
	out := path.Join(dir, base[:len(base)-len(ext)]+".sna")
	if err := z80io.SaveSNA(out, m); err != nil {
		log.Fatalf("Failed to write snapshot: %v", err)
	}
}
