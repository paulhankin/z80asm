package z80asm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/scanner"
)

// An Assembler can assemble Z80 instructions
// into RAM.
type Assembler struct {
	opener      func(string) (io.ReadCloser, error)
	pass        int
	p           uint16
	l           map[string]uint16
	labelAssign map[string]string
	m           []uint8
	scan        *scanner.Scanner
	scanErr     error
	lastToken   token
}

func openFile(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	return f, err
}

// NewAssembler constructs a new assembler with the given data as RAM.
// By default, the assmebler will assemble code starting at address
// 0x8000.
func NewAssembler(ram []uint8) (*Assembler, error) {
	return &Assembler{
		opener:      openFile,
		p:           0x8000,
		l:           make(map[string]uint16),
		labelAssign: make(map[string]string),
		m:           ram,
	}, nil
}

// AssembleFile reads the named file, and assembles it as z80
// instructions.
func (asm *Assembler) AssembleFile(filename string) error {
	pc := asm.p
	defer func() {
		asm.p = pc
	}()
	for pass := 0; pass < 2; pass++ {
		asm.p = pc
		asm.pass = pass
		if err := asm.assembleFile(filename); pass == 1 && err != nil {
			return err
		}
	}
	return nil
}

func endStatement(t token) bool {
	return t.t == ';' || t.t == scanner.EOF || t.t == '\n'
}

