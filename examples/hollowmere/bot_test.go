package hollowmere

import (
	"fmt"
	"testing"
)

// A bot that plays the whole game through the keyboard. It plans using
// what's in the game's memory (the room descriptions, the objects in the
// current room and the rectangles the player can walk in), but all the
// movement is done by the game itself in response to key presses.

const (
	fSolid = 0x01
	fItem  = 0x04
	fUp    = 0x10
	fDown  = 0x20
	fGate  = 0x40
	fBell  = 0x80
)

type roomInfo struct {
	id, wt, dt int
	floor      int
	exits      [4]int // N E S W, -1 for none
	locks      [4]int // lock on each doorway
	stairs     []int  // rooms reached by stairs
	items      []int
	bell       bool
}

type bot struct {
	t                     *testing.T
	s                     *spectrum
	rooms                 []roomInfo
	types                 uint16
	log                   []string
	ghostSeen             bool
	memShot               bool
	walkFrames, walkSteps int
	hist                  map[int]int
}

func (b *bot) m(label string) byte { return b.s.mem[b.s.label(b.t, label)] }

func (b *bot) typeFlags(typ int) byte {
	return b.s.mem[b.types+uint16(11*(typ-1))]
}

func (b *bot) readRooms() {
	n := int(b.s.konst(b.t, "NROOMS"))
	tItem := int(b.s.konst(b.t, "T_ITEM"))
	for id := 0; id < n; id++ {
		a := b.s.word(b.s.label(b.t, "room_table") + uint16(2*id))
		r := roomInfo{id: id, wt: int(b.s.mem[a] >> 4), dt: int(b.s.mem[a] & 15)}
		for d := 0; d < 4; d++ {
			r.exits[d] = int(b.s.mem[a+4+uint16(d)])
			if r.exits[d] == 0xff {
				r.exits[d] = -1
			}
		}
		r.floor = int(b.s.mem[a+8] >> 6)
		p := a + 9 + uint16(r.wt+r.dt)
		for b.s.mem[p] != 0 {
			typ, pos, extra := int(b.s.mem[p]), int(b.s.mem[p+1]), int(b.s.mem[p+2])
			p += 3
			tx, ty := pos&15, pos>>4
			fl := b.typeFlags(typ)
			switch {
			case typ == tItem:
				r.items = append(r.items, extra)
			case fl&(fUp|fDown) != 0:
				r.stairs = append(r.stairs, extra)
			case fl&fGate != 0:
				d := 0
				switch {
				case ty == 0 && tx == r.wt/2 && r.exits[0] >= 0:
					d = 0
				case ty == r.dt-1 && tx == r.wt/2 && r.exits[2] >= 0:
					d = 2
				case tx == 0 && ty == r.dt/2 && r.exits[3] >= 0:
					d = 3
				case tx == r.wt-1 && ty == r.dt/2 && r.exits[1] >= 0:
					d = 1
				default:
					b.t.Fatalf("room %d: gate at %d,%d not on a doorway", id, tx, ty)
				}
				r.locks[d] = extra
			case fl&fBell != 0:
				r.bell = true
			}
		}
		b.rooms = append(b.rooms, r)
	}
}

func (b *bot) got(item int) bool { return b.s.mem[b.s.label(b.t, "items_got")+uint16(item)] != 0 }

func (b *bot) canOpen(lock int) bool {
	if lock == 0 || b.s.mem[b.s.label(b.t, "locks_open")+uint16(lock)] != 0 {
		return true
	}
	if lock == 4 {
		return b.m("memories") == 8
	}
	return b.got(7 + lock)
}

// step is one move between rooms: through doorway dir (0-3), or up or
// down stairs to a room (dir 4).
type step struct{ dir, to int }

