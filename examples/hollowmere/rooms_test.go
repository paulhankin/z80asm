package hollowmere

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// enterRoom enters room id directly, arriving as if at the start.
func (s *spectrum) enterRoom(t testing.TB, id int) {
	s.mem[s.label(t, "arrive")] = byte(s.konst(t, "ARRIVE_START"))
	s.cpu.A = byte(id)
	s.call(t, s.label(t, "enter_room"))
}

func TestAllRooms(t *testing.T) {
	s := newSpectrum(t)
	s.call(t, s.label(t, "make_revtab"))
	s.call(t, s.label(t, "make_bandtab"))
	n := int(s.konst(t, "NROOMS"))
	maxObj, maxBG := 0, 0
	type sheet struct {
		img *image.RGBA
		n   int
	}
	sheets := map[int]*sheet{}
	for id := 0; id < n; id++ {
		s.enterRoom(t, id)
		nobj := int(s.mem[s.label(t, "nobj")])
		nbg := int(s.mem[s.label(t, "nbg")])
		if nobj > maxObj {
			maxObj = nobj
		}
		if nbg > maxBG {
			maxBG = nbg
		}
		if nbg >= int(s.konst(t, "MAXBG")) {
			t.Errorf("room %d: too many background pieces", id)
		}
		if os.Getenv("HM_SHOTS") == "" {
			continue
		}
		roomAddr := s.word(s.label(t, "room_table") + uint16(2*id))
		floor := int(s.mem[roomAddr+8] >> 6)
		sh := sheets[floor]
		if sh == nil {
			sh = &sheet{img: image.NewRGBA(image.Rect(0, 0, 5*272, 8*208))}
			sheets[floor] = sh
		}
		shot := s.screenshot()
		x, y := (sh.n%5)*272, (sh.n/5)*208
		// Crop the border (scale 2): take the screen at half size.
		for yy := 0; yy < 192+16; yy++ {
			for xx := 0; xx < 256+16; xx++ {
				sh.img.Set(x+xx, y+yy, shot.At((xx+8)*2, (yy+8)*2))
			}
		}
		sh.n++
	}
	t.Logf("max objects %d, max background pieces %d", maxObj, maxBG)
	for f, sh := range sheets {
		out := image.NewRGBA(image.Rect(0, 0, 5*272, ((sh.n+4)/5)*208))
		draw.Draw(out, out.Bounds(), sh.img, image.Point{}, draw.Src)
		fn, _ := os.Create(filepath.Join(os.Getenv("HM_SHOTS"), "floor"+string(rune('0'+f))+".png"))
		png.Encode(fn, out)
		fn.Close()
	}
}

func (s *spectrum) word(a uint16) uint16 {
	return uint16(s.mem[a]) | uint16(s.mem[a+1])<<8
}
