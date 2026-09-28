package backgammon

// A minimal headless 48K ZX Spectrum, good enough to run and test the
// game: RAM, keyboard ports, 50Hz interrupts and screen rendering. There
// is no ROM (the game doesn't use it).

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/paulhankin/z80asm"
	"github.com/paulhankin/z80asm/z80test/z80"
)

const tstatesPerFrame = 69888

type spectrum struct {
	cpu   *z80.Z80
	mem   [65536]byte
	keys  [8]byte // keyboard half-rows, active low
	asm   *z80asm.Assembler
	frame int
}

// Memory accessor. Every access costs time so that Tstates advances.
func (s *spectrum) ReadByte(a uint16) byte             { s.cpu.Tstates += 3; return s.mem[a] }
func (s *spectrum) ReadByteInternal(a uint16) byte     { return s.mem[a] }
func (s *spectrum) WriteByte(a uint16, v byte)         { s.cpu.Tstates += 3; s.WriteByteInternal(a, v) }
func (s *spectrum) ContendRead(a uint16, t int)        { s.cpu.Tstates += t }
func (s *spectrum) ContendReadNoMreq(a uint16, t int)  { s.cpu.Tstates += t }
func (s *spectrum) ContendWriteNoMreq(a uint16, t int) { s.cpu.Tstates += t }
func (s *spectrum) ContendReadNoMreq_loop(a uint16, t int, n uint) {
	s.cpu.Tstates += t * int(n)
}
func (s *spectrum) ContendWriteNoMreq_loop(a uint16, t int, n uint) {
	s.cpu.Tstates += t * int(n)
}
func (s *spectrum) Read(a uint16) byte                   { return s.mem[a] }
func (s *spectrum) Write(a uint16, v byte, protect bool) { s.WriteByteInternal(a, v) }
func (s *spectrum) Data() []byte                         { return s.mem[:] }
func (s *spectrum) WriteByteInternal(a uint16, v byte) {
	if a >= 0x4000 {
		s.mem[a] = v
	}
}

// Port accessor: just the keyboard.
func (s *spectrum) ReadPort(a uint16) byte {
	if a&1 != 0 {
		return 0xff
	}
	r := byte(0x1f)
	for row := 0; row < 8; row++ {
		if a&(0x100<<row) == 0 {
			r &= s.keys[row]
		}
	}
	return r | 0xe0
}
func (s *spectrum) WritePort(a uint16, b byte)                       {}
func (s *spectrum) ReadPortInternal(a uint16, contend bool) byte     { return s.ReadPort(a) }
func (s *spectrum) WritePortInternal(a uint16, b byte, contend bool) {}
func (s *spectrum) ContendPortPreio(a uint16)                        { s.cpu.Tstates++ }
func (s *spectrum) ContendPortPostio(a uint16)                       { s.cpu.Tstates += 3 }

var keyMap = map[string][2]int{
	"SHIFT": {0, 0}, "Z": {0, 1}, "X": {0, 2}, "C": {0, 3}, "V": {0, 4},
	"A": {1, 0}, "S": {1, 1}, "D": {1, 2}, "F": {1, 3}, "G": {1, 4},
	"Q": {2, 0}, "W": {2, 1}, "E": {2, 2}, "R": {2, 3}, "T": {2, 4},
	"1": {3, 0}, "2": {3, 1}, "3": {3, 2}, "4": {3, 3}, "5": {3, 4},
	"0": {4, 0}, "9": {4, 1}, "8": {4, 2}, "7": {4, 3}, "6": {4, 4},
	"P": {5, 0}, "O": {5, 1}, "I": {5, 2}, "U": {5, 3}, "Y": {5, 4},
	"ENTER": {6, 0}, "L": {6, 1}, "K": {6, 2}, "J": {6, 3}, "H": {6, 4},
	"SPACE": {7, 0}, "SYM": {7, 1}, "M": {7, 2}, "N": {7, 3}, "B": {7, 4},
}

func (s *spectrum) setKey(name string, down bool) {
	k, ok := keyMap[name]
	if !ok {
		panic("unknown key " + name)
	}
	if down {
		s.keys[k[0]] &^= 1 << k[1]
	} else {
		s.keys[k[0]] |= 1 << k[1]
	}
}

// newSpectrum assembles the game and loads it into a new machine.
func newSpectrum(t testing.TB) *spectrum {
	asm, err := z80asm.NewAssembler()
	if err != nil {
		t.Fatal(err)
	}
	if err := asm.AssembleFile("backgammon.z80"); err != nil {
		t.Fatalf("assembly failed: %v", err)
	}
	s := &spectrum{asm: asm}
	for i := range s.keys {
		s.keys[i] = 0x1f
	}
	copy(s.mem[:], asm.RAM())
	s.cpu = z80.NewZ80(s, s, nil)
	s.cpu.SetPC(s.label(t, "main"))
	return s
}