// route finds the shortest way from room `from` to a room satisfying goal.
func (b *bot) route(from int, goal func(r *roomInfo) bool) []step {
	type node struct {
		prev int
		st   step
	}
	seen := map[int]node{from: {-1, step{}}}
	q := []int{from}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		if goal(&b.rooms[c]) {
			var path []step
			for c != from {
				n := seen[c]
				path = append([]step{n.st}, path...)
				c = n.prev
			}
			return path
		}
		r := &b.rooms[c]
		var next []step
		for d := 0; d < 4; d++ {
			if r.exits[d] >= 0 && b.canOpen(r.locks[d]) {
				next = append(next, step{d, r.exits[d]})
			}
		}
		for _, to := range r.stairs {
			next = append(next, step{4, to})
		}
		for _, st := range next {
			if _, ok := seen[st.to]; !ok {
				seen[st.to] = node{c, st}
				q = append(q, st.to)
			}
		}
	}
	return nil
}

type obj struct {
	typ, flags, extra  int
	bx0, by0, bx1, by1 int
}

func (b *bot) objects() []obj {
	n := int(b.m("nobj"))
	var out []obj
	for i := 2; i < n; i++ {
		a := uint16(0xf600 + 16*i)
		m := b.s.mem[a : a+16]
		if m[0] == 0 {
			continue
		}
		out = append(out, obj{int(m[0]), int(m[1]), int(m[2]), int(m[12]), int(m[13]), int(m[14]), int(m[15])})
	}
	return out
}

func (b *bot) walkable(x, y int, objs []obj) bool {
	if x < 0 || y < 0 {
		return false
	}
	rects := b.s.label(b.t, "walk_rects")
	ok := false
	for i := 0; i < int(b.m("nwalk")); i++ {
		r := b.s.mem[rects+uint16(4*i) : rects+uint16(4*i+4)]
		if x >= int(r[0]) && x+6 <= int(r[1]) && y >= int(r[2]) && y+6 <= int(r[3]) {
			ok = true
		}
	}
	if !ok {
		return false
	}
	for _, o := range objs {
		if o.flags&fSolid == 0 {
			continue
		}
		if o.flags&fGate != 0 && b.canOpen(o.extra) {
			continue
		}
		if x < o.bx1 && x+6 > o.bx0 && y < o.by1 && y+6 > o.by0 {
			return false
		}
	}
	return true
}

type pt struct{ x, y int }

// path finds the shortest walk from the player to a position satisfying
// goal.
func (b *bot) path(goal func(x, y int) bool) []pt {
	objs := b.objects()
	start := pt{int(b.m("px")), int(b.m("py"))}
	prev := map[pt]pt{start: start}
	q := []pt{start}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		if goal(c.x, c.y) {
			var p []pt
			for c != start {
				p = append([]pt{c}, p...)
				c = prev[c]
			}
			return p
		}
		for _, d := range []pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := pt{c.x + d.x, c.y + d.y}
			if _, ok := prev[n]; ok {
				continue
			}
			if !b.walkable(n.x, n.y, objs) {
				continue
			}
			prev[n] = c
			q = append(q, n)
		}
	}
	return nil
}

// settle runs frames until the game is waiting for the next frame in its
// main loop (or for a key).
func (b *bot) settle() {
	for i := 0; i < 1000; i++ {
		if b.m("busy") == 0 || b.m("waiting") != 0 {
			return
		}
		b.frame()
	}
	b.s.history(b.t)
	sp := b.s.cpu.SP()
	for k := uint16(0); k < 8; k++ {
		b.t.Logf("stack %d: %s", k, b.s.where(b.t, b.s.word(sp+2*k)))
	}
	b.t.Fatalf("game busy for too long (%s)", b.s)
}

func (b *bot) frame() {
	b.s.runFrame()
	if b.m("gstate") == 2 && !b.ghostSeen {
		b.ghostSeen = true
		shot(b.t, b.s, "ghost")
	}
	if b.s.frame > 200000 {
		b.t.Fatalf("game took too long")
	}
}

// press presses a key for n frames, then releases it for n frames.
func (b *bot) press(key string, n int) {
	b.s.setKey(key, true)
	for i := 0; i < n; i++ {
		b.frame()
	}
	b.s.setKey(key, false)
	for i := 0; i < n; i++ {
		b.frame()
	}
}

// dismiss presses a key while the game is waiting for one.
func (b *bot) dismiss() bool {
	if b.m("waiting") == 0 {
		return false
	}
	if b.m("memories") == 1 && !b.memShot {
		b.memShot = true
		shot(b.t, b.s, "memory")
	}
	b.press("SPACE", 3)
	for i := 0; i < 20 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	return true
}

