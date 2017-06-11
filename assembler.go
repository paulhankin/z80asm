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
	scan_       *scanner.Scanner
	scanErr     error
	lastToken   token
}

func openFile(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	return f, err
}

func NewAssembler(ram []uint8) (*Assembler, error) {
	return &Assembler{
		opener:      openFile,
		p:           0x8000,
		l:           make(map[string]uint16),
		labelAssign: make(map[string]string),
		m:           ram,
	}, nil
}

func (a *Assembler) AssembleFile(filename string) error {
	pc := a.p
	defer func() {
		a.p = pc
	}()
	for pass := 0; pass < 2; pass++ {
		a.p = pc
		a.pass = pass
		if err := a.assembleFile(filename); pass == 1 && err != nil {
			return err
		}
	}
	return nil
}

func endStatement(t token) bool {
	return t.t == ';' || t.t == scanner.EOF || t.t == '\n'
}

func (a *Assembler) assembleFile(filename string) error {
	f, err := a.opener(filename)
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
		a.scanErr = a.scanErrorf("%s", msg)
	}
	a.scan_ = &scan
	var errs []string
	for a.canContinue() && len(errs) < 20 {
		if err := a.assemble(); err != nil {
			errs = append(errs, err.Error())
			for a.canContinue() && !endStatement(a.lastToken) {
				a.nextToken()
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

func (a *Assembler) location() string {
	return fmt.Sprintf("%s:%d.%d", a.scan_.Position.Filename, a.scan_.Position.Line, a.scan_.Position.Column)
}

func (a *Assembler) scanErrorf(fs string, args ...interface{}) error {
	return errors.New(a.location() + ": " + fmt.Sprintf(fs, args...))
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

func (a *Assembler) nextToken() (token, error) {
	t := a.scan_.Scan()
	if a.scanErr != nil {
		return token{}, a.scanErr
	}
	if m2 := tokOperatorPrefixes[t]; m2 != nil {
		if tok := m2[a.scan_.Peek()]; tok != 0 {
			a.scan_.Scan()
			return token{tok, ""}, a.scanErr
		}
	}
	a.lastToken = token{t, a.scan_.TokenText()}
	return a.lastToken, a.scanErr
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

func (a *Assembler) canContinue() bool {
	return a.scanErr == nil
}

func (a *Assembler) assemble() error {
	if a.scanErr != nil {
		return a.scanErr
	}
	for {
		tok, err := a.nextToken()
		if err != nil {
			return err
		}
		switch tok.t {
		case scanner.EOF:
			return nil
		case scanner.Ident:
			if err := a.assembleCommand(tok.s); err != nil {
				return err
			}
		case ';':
			continue
		case '\n':
			continue
		case '.':
			if err := a.assembleLabel(); err != nil {
				return err
			}
		default:
			return a.scanErrorf("unexpected %s", tok)
		}
	}
}

func (a *Assembler) writeByte(u uint8) error {
	if int(a.p) >= len(a.m) {
		return a.scanErrorf("byte write out of range: %d", a.p)
	}
	a.m[a.p] = u
	a.p += 1
	return nil
}

func (a *Assembler) writeBytes(bs []byte) error {
	for _, b := range bs {
		if err := a.writeByte(b); err != nil {
			return err
		}
	}
	return nil
}

// GetLabel returns the value of the given label.
// It is only valid after the assembler has run.
func (a *Assembler) GetLabel(l string) (uint16, bool) {
	v, ok := a.l[l]
	return v, ok
}

type cmdData arg

func (n cmdData) W(a *Assembler) error {
	args, err := a.parseArgs(true)
	if err != nil {
		return err
	}
	for _, arg0 := range args {
		bs, ok, err := arg0.evalAs(a, arg(n), false)
		if err != nil {
			return err
		}
		if !ok {
			return a.scanErrorf("bad data value: %s", arg0)
		}
		if err := a.writeBytes(bs); err != nil {
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

func (ca commandAssembler) W(a *Assembler) error {
	vals, err := a.parseArgs(false)
	if err != nil {
		return err
	}
	found := false
	for argVariant, bs := range ca.args {
		argData, ok, err := a.argsCompatible(vals, argVariant)
		if err != nil {
			return err
		}
		if ok {
			if found {
				log.Fatalf("more than one variant of %s possible: args %#v, found alt variant %s", ca.cmd, vals, argVariant)
			}
			found = true
			if err := a.writeBytes(bs); err != nil {
				return err
			}
			if err := a.writeBytes(argData); err != nil {
				return err
			}
		}
	}
	if !found {
		vs := []string{}
		for _, v := range vals {
			vs = append(vs, fmt.Sprintf("%s", v))
		}
		return a.scanErrorf("no suitable form of %s found that matches %s %s", ca.cmd, ca.cmd, strings.Join(vs, ", "))
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

func (commandOrg) W(a *Assembler) error {
	args, err := a.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return a.scanErrorf("org takes one argument: %d found", len(args))
	}
	n, ok, err := getIntValue(a, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return a.scanErrorf("org wants address, found %s", args[0])
	}
	if n < 16384 || n > 65535 {
		return a.scanErrorf("org %x out of range", n)
	}
	a.p = uint16(n)
	return nil
}

func (a *Assembler) assembleCommand(cmdOrig string) error {
	cmd := strings.ToLower(cmdOrig)
	if f, ok := commandTable[cmd]; ok {
		return f.W(a)
	}
	return a.scanErrorf("unknown command %v", cmdOrig)
}

func (a *Assembler) setLabel(label string) error {
	if a.pass == 1 {
		fass := a.labelAssign[label]
		if a.location() != fass {
			return a.scanErrorf("Label %q redefined. First defined at %s", label, fass)
		}
		return nil
	}
	a.l[label] = a.p
	if a.pass == 0 && a.labelAssign[label] == "" {
		a.labelAssign[label] = a.location()
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
