"""Graphics for Hollowmere: a tiny rasteriser for isometric shapes, plus
the pixel art. Used by gen.py.

Coordinates
-----------
World units: a floor tile is 8x8 units. A world point (x, y, z) appears
on screen at

    sx = CX + 2*(x - y)
    sy = CY + (x + y) - z

so x runs down and to the right, y runs down and to the left, and z up.

Every sprite is stored with an anchor (ax, ay): drawn for an object whose
origin is at world (x, y), its top-left pixel is at
(CX + 2*(x - y) + ax, CY + x + y + ay).
"""

import math

T, INK, PAP = 0, 1, 2   # transparent, ink, paper (opaque black)

WALL_H = 40             # wall height in pixels
RIM = 5                 # depth of the floor slab's front face


class Img:
    def __init__(self, w, h):
        self.w, self.h = w, h
        self.p = [[T] * w for _ in range(h)]

    def get(self, x, y):
        if 0 <= x < self.w and 0 <= y < self.h:
            return self.p[y][x]
        return T

    def set(self, x, y, v):
        if 0 <= x < self.w and 0 <= y < self.h:
            self.p[y][x] = v

    def mirrored(self):
        m = Img(self.w, self.h)
        for y in range(self.h):
            m.p[y] = self.p[y][::-1]
        return m

    def outlined(self):
        """Adds a 1-pixel paper border around everything non-transparent."""
        o = Img(self.w, self.h)
        for y in range(self.h):
            for x in range(self.w):
                v = self.p[y][x]
                if v == T:
                    for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                        if self.get(x + dx, y + dy) != T:
                            v = PAP
                            break
                o.p[y][x] = v
        return o