// walk follows a path to a goal in the current room. Returns false if the
// room changed on the way.
func (b *bot) walk(goal func(x, y int) bool) {
	room := b.m("room_id")
	stuck := 0
	for {
		b.settle()
		if b.m("room_id") != room || goal(int(b.m("px")), int(b.m("py"))) {
			break
		}
		if b.dismiss() {
			continue
		}
		p := b.path(goal)
		if p == nil {
			b.t.Fatalf("room %d: no path from %d,%d (%s) objs %+v rects % x", room, b.m("px"), b.m("py"), b.s, b.objects(), b.s.mem[b.s.label(b.t, "walk_rects"):b.s.label(b.t, "walk_rects")+20])
		}
		x, y := int(b.m("px")), int(b.m("py"))
		key := ""
		switch {
		case p[0].x > x:
			key = "P"
		case p[0].x < x:
			key = "O"
		case p[0].y > y:
			key = "A"
		default:
			key = "Q"
		}
		b.s.setKey(key, true)
		f0 := b.s.frame
		for i := 0; i < 4; i++ {
			b.frame()
			if b.m("room_id") != room || int(b.m("px")) != x || int(b.m("py")) != y {
				break
			}
		}
		b.s.setKey(key, false)
		b.settle()
		b.walkFrames += b.s.frame - f0
		b.walkSteps++
		if b.hist == nil {
			b.hist = map[int]int{}
		}
		b.hist[b.s.frame-f0]++
		if b.m("room_id") != room {
			return
		}
		if int(b.m("px")) == x && int(b.m("py")) == y {
			stuck++
			if stuck > 100 {
				b.t.Fatalf("room %d: stuck at %d,%d heading to %v", room, x, y, p[0])
			}
		} else {
			stuck = 0
		}
	}
}

func (b *bot) roomDims() (w, d int) { return int(b.m("room_w")), int(b.m("room_d")) }

// travel follows one step of a route.
func (b *bot) travel(st step) {
	from := int(b.m("room_id"))
	w, d := b.roomDims()
	switch st.dir {
	case 0:
		b.walk(func(x, y int) bool { return y < 13 })
	case 1:
		b.walk(func(x, y int) bool { return x >= w+20 })
	case 2:
		b.walk(func(x, y int) bool { return y >= d+20 })
	case 3:
		b.walk(func(x, y int) bool { return x < 13 })
	case 4:
		var stair obj
		for _, o := range b.objects() {
			if o.flags&(fUp|fDown) != 0 && o.extra == st.to {
				stair = o
			}
		}
		b.walk(func(x, y int) bool {
			return x+3 >= stair.bx0 && x+3 < stair.bx1 && y+3 >= stair.by0 && y+3 < stair.by1
		})
	}
	for i := 0; i < 100 && int(b.m("room_id")) != st.to; i++ {
		b.frame()
	}
	b.settle()
	if got := int(b.m("room_id")); got != st.to {
		b.t.Logf("pc=%04x waiting=%d nobj=%d", b.s.cpu.PC(), b.m("waiting"), b.m("nobj"))
		for i := 0; i < int(b.m("nobj")); i++ {
			b.t.Logf("obj %d: % x", i, b.s.mem[0xf600+16*i:0xf610+16*i])
		}
		b.t.Fatalf("expected to go from room %d to %d, but in %d; at %d,%d objs %+v", from, st.to, got, b.m("px"), b.m("py"), b.objects())
	}
	b.log = append(b.log, fmt.Sprintf("%d", st.to))
}

func (b *bot) goTo(goal func(r *roomInfo) bool, what string) {
	for {
		b.settle()
		cur := int(b.m("room_id"))
		if goal(&b.rooms[cur]) {
			return
		}
		r := b.route(cur, goal)
		if r == nil {
			b.t.Fatalf("no route from room %d to %s", cur, what)
		}
		b.travel(r[0])
	}
}