func (s *spectrum) label(t testing.TB, name string) uint16 {
	v, ok := s.asm.GetLabel("", name)
	if !ok {
		t.Fatalf("label %q not found", name)
	}
	return v
}

func (s *spectrum) konst(t testing.TB, name string) uint16 {
	v, ok, err := s.asm.GetConst(name)
	if err != nil || !ok {
		t.Fatalf("const %q not found: %v", name, err)
	}
	return uint16(v)
}

// runFrame runs one 50th of a second, then raises an interrupt.
func (s *spectrum) runFrame() {
	for s.cpu.Tstates < tstatesPerFrame {
		if s.cpu.Halted {
			s.cpu.Tstates = tstatesPerFrame
			break
		}
		s.cpu.DoOpcode()
	}
	s.cpu.Tstates -= tstatesPerFrame
	s.cpu.Interrupt()
	s.frame++
}

func (s *spectrum) runFrames(n int) {
	for i := 0; i < n; i++ {
		s.runFrame()
	}
}

// call calls the routine at addr with interrupts disabled, returning the
// number of T-states it took.
func (s *spectrum) call(t testing.TB, addr uint16) int {
	const magic = 0x0000
	s.cpu.IFF1, s.cpu.IFF2 = 0, 0
	s.cpu.Halted = false
	sp := uint16(0xfd00)
	sp -= 2
	s.mem[sp] = byte(magic & 0xff)
	s.mem[sp+1] = byte(magic >> 8)
	s.cpu.SetSP(sp)
	s.cpu.SetPC(addr)
	s.cpu.Tstates = 0
	total := 0
	for s.cpu.PC() != magic {
		s.cpu.DoOpcode()
		if s.cpu.Halted {
			t.Fatalf("halted while calling %04x", addr)
		}
		if s.cpu.Tstates > 1<<30 {
			t.Fatalf("call to %04x took too long", addr)
		}
	}
	total += s.cpu.Tstates
	return total
}

var palette = [16]color.RGBA{
	{0, 0, 0, 255}, {0, 0, 0xd7, 255}, {0xd7, 0, 0, 255}, {0xd7, 0, 0xd7, 255},
	{0, 0xd7, 0, 255}, {0, 0xd7, 0xd7, 255}, {0xd7, 0xd7, 0, 255}, {0xd7, 0xd7, 0xd7, 255},
	{0, 0, 0, 255}, {0, 0, 0xff, 255}, {0xff, 0, 0, 255}, {0xff, 0, 0xff, 255},
	{0, 0xff, 0, 255}, {0, 0xff, 0xff, 255}, {0xff, 0xff, 0, 255}, {0xff, 0xff, 0xff, 255},
}

// screenshot renders the screen (at 2x scale, with a border) as an image.
func (s *spectrum) screenshot() *image.RGBA {
	const border, scale = 16, 2
	img := image.NewRGBA(image.Rect(0, 0, (256+2*border)*scale, (192+2*border)*scale))
	for i := range img.Pix {
		img.Pix[i] = 0
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	for y := 0; y < 192; y++ {
		for x := 0; x < 256; x++ {
			addr := 0x4000 | (y&0xc0)<<5 | (y&7)<<8 | (y&0x38)<<2 | x>>3
			attr := s.mem[0x5800+(y/8)*32+x/8]
			bit := s.mem[addr]&(0x80>>(x&7)) != 0
			if attr&0x80 != 0 && (s.frame/16)%2 == 1 {
				bit = !bit
			}
			bright := int(attr>>6&1) * 8
			c := palette[int(attr>>3&7)+bright]
			if bit {
				c = palette[int(attr&7)+bright]
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetRGBA((x+border)*scale+dx, (y+border)*scale+dy, c)
				}
			}
		}
	}
	return img
}

func (s *spectrum) savePNG(t testing.TB, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, s.screenshot()); err != nil {
		t.Fatal(err)
	}
}

// screenText reads a row of text from the screen by matching the font.
func (s *spectrum) screenText(t testing.TB, row, col, n int) string {
	font := s.label(t, "font")
	r := []byte{}
	for c := col; c < col+n; c++ {
		var cell [8]byte
		for y := 0; y < 8; y++ {
			py := row*8 + y
			addr := 0x4000 | (py&0xc0)<<5 | (py&7)<<8 | (py&0x38)<<2 | c
			cell[y] = s.mem[addr]
		}
		ch := byte('~')
		for code := 32; code <= 90; code++ {
			match := true
			for y := 0; y < 8; y++ {
				if s.mem[int(font)+(code-32)*8+y] != cell[y] {
					match = false
					break
				}
			}
			if match {
				ch = byte(code)
				break
			}
		}
		r = append(r, ch)
	}
	return string(r)
}

func (s *spectrum) String() string {
	return fmt.Sprintf("frame %d pc=%04x", s.frame, s.cpu.PC())
}
