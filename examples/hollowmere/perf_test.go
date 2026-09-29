package hollowmere

import "testing"

// TestWalkSpeed walks back and forth across the courtyard and reports
// how fast the game runs.
func TestWalkSpeed(t *testing.T) {
	s := newSpectrum(t)
	b := &bot{t: t, s: s, types: s.label(t, "types")}
	for i := 0; i < 100 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	b.dismiss()
	for i := 0; i < 200 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	b.dismiss()
	b.settle()
	// Go north into the courtyard.
	b.walk(func(x, y int) bool { return y < 13 })
	b.walk(func(x, y int) bool { return y < 13 })
	b.settle()
	for i := 0; i < 100; i++ {
		b.frame()
	}
	// No ghost, so the timing is repeatable.
	s.mem[s.label(t, "gstate")] = 0
	s.mem[0xf600+16] = 0
	check, report := s.profile(t)
	s.check = check
	// Walk to the west side, then hold east until blocked; then the same
	// north to south.
	w, d := b.roomDims()
	total, frames := 0, 0
	for _, leg := range []struct {
		start func(x, y int) bool
		key   string
	}{
		{func(x, y int) bool { return x <= 17 }, "P"},
		{func(x, y int) bool { return y <= 17 }, "A"},
	} {
		b.walk(leg.start)
		b.settle()
		s.setKey(leg.key, true)
		still := 0
		for still < 10 {
			x, y := b.m("px"), b.m("py")
			b.frame()
			frames++
			if b.m("px") != x || b.m("py") != y {
				total++
				still = 0
			} else {
				still++
			}
		}
		frames -= 10
		s.setKey(leg.key, false)
	}
	t.Logf("room %d (%dx%d): %d steps in %d frames: %.2f frames per step", b.m("room_id"), w/8, d/8, total, frames, float64(frames)/float64(total))
	report()
}
