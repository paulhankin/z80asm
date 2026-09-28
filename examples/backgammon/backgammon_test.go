package backgammon

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------
// A straightforward reference implementation of the rules of moving.
// ---------------------------------------------------------------------

// pos is a position from the point of view of the side to move.
// Index 0 = borne off, 1..24 = points, 25 = bar.
type pos struct {
	me, opp [26]int
}

type step struct{ from, die int }

func (p *pos) legal(from, die int) bool {
	if p.me[from] == 0 {
		return false
	}
	if p.me[25] > 0 && from != 25 {
		return false
	}
	to := from - die
	if to >= 1 {
		return p.opp[25-to] < 2
	}
	for i := 7; i <= 25; i++ {
		if p.me[i] > 0 {
			return false
		}
	}
	if to == 0 {
		return true
	}
	for i := from + 1; i <= 6; i++ {
		if p.me[i] > 0 {
			return false
		}
	}
	return true
}

func (p pos) move(from, die int) pos {
	p.me[from]--
	to := from - die
	if to <= 0 {
		p.me[0]++
		return p
	}
	p.me[to]++
	if p.opp[25-to] == 1 {
		p.opp[25-to] = 0
		p.opp[25]++
	}
	return p
}

// sequences returns every complete sequence of moves (a sequence is
// complete when no further move is possible).
func sequences(p pos, dice []int) [][]step {
	var out [][]step
	var rec func(p pos, dice []int, prefix []step)
	rec = func(p pos, dice []int, prefix []step) {
		found := false
		tried := map[int]bool{}
		for i, d := range dice {
			if tried[d] {
				continue
			}
			tried[d] = true
			rest := append(append([]int{}, dice[:i]...), dice[i+1:]...)
			for from := 25; from >= 1; from-- {
				if p.legal(from, d) {
					found = true
					rec(p.move(from, d), rest, append(append([]step{}, prefix...), step{from, d}))
				}
			}
		}
		if !found {
			out = append(out, prefix)
		}
	}
	rec(p, dice, nil)
	return out
}

// validSequences returns the sequences that are allowed by the rules:
// as many dice as possible must be played, and if only one die of a
// non-double can be played, it must be the larger if possible.
func validSequences(p pos, dice []int) [][]step {
	all := sequences(p, dice)
	maxlen := 0
	for _, s := range all {
		if len(s) > maxlen {
			maxlen = len(s)
		}
	}
	var r [][]step
	for _, s := range all {
		if len(s) == maxlen {
			r = append(r, s)
		}
	}
	if maxlen == 1 && len(dice) == 2 && dice[0] != dice[1] {
		hi := 0
		for _, s := range r {
			if s[0].die > hi {
				hi = s[0].die
			}
		}
		var r2 [][]step
		for _, s := range r {
			if s[0].die == hi {
				r2 = append(r2, s)
			}
		}
		r = r2
	}
	return r
}

func startPos() pos {
	var p pos
	for _, b := range []*[26]int{&p.me, &p.opp} {
		b[24], b[13], b[8], b[6] = 2, 5, 3, 5
	}
	return p
}

func randomPos(r *rand.Rand) pos {
	var p pos
	style := r.Intn(4)
	pick := func() int {
		switch style {
		case 0: // anywhere, occasionally on the bar or off
			x := r.Intn(30)
			if x >= 26 {
				return r.Intn(24) + 1
			}
			return x
		case 1: // bearing off
			if r.Intn(8) == 0 {
				return r.Intn(12) + 1
			}
			return r.Intn(7)
		case 2: // mostly home, some on the bar
			if r.Intn(10) == 0 {
				return 25
			}
			return r.Intn(9) + 1
		default:
			return r.Intn(24) + 1
		}
	}
	for side := 0; side < 2; side++ {
		for n := 0; n < 15; n++ {
			for {
				loc := pick()
				mine, theirs := &p.me, &p.opp
				if side == 1 {
					mine, theirs = &p.opp, &p.me
				}
				if loc >= 1 && loc <= 24 && theirs[25-loc] > 0 {
					continue
				}
				mine[loc]++
				break
			}
		}
	}
	return p
}