class Sprite:
    def __init__(self, img, ax, ay):
        assert img.w % 8 == 0
        self.img, self.ax, self.ay = img, ax, ay

    def encode(self):
        """Returns the bytes: w, h, ax, ay, then (mask, data) pairs."""
        img = self.img
        out = [img.w // 8, img.h, self.ax & 0xff, self.ay & 0xff]
        for y in range(img.h):
            for bx in range(img.w // 8):
                m = d = 0
                for b in range(8):
                    v = img.p[y][bx * 8 + b]
                    m = m * 2 + (1 if v == T else 0)
                    d = d * 2 + (1 if v == INK else 0)
                out += [m, d]
        return out


def art(rows, key=None):
    """Makes an image from ASCII art: '#' ink, '.' paper, ' ' transparent."""
    key = key or {'#': INK, '.': PAP, ' ': T}
    w = max(len(r) for r in rows)
    w = (w + 7) // 8 * 8
    img = Img(w, len(rows))
    for y, r in enumerate(rows):
        for x, c in enumerate(r):
            img.p[y][x] = key[c]
    return img


def centred(img, wbytes):
    """Pads an image horizontally (centred) to the given width in bytes."""
    w = wbytes * 8
    if img.w == w:
        return img
    out = Img(w, img.h)
    off = (w - img.w) // 2
    for y in range(img.h):
        for x in range(img.w):
            out.set(x + off, y, img.p[y][x])
    return out


# ---------------------------------------------------------------------
# Fill patterns. Each takes absolute pixel coordinates and returns INK or
# PAP. Screen x parity is preserved (sprites are drawn byte aligned or
# pre-shifted by even amounts), y parity may vary.
# ---------------------------------------------------------------------

def p_paper(x, y): return PAP
def p_ink(x, y): return INK
def p_d50(x, y): return INK if (x + y) & 1 else PAP
def p_d25(x, y): return INK if (x & 1) and (y & 1) else PAP
def p_d12(x, y): return INK if (x & 3) == 1 and (y & 1) and ((x >> 2) + (y >> 1)) & 1 else PAP
def p_d75(x, y): return PAP if (x & 1) and (y & 1) else INK
def p_hl(x, y): return INK if y % 3 == 0 else PAP
def p_vl(x, y): return INK if x % 3 == 0 else PAP
def p_hl2(x, y): return INK if y & 1 else PAP
def p_vl2(x, y): return INK if x & 1 else PAP


# ---------------------------------------------------------------------
# Polygon rasterisation.
# ---------------------------------------------------------------------

def inside(pts, px, py):
    n = len(pts)
    c = False
    j = n - 1
    for i in range(n):
        xi, yi = pts[i]
        xj, yj = pts[j]
        if (yi > py) != (yj > py):
            xc = xi + (py - yi) * (xj - xi) / (yj - yi)
            if px < xc:
                c = not c
        j = i
    return c


def poly_pixels(pts, w, h):
    xs = [p[0] for p in pts]
    ys = [p[1] for p in pts]
    s = set()
    for j in range(max(0, int(math.floor(min(ys))) - 1), min(h, int(math.ceil(max(ys))) + 1)):
        for i in range(max(0, int(math.floor(min(xs))) - 1), min(w, int(math.ceil(max(xs))) + 1)):
            if inside(pts, i + 0.5, j + 0.5):
                s.add((i, j))
    return s


class Iso:
    """A canvas for an object whose anchor footprint is fw x fd units.
    The image is wbytes*8 wide, h rows high, and its bottom row is at
    screen y offset `bottom` from the object's origin (default: the
    front corner of the footprint)."""

    def __init__(self, wbytes, h, fw=8, fd=8, bottom=None, xoff=0):
        self.img = Img(wbytes * 8, h)
        self.left = (fw - fd) - 4 * wbytes + xoff
        bot = fw + fd if bottom is None else bottom
        self.top = bot - (h - 1)

    def proj(self, x, y, z):
        return (2 * (x - y) - self.left, (x + y) - z - self.top)

    def region(self, pts3):
        return poly_pixels([self.proj(*p) for p in pts3], self.img.w, self.img.h)

    def paint(self, faces, outline=True, silhouette=True):
        """faces is a list of (pixelset, pattern). Pixels on the
        silhouette of the union, and on boundaries with earlier faces,
        are inked."""
        union = set()
        for f, _ in faces:
            union |= f
        owner = {}
        for k, (f, _) in enumerate(faces):
            for p in f:
                owner[p] = k
        for k, (f, pat) in enumerate(faces):
            for (x, y) in f:
                if owner[(x, y)] != k:
                    continue
                v = pat(x, y)
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    q = (x + dx, y + dy)
                    if q not in union:
                        if silhouette:
                            v = INK
                    elif outline and owner[q] < k:
                        v = INK
                self.img.set(x, y, v)

    def box(self, x0, y0, z0, x1, y1, z1, top=p_paper, left=p_d50, right=p_d25, **kw):
        ft = self.region([(x0, y0, z1), (x1, y0, z1), (x1, y1, z1), (x0, y1, z1)])
        fl = self.region([(x0, y1, z0), (x1, y1, z0), (x1, y1, z1), (x0, y1, z1)])
        fr = self.region([(x1, y0, z0), (x1, y1, z0), (x1, y1, z1), (x1, y0, z1)])
        self.paint([(ft, top), (fl, left), (fr, right)], **kw)

    def ellipse(self, cx, cy, r, z):
        pts = []
        for k in range(32):
            a = 2 * math.pi * k / 32
            pts.append((cx + r * math.cos(a), cy + r * math.sin(a), z))
        return self.region(pts)

    def cyl(self, cx, cy, r, z0, z1, top=p_paper, shades=(p_d50, p_d25, p_d12), **kw):
        """A vertical cylinder. The side is shaded from left (lit) to right."""
        ft = self.ellipse(cx, cy, r, z1)
        fb = self.ellipse(cx, cy, r, z0)
        mx, my0 = self.proj(cx, cy, z1)
        _, my1 = self.proj(cx, cy, z0)
        a = 2 * math.sqrt(2) * r
        side = set(fb)
        for j in range(int(my0) - 1, int(my1) + 2):
            for i in range(int(mx - a) - 1, int(mx + a) + 2):
                if abs(i + 0.5 - mx) < a and my0 <= j + 0.5 <= my1:
                    side.add((i, j))
        side = {p for p in side if 0 <= p[0] < self.img.w and 0 <= p[1] < self.img.h}

        def shade(x, y):
            f = (x + 0.5 - (mx - a)) / (2 * a)
            k = 0 if f < 0.4 else (1 if f < 0.75 else 2)
            return shades[k](x, y)
        self.paint([(ft, top), (side - ft, shade)], **kw)

    def line(self, p, q, v=INK):
        (x0, y0), (x1, y1) = self.proj(*p), self.proj(*q)
        n = int(max(abs(x1 - x0), abs(y1 - y0)) * 2) + 1
        for k in range(n + 1):
            t = k / n
            self.img.set(int(math.floor(x0 + (x1 - x0) * t)), int(math.floor(y0 + (y1 - y0) * t)), v)

    def dot(self, p, v=INK):
        x, y = self.proj(*p)
        self.img.set(int(math.floor(x)), int(math.floor(y)), v)

    def blit(self, img, p, dx=0, dy=0):
        """Copies non-transparent pixels of img so that its bottom-centre
        lands on world point p."""
        x, y = self.proj(*p)
        ox = int(round(x)) - img.w // 2 + dx
        oy = int(round(y)) - img.h + 1 + dy
        for j in range(img.h):
            for i in range(img.w):
                v = img.p[j][i]
                if v != T:
                    self.img.set(ox + i, oy + j, v)

    def sprite(self):
        return Sprite(self.img, self.left, self.top)


def ascii_sprite(rows, wbytes=None, bottom=12, fw=8, fd=8, outline=True):
    """An upright object from ASCII art standing at the centre of its
    footprint; `bottom` is the screen offset of its lowest row."""
    img = art(rows)
    if outline:
        img = img.outlined()
    if wbytes is None:
        wbytes = img.w // 8
    img = centred(img, wbytes)
    return Sprite(img, (fw - fd) - 4 * wbytes, bottom - (img.h - 1))


# ---------------------------------------------------------------------
# Floor tiles, rims and walls.
# ---------------------------------------------------------------------

def floor_tile(style, parity=0):
    c = Iso(4, 17)
    reg = c.region([(0, 0, 0), (8, 0, 0), (8, 8, 0), (0, 8, 0)])
    for (x, y) in reg:
        v = PAP
        if style == 'flag':
            # Flagstones with a few worn marks.
            if (x * 7 + y * 13) % 29 == 0:
                v = INK
        elif style == 'check':
            if parity and (x & 1) and (y & 1) and ((x + y) & 2):
                v = INK
        elif style == 'wood':
            # Boards running along the x axis.
            if (2 * y - x) % 8 == 0:
                v = INK
        elif style == 'earth':
            if (x * 5 + y * 11) % 17 == 0 or (x * 3 + y * 7) % 23 == 0:
                v = INK
        elif style == 'carpet':
            v = p_d25(x, y)
        elif style == 'dark':
            if (x * 3 + y * 5) % 37 == 0:
                v = INK
        c.img.set(x, y, v)
    if style != 'earth':
        for (x, y) in reg:
            if (x, y - 1) not in reg:
                c.img.set(x, y, INK)
    return c.sprite()


def rim(edge):
    """The front face of the floor slab below one tile edge. edge 's' is
    the edge parallel to x (lower left), 'e' parallel to y (lower right).
    Anchored at the edge's start point."""
    img = Img(16, 8 + RIM + 1)
    for j in range(img.h):
        for i in range(16):
            u = (i + 0.5) / 2 if edge == 's' else (16 - i - 0.5) / 2
            s = j + 0.5 - u
            if 0 <= s <= RIM:
                img.set(i, j, PAP)
    for j in range(img.h):
        for i in range(16):
            if img.get(i, j) == PAP:
                below = img.get(i, j + 1) == T
                above = img.get(i, j - 1) == T
                if below or above:
                    img.set(i, j, INK)
                elif edge == 's':
                    img.set(i, j, p_d50(i, j))
                else:
                    img.set(i, j, p_d25(i, j))
    return img


def stone(u2, r):
    """Wall texture at flat wall coordinates (u2 = 0..15 along the wall,
    r = rows down from the top of the wall)."""
    course = r // 8
    if r % 8 == 7:
        return INK if u2 % 2 == 0 else PAP
    if (u2 + course * 4) % 8 == 0 and r % 8 != 0:
        return INK if r % 2 == 0 else PAP
    return PAP


def wall_segment(side, design):
    """A 16-pixel wide piece of back wall (one tile long) with a flat
    design drawn on it. side 'n' is the wall along x (upper right), 'w'
    the wall along y (upper left). design is a list of WALL_H strings of
    16 chars: ' ' = stone, '#' ink, '.' paper, '~' plain paper with no
    stone texture."""
    h = WALL_H + 9
    img = Img(16, h)
    for j in range(h):
        for i in range(16):
            u = (i + 0.5) / 2
            if side == 'w':
                u = 8 - u
            v = u + WALL_H - (j + 0.5)
            if 0 <= v <= WALL_H:
                r = WALL_H - 1 - int(v)
                u2 = int(u * 2)
                if side == 'w':
                    u2 = 15 - u2
                c = design[r][u2] if design else ' '
                if c == ' ':
                    val = stone(u2, r)
                elif c == '#':
                    val = INK
                else:
                    val = PAP
                img.set(i, j, val)
    # Top and bottom edges.
    for j in range(h):
        for i in range(16):
            if img.get(i, j) != T and (img.get(i, j - 1) == T or img.get(i, j + 1) == T):
                img.set(i, j, INK)
    return img


def wall_sprite(img):
    # Wall segments are positioned explicitly by the engine.
    return Sprite(img, 0, 0)


def design(rows):
    """Pads a partial wall design (listed from the top) to WALL_H rows."""
    rows = [r.ljust(16) for r in rows]
    return rows + [' ' * 16] * (WALL_H - len(rows))


def design_bottom(rows):
    rows = [r.ljust(16) for r in rows]
    return [' ' * 16] * (WALL_H - len(rows)) + rows


WALL_DESIGNS = {
    'plain': None,
    'arch': design_bottom([
        "    ########    ",
        "  ##........##  ",
        " #............# ",
        " #............# ",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
        "#..............#",
    ]),
    'window': design([
        "",
        "",
        "",
        "     ######     ",
        "    #......#    ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   ##########   ",
        "  ############  ",
    ]),
    'moon': design([
        "",
        "",
        "",
        "     ######     ",
        "    #......#    ",
        "   #........#   ",
        "   #.....##.#   ",
        "   #....####.   ",
        "   #....###.#   ",
        "   #....####.   ",
        "   #.....##.#   ",
        "   #........#   ",
        "   #........#   ",
        "   #..#.....#   ",
        "   #........#   ",
        "   #........#   ",
        "   ##########   ",
        "  ############  ",
    ]),
    'portrait': design([
        "",
        "",
        "",
        "",
        "  ############  ",
        "  #..........#  ",
        "  #.########.#  ",
        "  #.#......#.#  ",
        "  #.#..##..#.#  ",
        "  #.#.####.#.#  ",
        "  #.#.#..#.#.#  ",
        "  #.#.####.#.#  ",
        "  #.#..##..#.#  ",
        "  #.#.####.#.#  ",
        "  #.#######..#  ",
        "  #.########.#  ",
        "  #..........#  ",
        "  ############  ",
    ]),
    'torch': design([
        "",
        "",
        "",
        "",
        "",
        "",
        "       #        ",
        "      ##        ",
        "      #.#       ",
        "     #..#       ",
        "     #.#.#      ",
        "     #..#       ",
        "      ##        ",
        "    ######      ",
        "    #....#      ",
        "     #..#       ",
        "      ##        ",
        "      ##        ",
        "      ##        ",
        "     ####       ",
    ]),
    'banner': design([
        "",
        "",
        "  ############  ",
        "  ############  ",
        "   #........#   ",
        "   #.######.#   ",
        "   #.#....#.#   ",
        "   #.#.##.#.#   ",
        "   #.#.##.#.#   ",
        "   #.#.##.#.#   ",
        "   #.#....#.#   ",
        "   #.######.#   ",
        "   #........#   ",
        "   #.#.#.#.##   ",
        "   ##.#.#.#.#   ",
        "   #........#   ",
        "   #........#   ",
        "   #...##...#   ",
        "   #..#..#..#   ",
        "   #.#....#.#   ",
        "   ##......##   ",
        "   #        #   ",
    ]),
    'shield': design([
        "",
        "",
        "",
        "",
        "",
        " #            # ",
        "  #          #  ",
        "   # ###### #   ",
        "    #......#    ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #..#..#..#   ",
        "   #.#.##.#.#   ",
        "   #........#   ",
        "  # #......# #  ",
        " #   #....#   # ",
        "      ####      ",
    ]),
    'web': design([
        "#..#....#......#",
        "#.#.....#....##.",
        "##.#####.#.##...",
        "#.#.....###.....",
        "#.#...##.#......",
        "##..##...#......",
        "#.##....#.......",
        "##.....#........",
        "#.#...#.........",
        "#..#.#..........",
        "#...#...........",
        "#..#............",
        "#.#.............",
        "##..............",
        "#...............",
    ]),
    'hearth': design_bottom([
        "################",
        "#..............#",
        "################",
        " #............# ",
        " #..########..# ",
        " #.#........#.# ",
        " #.#........#.# ",
        " #.#...#....#.# ",
        " #.#..#.#...#.# ",
        " #.#..#.#.#.#.# ",
        " #.#.#...#.##.# ",
        " #.#.#.#.#..#.# ",
        " #.#########..# ",
        " #.#........#.# ",
        "################",
    ]),
    'crack': design([
        "",
        "",
        "",
        "",
        "",
        "        #       ",
        "        #       ",
        "       #        ",
        "       #        ",
        "        #       ",
        "        ##      ",
        "       #  #     ",
        "      #    #    ",
        "      #         ",
        "     #          ",
        "     #          ",
        "    #           ",
        "     #          ",
        "     #          ",
        "    #           ",
    ]),
}
WALL_KINDS = list(WALL_DESIGNS.keys())