func (asm *Assembler) assembleFile(filename string) error {
	f, err := asm.opener(filename)
	if err != nil {
		return fmt.Errorf("failed to assemble %q: %v", filename, err)
	}
	defer f.Close()

	var scan scanner.Scanner
	scan.Init(f)
	scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments | scanner.SkipComments
	scan.Whitespace = (1 << ' ') | (1 << '\t')
	scan.Position.Filename = filename
	scan.Error = func(s *scanner.Scanner, msg string) {
		asm.scanErr = asm.scanErrorf("%s", msg)
	}
	asm.scan = &scan
	var errs []string
	for asm.canContinue() && len(errs) < 20 {
		if err := asm.assemble(); err != nil {
			errs = append(errs, err.Error())
			for asm.canContinue() && !endStatement(asm.lastToken) {
				asm.nextToken()
			}
		} else {
			break
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (asm *Assembler) location() string {
	return fmt.Sprintf("%s:%d.%d", asm.scan.Position.Filename, asm.scan.Position.Line, asm.scan.Position.Column)
}

func (asm *Assembler) scanErrorf(fs string, args ...interface{}) error {
	return errors.New(asm.location() + ": " + fmt.Sprintf(fs, args...))
}

type token struct {
	t rune
	s string
}

const (
	tokLTLT rune = -iota - 20
	tokGTGT
	tokAndNot
	tokEqEq
	tokNotEq
	tokLTEq
	tokGTEq
	tokAndAnd
	tokOrOr
)

var tokStrings = map[rune]string{
	tokLTLT:   "<<",
	tokGTGT:   ">>",
	tokAndNot: "&^",
	tokEqEq:   "==",
	tokNotEq:  "!=",
	tokLTEq:   "<=",
	tokGTEq:   ">=",
	tokAndAnd: "&&",
	tokOrOr:   "||",
}

var tokOperatorPrefixes = makeOperatorCompletions()

func makeOperatorCompletions() map[rune]map[rune]rune {
	var r = map[rune]map[rune]rune{}
	for k, v := range tokStrings {
		if _, ok := r[rune(v[0])]; !ok {
			r[rune(v[0])] = map[rune]rune{}
		}
		r[rune(v[0])][rune(v[1])] = k
	}
	return r
}

func (asm *Assembler) nextToken() (token, error) {
	t := asm.scan.Scan()
	if asm.scanErr != nil {
		return token{}, asm.scanErr
	}
	if m2 := tokOperatorPrefixes[t]; m2 != nil {
		if tok := m2[asm.scan.Peek()]; tok != 0 {
			asm.scan.Scan()
			return token{tok, ""}, asm.scanErr
		}
	}
	asm.lastToken = token{t, asm.scan.TokenText()}
	return asm.lastToken, asm.scanErr
}

func (t token) String() string {
	switch t.t {
	case scanner.Int:
		return t.s
	case scanner.Ident:
		return fmt.Sprintf(`identifier "%s"`, t.s)
	}
	if s, ok := tokStrings[t.t]; ok {
		return s
	}
	return scanner.TokenString(t.t)
}

func (asm *Assembler) canContinue() bool {
	return asm.scanErr == nil
}

func (asm *Assembler) assemble() error {
	if asm.scanErr != nil {
		return asm.scanErr
	}
	for {
		tok, err := asm.nextToken()
		if err != nil {
			return err
		}
		switch tok.t {
		case scanner.EOF:
			return nil
		case scanner.Ident:
			if err := asm.assembleCommand(tok.s); err != nil {
				return err
			}
		case ';':
			continue
		case '\n':
			continue
		case '.':
			if err := asm.assembleLabel(); err != nil {
				return err
			}
		default:
			return asm.scanErrorf("unexpected %s", tok)
		}
	}
}

func (asm *Assembler) writeByte(u uint8) error {
	if int(asm.p) >= len(asm.m) {
		return asm.scanErrorf("byte write out of range: %d", asm.p)
	}
	asm.m[asm.p] = u
	asm.p++
	return nil
}

func (asm *Assembler) writeBytes(bs []byte) error {
	for _, b := range bs {
		if err := asm.writeByte(b); err != nil {
			return err
		}
	}
	return nil
}

// GetLabel returns the value of the given label.
// It is only valid after the assembler has run.
func (asm *Assembler) GetLabel(l string) (uint16, bool) {
	v, ok := asm.l[l]
	return v, ok
}

type cmdData arg

func (n cmdData) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	for _, arg0 := range args {
		bs, ok, err := arg0.evalAs(asm, arg(n), false)
		if err != nil {
			return err
		}
		if !ok {
			return asm.scanErrorf("bad data value: %s", arg0)
		}
		if err := asm.writeBytes(bs); err != nil {
			return err
		}
	}
	return nil
}

type instrAssembler interface {
	W(a *Assembler) error
}

var commandTable = map[string]instrAssembler{
	"org": commandOrg{},
	"db":  cmdData(const8),
	"dw":  cmdData(const16),
	"ds":  cmdData(argstring),
}

type commandAssembler struct {
	cmd  string
	args map[arg][]byte
}

func (asm *Assembler) argsCompatible(vals []expr, a arg) ([]byte, bool, error) {
	if len(vals) != argLen(a) {
		return nil, false, nil
	}
	if a == 0 {
		return nil, true, nil
	}
	if a >= 1024 {
		a0, ok, err := vals[0].evalAs(asm, a/1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		a1, ok, err := vals[1].evalAs(asm, a%1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		return append(a0, a1...), true, nil
	}
	return vals[0].evalAs(asm, a, true)
}

func (ca commandAssembler) W(asm *Assembler) error {
	vals, err := asm.parseArgs(false)
	if err != nil {
		return err
	}
	found := false
	for argVariant, bs := range ca.args {
		argData, ok, err := asm.argsCompatible(vals, argVariant)
		if err != nil {
			return err
		}
		if ok {
			if found {
				log.Fatalf("more than one variant of %s possible: args %#v, found alt variant %s", ca.cmd, vals, argVariant)
			}
			found = true
			// Longer instructions (bit operations on ix or iy)
			// interleave the fixed part of the instruction with
			// the variable part.
			// For example: sla (ix+4), * -> dd cb 04 26
			n := len(bs)
			if n > 2 {
				n = 2
			}
			if err := asm.writeBytes(bs[:n]); err != nil {
				return err
			}
			if err := asm.writeBytes(argData); err != nil {
				return err
			}
			if err := asm.writeBytes(bs[n:]); err != nil {
				return err
			}
		}
	}
	if !found {
		vs := []string{}
		for _, v := range vals {
			vs = append(vs, fmt.Sprintf("%s", v))
		}
		return asm.scanErrorf("no suitable form of %s found that matches %s %s", ca.cmd, ca.cmd, strings.Join(vs, ", "))
	}

	return nil
}

func joinCommands(cmdss ...map[string]args) map[string]args {
	r := map[string]args{}
	for _, cmds := range cmdss {
		for k, argss := range cmds {
			if r[k] == nil {
				r[k] = args{}
			}
			for a, bs := range argss {
				if _, ok := r[k][a]; ok {
					panic(fmt.Sprintf("duplicate args %s found for %s", a, k))
				}
				r[k][a] = bs
			}
		}
	}
	return r
}

func init() {
	for c0, bs := range commands0arg {
		if _, ok := commandTable[c0]; ok {
			panic("duplicate command: " + c0)
		}
		commandTable[c0] = commandAssembler{c0, map[arg][]byte{void: bs}}
	}
	for c0, os := range joinCommands(commandsArgs, ixCommands, iyCommands) {
		if _, ok := commandTable[c0]; ok {
			panic("duplicate command: " + c0)
		}
		commandTable[c0] = commandAssembler{c0, os}
	}
}

type commandOrg struct{}

func (commandOrg) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return asm.scanErrorf("org takes one argument: %d found", len(args))
	}
	n, ok, err := getIntValue(asm, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("org wants address, found %s", args[0])
	}
	if n < 16384 || n > 65535 {
		return asm.scanErrorf("org %x out of range", n)
	}
	asm.p = uint16(n)
	return nil
}

func (asm *Assembler) assembleCommand(cmdOrig string) error {
	cmd := strings.ToLower(cmdOrig)
	if f, ok := commandTable[cmd]; ok {
		return f.W(asm)
	}
	return asm.scanErrorf("unknown command %v", cmdOrig)
}

func (asm *Assembler) setLabel(label string) error {
	if asm.pass == 1 {
		fass := asm.labelAssign[label]
		if asm.location() != fass {
			return asm.scanErrorf("Label %q redefined. First defined at %s", label, fass)
		}
		return nil
	}
	asm.l[label] = asm.p
	if asm.pass == 0 && asm.labelAssign[label] == "" {
		asm.labelAssign[label] = asm.location()
	}
	return nil
}

func (a *Assembler) assembleLabel() error {
	tok, err := a.nextToken()
	if err != nil {
		return err
	}
	switch tok.t {
	case scanner.Ident:
		return a.setLabel(tok.s)
	default:
		return a.scanErrorf("unexpected %s", tok)
	}
}

func getByte(prefix, bs []byte) (byte, bool) {
	n := len(bs)
	if !bytes.HasPrefix(bs, prefix) || n != len(prefix)+1 {
		return 0, false
	}
	return bs[n-1], true
}