func randomDice(r *rand.Rand) []int {
	a, b := r.Intn(6)+1, r.Intn(6)+1
	if a == b {
		return []int{a, a, a, a}
	}
	return []int{a, b}
}

// ---------------------------------------------------------------------
// Helpers to poke positions into the Z80 program.
// ---------------------------------------------------------------------

// setup puts position p into the game's memory with the given side to
// move (0 = human, 1 = computer), and the given dice.
func (s *spectrum) setup(t testing.TB, p pos, side int, dice []int) {
	me, opp := s.konst(t, "HUMB"), s.konst(t, "CPUB")
	if side == 1 {
		me, opp = opp, me
	}
	for i := 0; i < 26; i++ {
		s.mem[int(me)+i] = byte(p.me[i])
		s.mem[int(opp)+i] = byte(p.opp[i])
	}
	dv, du := s.label(t, "dv"), s.label(t, "dused")
	for i := 0; i < 4; i++ {
		s.mem[dv+uint16(i)] = 0
		s.mem[du+uint16(i)] = 0
	}
	for i, d := range dice {
		s.mem[dv+uint16(i)] = byte(d)
	}
	s.mem[s.label(t, "dn")] = byte(len(dice))
	s.mem[s.label(t, "isdbl")] = 0
	if len(dice) == 4 {
		s.mem[s.label(t, "isdbl")] = 1
	}
	if side == 0 {
		s.call(t, s.label(t, "side_hum"))
	} else {
		s.call(t, s.label(t, "side_cpu"))
	}
}

func (s *spectrum) byteAt(t testing.TB, label string) int {
	return int(s.mem[s.label(t, label)])
}

// humlegal calls the game's humlegal routine.
func (s *spectrum) humlegal(t testing.TB, from, k int) bool {
	s.cpu.B = byte(from)
	s.cpu.A = byte(k)
	s.call(t, s.label(t, "humlegal"))
	return s.cpu.F&1 == 0
}

func fmtSeq(seq []step) string {
	r := ""
	for _, s := range seq {
		r += fmt.Sprintf("%d/%d ", s.from, s.from-s.die)
	}
	return r
}

// ---------------------------------------------------------------------
// Tests.
// ---------------------------------------------------------------------

// TestRulesEngine compares the Z80 rules engine with the reference
// implementation on lots of random positions and dice.
func TestRulesEngine(t *testing.T) {
	s := newSpectrum(t)
	r := rand.New(rand.NewSource(1))
	positions := 3000
	if testing.Short() {
		positions = 300
	}
	for n := 0; n < positions; n++ {
		p := randomPos(r)
		if n == 0 {
			p = startPos()
		}
		dice := randomDice(r)
		valid := validSequences(p, dice)
		want := len(valid[0])

		s.setup(t, p, 0, dice)
		s.call(t, s.label(t, "turn_setup"))
		if got := s.byteAt(t, "need"); got != want {
			t.Fatalf("pos %d %+v dice %v: need=%d, want %d", n, p, dice, got, want)
		}

		// First moves.
		firsts := map[step]bool{}
		for _, seq := range valid {
			if len(seq) > 0 {
				firsts[seq[0]] = true
			}
		}
		for from := 1; from <= 25; from++ {
			for k := range dice {
				got := s.humlegal(t, from, k)
				if want := firsts[step{from, dice[k]}]; got != want {
					t.Fatalf("pos %d %+v dice %v: humlegal(%d, die %d) = %v, want %v", n, p, dice, from, dice[k], got, want)
				}
			}
		}

		// Second moves, after a random legal first move.
		if want >= 2 {
			first := valid[r.Intn(len(valid))][0]
			seconds := map[step]bool{}
			for _, seq := range valid {
				if seq[0] == first {
					seconds[seq[1]] = true
				}
			}
			p2 := p.move(first.from, first.die)
			s.setup(t, p2, 0, dice)
			// Mark the die used and keep need/forcehi from the start of the turn.
			for k, d := range dice {
				if d == first.die {
					s.mem[s.label(t, "dused")+uint16(k)] = 1
					break
				}
			}
			s.mem[s.label(t, "need")] = byte(want - 1)
			s.mem[s.label(t, "forcehi")] = 0
			for from := 1; from <= 25; from++ {
				for k := range dice {
					got := s.humlegal(t, from, k)
					used := s.mem[s.label(t, "dused")+uint16(k)] != 0
					want := !used && seconds[step{from, dice[k]}]
					if got != want {
						t.Fatalf("pos %d %+v dice %v after %v: humlegal(%d, die %d) = %v, want %v", n, p, dice, first, from, dice[k], got, want)
					}
				}
			}
		}
	}
}

