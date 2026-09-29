package hollowmere

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLayout checks that the program doesn't overlap the buffers above it.
func TestLayout(t *testing.T) {
	s := newSpectrum(t)
	end := s.label(t, "data_end")
	cache := s.konst(t, "PCACHE")
	t.Logf("program 0x5b00-%04x, %d bytes free", end, int(cache)-int(end))
	if end > cache {
		t.Errorf("program ends at %04x, overlapping the buffers at %04x", end, cache)
	}
	if top := cache + 12*s.konst(t, "PSLOT"); top > s.konst(t, "SCRATCH") {
		t.Errorf("player cache ends at %04x, overlapping the scratch buffer", top)
	}
}

func shot(t *testing.T, s *spectrum, name string) {
	dir := os.Getenv("HM_SHOTS")
	if dir == "" {
		return
	}
	s.savePNG(t, filepath.Join(dir, name+".png"))
}

// TestSnapshot checks that hollowmere.sna is up to date with the source.
func TestSnapshot(t *testing.T) {
	s := newSpectrum(t)
	sna, err := os.ReadFile("hollowmere.sna")
	if err != nil {
		t.Fatal(err)
	}
	if len(sna) != 27+49152 {
		t.Fatalf("hollowmere.sna is %d bytes, want %d", len(sna), 27+49152)
	}
	ram := s.asm.RAM()
	sp := int(sna[23]) | int(sna[24])<<8
	for a := 0x4000; a < 0x10000; a++ {
		if a == sp || a == sp+1 {
			continue // the snapshot's PC is pushed here
		}
		if sna[27+a-0x4000] != ram[a] {
			t.Fatalf("hollowmere.sna differs from the source at %04x: rebuild it with go run ./cmd/z80asm examples/hollowmere/hollowmere.z80", a)
		}
	}
	if pc := uint16(sna[27+sp-0x4000]) | uint16(sna[28+sp-0x4000])<<8; pc != s.label(t, "main") {
		t.Errorf("hollowmere.sna starts at %04x, want %04x", pc, s.label(t, "main"))
	}
}
