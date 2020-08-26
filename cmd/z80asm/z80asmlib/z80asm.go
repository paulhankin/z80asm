package z80asmlib

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/paulhankin/z80asm"
	"github.com/paulhankin/z80asm/z80io"
)

type Options struct {
	SourceFile string
	OutFile    string
	AsmOptions []z80asm.AssemblerOpt
}

func OptionsFromFlags(args []string) *Options {
	var (
		outFile string
		help    bool
		cpu     string
	)

	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&outFile, "o", "", "the sna filename to output")
	fs.BoolVar(&help, "help", false, "show usage information about this command.")
	fs.StringVar(&cpu, "cpu", "z80", "which cpu to use: z80, z80n1, z80n=z80n2")

	arg0 := args[0]
	if err := fs.Parse(args[1:]); err != nil {
		usage(fs, arg0)
	}
	if help {
		usage(fs, arg0)
	}
	if len(fs.Args()) < 1 {
		usage(fs, arg0)
	}
	if len(fs.Args()) > 1 {
		pf("ERROR: too many command-line arguments: %s\n\n", fs.Args())
		usage(fs, arg0)
	}
	aopts, ok := asmOpts[cpu]
	if !ok {
		pf("ERROR: unrecognized cpu: %q\n", cpu)
		usage(fs, arg0)
	}
	return &Options{
		SourceFile: fs.Arg(0),
		OutFile:    outFile,
		AsmOptions: aopts,
	}
}

var (
	outFile = flag.String("o", "", "the sna filename to output")
	help    = flag.Bool("help", false, "show usage information about this command.")
	cpu     = flag.String("cpu", "z80", "which cpu to use: z80, z80n1, z80n=z80n2")
)

var asmOpts = map[string][]z80asm.AssemblerOpt{
	"z80":   nil,
	"z80n":  []z80asm.AssemblerOpt{z80asm.UseNextCore(2)},
	"z80n2": []z80asm.AssemblerOpt{z80asm.UseNextCore(2)},
	"z80n1": []z80asm.AssemblerOpt{z80asm.UseNextCore(1)},
}

func pf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
}

func usage(fs *flag.FlagSet, arg0 string) {
	pf("%s is a z80 assembler, which writes ZX Spectrum .sna files\n\n", arg0)
	pf("Usage:\n\n")
	pf("%s <filename>: file to assemble\n", arg0)
	fs.PrintDefaults()
	os.Exit(2)
}

func Main(opts *Options) int {
	asm, err := z80asm.NewAssembler(opts.AsmOptions...)
	if err != nil {
		pf("%s\n", err)
		return 1
	}
	if err := asm.AssembleFile(opts.SourceFile); err != nil {
		pf("%s\n", err)
		return 1
	}

	m, err := z80io.NewSNAMachine(asm.RAM())
	if err != nil {
		pf("%s\n", err)
		return 1
	}

	value, ok := asm.GetLabel("", "main")
	if !ok {
		pf("ERROR: missing .main entrypoint in %s\n", os.Args[1:])
		return 3
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
		return 3
	}
	return 0
}