func TestPlaythrough(t *testing.T) {
	s := newSpectrum(t)
	s.ring = make([]uint16, 3000)
	s.protLo, s.protHi = s.label(t, "main"), s.label(t, "frames")
	s.onProt = func(a uint16) {
		s.history(t)
		t.Fatalf("write to code at %04x (%s) from %s", a, s.where(t, a), s.where(t, s.cpu.PC()))
	}
	b := &bot{t: t, s: s, types: s.label(t, "types")}
	b.readRooms()
	nitems := int(s.konst(t, "NITEMS"))
	// Title, then the introduction.
	for i := 0; i < 100 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	shot(t, s, "title")
	b.dismiss()
	for i := 0; i < 200 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	shot(t, s, "intro")
	b.dismiss()
	if got, want := int(b.m("room_id")), int(s.konst(t, "START_ROOM")); got != want {
		t.Fatalf("started in room %d, want %d", got, want)
	}
	shot(t, s, "start")
	for {
		// The nearest item not yet found.
		best, bestLen := -1, 1<<30
		for i := 0; i < nitems; i++ {
			if b.got(i) {
				continue
			}
			item := i
			r := b.route(int(b.m("room_id")), func(r *roomInfo) bool {
				for _, it := range r.items {
					if it == item {
						return true
					}
				}
				return false
			})
			if r != nil && len(r) < bestLen {
				best, bestLen = i, len(r)
			}
		}
		if best < 0 {
			break
		}
		item := best
		b.goTo(func(r *roomInfo) bool {
			for _, it := range r.items {
				if it == item {
					return true
				}
			}
			return false
		}, fmt.Sprintf("item %d", item))
		var o obj
		for _, x := range b.objects() {
			if x.flags&fItem != 0 && x.extra&0x7f == item {
				o = x
			}
		}
		b.walk(func(x, y int) bool {
			return b.got(item) || x < o.bx1 && x+6 > o.bx0 && y < o.by1 && y+6 > o.by0
		})
		for i := 0; i < 400 && !b.got(item); i++ {
			if !b.dismiss() {
				b.frame()
			}
		}
		if !b.got(item) {
			t.Fatalf("didn't pick up item %d", item)
		}
		for i := 0; i < 60; i++ {
			if !b.dismiss() {
				b.frame()
			}
		}
		t.Logf("found item %d after %d frames (memories %d)", item, s.frame, b.m("memories"))
	}
	if got := b.m("memories"); got != 8 {
		t.Fatalf("found %d memories, want 8", got)
	}
	// Ring the bell.
	b.goTo(func(r *roomInfo) bool { return r.bell }, "the belfry")
	shot(t, s, "belfry")
	b.press("M", 3)
	for i := 0; i < 100 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	shot(t, s, "map")
	b.dismiss()
	b.settle()
	var bell obj
	for _, o := range b.objects() {
		if o.flags&fBell != 0 {
			bell = o
		}
	}
	b.walk(func(x, y int) bool {
		return x-4 < bell.bx1 && x+10 > bell.bx0 && y-4 < bell.by1 && y+10 > bell.by0
	})
	b.press("SPACE", 3)
	panels := 0
	for i := 0; i < 3000 && b.m("game_over") == 0; i++ {
		if b.dismiss() {
			panels++
		} else {
			b.frame()
		}
	}
	if panels != int(s.konst(t, "NENDING")) {
		t.Errorf("saw %d ending panels, want %d", panels, s.konst(t, "NENDING"))
	}
	// Back to the title.
	for i := 0; i < 200 && b.m("waiting") == 0; i++ {
		b.frame()
	}
	if b.m("waiting") == 0 {
		t.Fatalf("not back at the title screen")
	}
	shot(t, s, "end")
	visited := 0
	for i := 0; i < len(b.rooms); i++ {
		if s.mem[s.label(t, "visited")+uint16(i)] != 0 {
			visited++
		}
	}
	t.Logf("walking: %.2f frames per step %v", float64(b.walkFrames)/float64(b.walkSteps), b.hist)
	t.Logf("finished in %d frames (%d minutes), visiting %d of %d rooms", s.frame, s.frame/3000, visited, len(b.rooms))
}
