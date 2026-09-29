package hollowmere

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"testing"
)

// where names the routine containing address pc, for debugging.
func (s *spectrum) where(t testing.TB, pc uint16) string {
	f, err := os.Open("hollowmere.z80")
	if err != nil {
		return "?"
	}
	defer f.Close()
	re := regexp.MustCompile(`^([a-z_][a-z0-9_]*):`)
	best, bestA := "?", uint16(0)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := re.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		a, ok := s.asm.GetLabel("", m[1])
		if ok && a <= pc && a >= bestA {
			best, bestA = m[1], a
		}
	}
	return fmt.Sprintf("%s+%d", best, pc-bestA)
}

// history reports the routines recently run, oldest first.
func (s *spectrum) history(t testing.TB) {
	n := len(s.ring)
	last := ""
	for i := 0; i < n; i++ {
		pc := s.ring[(s.ringi+i)%n]
		w := s.where(t, pc)
		name := w
		for k := 0; k < len(w); k++ {
			if w[k] == '+' {
				name = w[:k]
			}
		}
		if name != last {
			t.Logf("  %s (%s)", name, w)
			last = name
		}
	}
}

type labelAddr struct {
	name string
	a    uint16
}

// labels returns the major labels of the program, sorted by address.
func (s *spectrum) labels(t testing.TB) []labelAddr {
	f, err := os.Open("hollowmere.z80")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	re := regexp.MustCompile(`^([a-z_][a-z0-9_]*):`)
	var out []labelAddr
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := re.FindStringSubmatch(sc.Text()); m != nil {
			if a, ok := s.asm.GetLabel("", m[1]); ok {
				out = append(out, labelAddr{m[1], a})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].a < out[j].a })
	return out
}

// profile returns a function to call before each instruction that
// counts T-states by routine, and a function to report them.
func (s *spectrum) profile(t testing.TB) (func(), func()) {
	ls := s.labels(t)
	counts := map[string]int{}
	last := ""
	lastT := 0
	check := func() {
		pc := s.cpu.PC()
		i := sort.Search(len(ls), func(i int) bool { return ls[i].a > pc }) - 1
		name := "?"
		if i >= 0 {
			name = ls[i].name
		}
		if last != "" {
			d := s.cpu.Tstates - lastT
			if d > 0 {
				counts[last] += d
			}
		}
		last, lastT = name, s.cpu.Tstates
	}
	report := func() {
		type kv struct {
			k string
			v int
		}
		var kvs []kv
		total := 0
		for k, v := range counts {
			kvs = append(kvs, kv{k, v})
			total += v
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].v > kvs[j].v })
		for i, x := range kvs {
			if i < 20 {
				t.Logf("%-16s %5.1f%%", x.k, 100*float64(x.v)/float64(total))
			}
		}
	}
	return check, report
}