// TestAI checks that the computer always chooses a legal, complete move,
// and measures how long it takes to think.
func TestAI(t *testing.T) {
	s := newSpectrum(t)
	r := rand.New(rand.NewSource(2))
	positions := 1000
	if testing.Short() {
		positions = 100
	}
	worst := 0
	total := 0
	for n := 0; n < positions; n++ {
		p := randomPos(r)
		if n%4 == 0 {
			p = startPos()
		}
		dice := randomDice(r)
		valid := validSequences(p, dice)
		want := len(valid[0])

		s.setup(t, p, 1, dice)
		s.call(t, s.label(t, "turn_setup"))
		if want == 0 {
			continue
		}
		ts := s.call(t, s.label(t, "ai_think"))
		total += ts
		if ts > worst {
			worst = ts
		}
		bl := s.byteAt(t, "bestlen")
		var seq []step
		for i := 0; i < bl; i++ {
			seq = append(seq, step{
				from: int(s.mem[s.label(t, "bestfrom")+uint16(i)]),
				die:  int(s.mem[s.label(t, "bestdie")+uint16(i)]),
			})
		}
		ok := false
		for _, v := range valid {
			if fmt.Sprint(v) == fmt.Sprint(seq) {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("pos %d %+v dice %v: AI chose %v which is not a valid sequence (e.g. %v)", n, p, dice, seq, valid[0])
		}
		// The position in memory must be unchanged by thinking.
		cpu := s.konst(t, "CPUB")
		for i := 0; i < 26; i++ {
			if int(s.mem[int(cpu)+i]) != p.me[i] {
				t.Fatalf("pos %d: board changed by ai_think", n)
			}
		}
	}
	t.Logf("AI thinking time: worst %.2fs, average %.2fs", float64(worst)/3.5e6, float64(total)/3.5e6/float64(positions))
	if float64(worst)/3.5e6 > 20 {
		t.Errorf("AI took %.1fs to think in the worst case", float64(worst)/3.5e6)
	}
}

// TestAIOpenings prints the computer's choice for each opening roll, as a
// sanity check of the evaluation function (run with -v).
func TestAIOpenings(t *testing.T) {
	s := newSpectrum(t)
	for a := 1; a <= 6; a++ {
		for b := 1; b <= a; b++ {
			dice := []int{a, b}
			if a == b {
				dice = []int{a, a, a, a}
			}
			s.setup(t, startPos(), 1, dice)
			s.call(t, s.label(t, "turn_setup"))
			s.call(t, s.label(t, "ai_think"))
			var seq []step
			for i := 0; i < s.byteAt(t, "bestlen"); i++ {
				seq = append(seq, step{
					from: int(s.mem[s.label(t, "bestfrom")+uint16(i)]),
					die:  int(s.mem[s.label(t, "bestdie")+uint16(i)]),
				})
			}
			t.Logf("%d-%d: %s", a, b, fmtSeq(seq))
		}
	}
}

// checkBoard verifies that both sides have 15 checkers and no point is
// occupied by both.
func (s *spectrum) checkBoard(t testing.TB) error {
	hum, cpu := s.konst(t, "HUMB"), s.konst(t, "CPUB")
	nh, nc := 0, 0
	for i := 0; i < 26; i++ {
		nh += int(s.mem[int(hum)+i])
		nc += int(s.mem[int(cpu)+i])
	}
	if nh != 15 || nc != 15 {
		return fmt.Errorf("checker counts %d, %d", nh, nc)
	}
	for i := 1; i <= 24; i++ {
		if s.mem[int(hum)+i] > 0 && s.mem[int(cpu)+25-i] > 0 {
			return fmt.Errorf("point %d has both colours", i)
		}
	}
	return nil
}

// TestPlayGames runs the whole program, with a "player" pressing keys
// at random, until several games have been completed.
func TestPlayGames(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	s := newSpectrum(t)
	r := rand.New(rand.NewSource(3))
	shots := os.Getenv("BG_SHOTS")
	keys := []string{"SPACE", "SPACE", "SPACE", "SPACE", "P", "P", "O", "8", "X", "U"}
	scoreHum, scoreCPU := s.label(t, "score_hum"), s.label(t, "score_cpu")
	games := 0
	lastScore := 0
	badChecks := 0
	started := false
	for s.frame < 400000 && games < 4 {
		// Press a key for 3 frames, then release for a few frames.
		key := keys[r.Intn(len(keys))]
		s.setKey(key, true)
		s.runFrames(3)
		s.setKey(key, false)
		s.runFrames(3 + r.Intn(5))
		if s.cpu.PC() < 0x8000 && s.cpu.PC() != 0xfdfd {
			t.Fatalf("%v: PC out of program", s)
		}
		if !started {
			// Wait for the board to appear.
			if s.mem[s.konst(t, "HUMB")+6] == 5 {
				started = true
			}
			continue
		}
		if err := s.checkBoard(t); err != nil {
			badChecks++
			if badChecks > 3 {
				t.Fatalf("%v: %v", s, err)
			}
		} else {
			badChecks = 0
		}
		score := int(s.mem[scoreHum]) + int(s.mem[scoreCPU])
		if score != lastScore {
			games++
			t.Logf("%v: game %d over: human %d, computer %d", s, games, s.mem[scoreHum], s.mem[scoreCPU])
			if shots != "" {
				s.savePNG(t, filepath.Join(shots, fmt.Sprintf("gameover%d.png", games)))
			}
			lastScore = score
		}
	}
	if games < 4 {
		t.Fatalf("%v: only %d games completed", s, games)
	}
}

// TestScreenshots writes some screenshots to $BG_SHOTS, if set.
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("BG_SHOTS")
	if dir == "" {
		t.Skip("BG_SHOTS not set")
	}
	s := newSpectrum(t)
	press := func(k string) {
		s.setKey(k, true)
		s.runFrames(3)
		s.setKey(k, false)
		s.runFrames(3)
	}
	s.runFrames(20)
	s.savePNG(t, filepath.Join(dir, "title.png"))
	press("SPACE")
	s.runFrames(30)
	s.savePNG(t, filepath.Join(dir, "opening.png"))
	// Play until it's the human's turn to choose a move.
	for i := 0; i < 20000; i++ {
		s.runFrame()
		if s.screenText(t, 11, 1, 9) == "YOUR MOVE" || s.screenText(t, 11, 1, 9) == "YOUR TURN" {
			break
		}
	}
	s.runFrames(5)
	s.savePNG(t, filepath.Join(dir, "turn1.png"))
	for i := 0; i < 6; i++ {
		t.Logf("message: %q / %q", s.screenText(t, 11, 1, 12), s.screenText(t, 12, 1, 12))
		press("SPACE")
		s.runFrames(20)
		s.savePNG(t, filepath.Join(dir, fmt.Sprintf("step%d.png", i)))
	}
	for i := 0; i < 20000; i++ {
		s.runFrame()
		if s.screenText(t, 11, 1, 9) == "THINKING." {
			break
		}
	}
	s.savePNG(t, filepath.Join(dir, "thinking.png"))
	s.runFrames(150)
	s.savePNG(t, filepath.Join(dir, "cpumove.png"))
}
