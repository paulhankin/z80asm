"""The sprites of Hollowmere: furniture, items, the wanderer and the
grey lady. Used by gen.py."""

from gfx import *


def iso(wbytes, zmax, tw=1, td=1, extra=1):
    fw, fd = 8 * tw, 8 * td
    return Iso(wbytes, fw + fd + zmax + extra, fw, fd)


# ---------------------------------------------------------------------
# Furniture built from boxes and cylinders.
# ---------------------------------------------------------------------

def block():
    c = iso(4, 8)
    c.box(0, 0, 0, 8, 8, 8, top=p_d12)
    return c.sprite()


def low_block():
    c = iso(4, 4)
    c.box(0, 0, 0, 8, 8, 4, top=p_d12)
    return c.sprite()


def pillar():
    c = iso(4, 40)
    c.box(1, 1, 0, 7, 7, 3)
    c.cyl(4, 4, 2.6, 3, 36)
    c.box(1, 1, 36, 7, 7, 40, top=p_d12)
    return c.sprite()


def table():
    c = iso(4, 9)
    for (x, y) in ((1, 1), (6, 1), (1, 6), (6, 6)):
        c.box(x, y, 0, x + 1, y + 1, 7, left=p_ink, right=p_ink)
    c.box(0.5, 0.5, 7, 7.5, 7.5, 9, top=p_paper)
    return c.sprite()


def chair():
    c = iso(2, 12)
    for (x, y) in ((2.5, 2.5), (5, 2.5), (2.5, 5), (5, 5)):
        c.box(x, y, 0, x + 0.5, y + 0.5, 4, left=p_ink, right=p_ink)
    c.box(2.5, 2.5, 4, 5.5, 5.5, 5)
    c.box(2.5, 2.2, 5, 5.5, 3, 12, left=p_d25, right=p_d12)
    return c.sprite()


def bed():
    c = iso(6, 15, tw=1, td=2)
    c.box(0.5, 0, 0, 7.5, 1.5, 15, top=p_d12, left=p_d25)
    c.box(0.5, 1.5, 0, 7.5, 15.5, 5, left=p_d50)

    def quilt(x, y):
        return INK if (x % 4 == 0) != (y % 2 == 0) and (x + y) % 3 else PAP
    c.box(1, 5, 5, 7, 15, 7, top=quilt, left=p_d25)
    c.box(1.5, 2, 5, 6.5, 5, 8, top=p_paper, left=p_d12)
    return c.sprite()


def coffin():
    c = iso(6, 8, tw=1, td=2)
    c.box(1.5, 1, 0, 6.5, 15, 5, left=p_d50, right=p_d25)
    c.box(1, 0.5, 5, 7, 15.5, 7, top=p_d12)
    # A cross on the lid.
    c.line((4, 3, 7), (4, 9, 7))
    c.line((2.5, 5, 7), (5.5, 5, 7))
    return c.sprite()


def bookcase():
    c = iso(4, 30)
    y1 = 3.5
    left, top = c.left, c.top

    def shelves(px, py):
        X = (px + 0.5 + left) / 2 + y1
        Z = X + y1 - (py + 0.5 + top)
        if Z % 7.5 < 1.0:
            return INK
        k = int(X * 2)
        if k % 2 == 0 and (k * 7 + int(Z / 7.5)) % 5 != 0:
            return INK
        return PAP
    c.box(0, 0, 0, 8, y1, 30, top=p_d12, left=shelves, right=p_d25)
    return c.sprite()


def barrel():
    c = iso(4, 11)
    c.cyl(4, 4, 3, 0, 11, top=p_d12)
    for z in (2, 9):
        for k in range(40):
            a = math.pi * (0.25 + k / 40)
            c.dot((4 + 3 * math.cos(a), 4 + 3 * math.sin(a), z))
    return c.sprite()


def throne():
    c = iso(4, 22)
    c.box(1, 1, 0, 7, 2.5, 22, top=p_d12, left=p_d25)
    c.box(1, 2.5, 0, 7, 7, 6, left=p_d50)
    c.box(1, 2.5, 6, 2, 7, 10, left=p_d25)
    c.box(6, 2.5, 6, 7, 7, 10, left=p_d25)
    # A crown carved in the back.
    for x in (2.5, 4, 5.5):
        c.line((x, 2.5, 17), (x, 2.5, 20))
    c.line((2.5, 2.5, 17), (5.5, 2.5, 17))
    return c.sprite()


def pew():
    c = Iso(6, 16 + 8 + 11, 16, 8)
    c.box(0.5, 2, 0, 15.5, 3, 11, top=p_d12, left=p_d25)
    c.box(0.5, 3, 3, 15.5, 6.5, 4.5, left=p_d50)
    c.box(0.5, 2, 0, 1.5, 7, 7, left=p_d25)
    c.box(14.5, 2, 0, 15.5, 7, 7, left=p_d25)
    return c.sprite()


def altar():
    c = Iso(6, 16 + 8 + 20, 16, 8)

    def cloth(x, y):
        return INK if (x // 2 + y) % 4 == 0 else PAP
    c.box(1, 1.5, 0, 15, 6.5, 10, top=p_paper, left=cloth, right=p_d25)
    # Two candlesticks and a cross.
    for x in (3, 13):
        c.box(x - 0.5, 3.5, 10, x + 0.5, 4.5, 15, left=p_ink, right=p_ink)
        c.dot((x, 4, 17))
    c.box(7.5, 3.5, 10, 8.5, 4.5, 20, left=p_ink, right=p_ink)
    c.box(6, 3.5, 16, 10, 4.5, 17, left=p_ink, right=p_ink)
    return c.sprite()


def well():
    c = iso(4, 24)
    for x in (0.8, 7.2):
        c.box(x - 0.5, 3.5, 0, x + 0.5, 4.5, 20, left=p_ink, right=p_ink)
    c.cyl(4, 4, 3.6, 0, 7, top=p_ink)
    inner = c.ellipse(4, 4, 2.6, 7)
    for p in inner:
        c.img.set(p[0], p[1], PAP)
    for x in (0.8, 7.2):
        c.box(x - 0.5, 3.5, 7, x + 0.5, 4.5, 20, left=p_ink, right=p_ink)
    c.box(0, 3, 20, 8, 5, 23, top=p_d25, left=p_d50)
    c.line((4, 4, 20), (4, 4, 12))
    return c.sprite()


def chest():
    c = iso(4, 8)
    c.box(1, 2, 0, 7, 6, 5, left=p_d25, right=p_d12)
    c.box(1, 2, 5, 7, 6, 8, top=p_d25, left=p_d50, right=p_d25)
    c.line((1, 6, 2), (7, 6, 2))
    c.dot((4, 6, 5))
    c.dot((4, 6, 4))
    return c.sprite()


BELL = [
    "  ##########################  ",
    "  #........................#  ",
    "  ##########################  ",
    "   #..#        ##       #..#  ",
    "   #..#        ##       #..#  ",
    "   #..#      ######     #..#  ",
    "   #..#     #......#    #..#  ",
    "   #..#    #.#......#   #..#  ",
    "   #..#    #.#......#   #..#  ",
    "   #..#   #.#........#  #..#  ",
    "   #..#   #.#........#  #..#  ",
    "   #..#   #.#........#  #..#  ",
    "   #..#  #.#..........# #..#  ",
    "   #..#  #.#..........# #..#  ",
    "   #..#  #.#..........# #..#  ",
    "   #..# #..............##..#  ",
    "   #..# ################ #..#  ",
    "   #..#  #............#  #..#  ",
    "   #..#   ############   #..#  ",
    "   #..#       #..#       #..#  ",
    "   #..#        ##        #..#  ",
    "   #..#                  #..#  ",
    "   #..#                  #..#  ",
    "   #..#                  #..#  ",
    "  ######                ######  ",
    "  #....#                #....#  ",
    "  ######                ######  ",
]


def bell():
    return ascii_sprite(BELL, 4, bottom=12)


def clock():
    c = iso(2, 31)
    c.box(2.5, 2.5, 0, 5.5, 5.5, 29, top=p_d12, left=p_paper, right=p_d25)
    c.box(2.2, 2.2, 29, 5.8, 5.8, 31, top=p_d12)
    # Face and pendulum.
    for k in range(16):
        a = 2 * math.pi * k / 16
        c.dot((4 + 1.1 * math.cos(a), 5.5, 24 + 1.3 * math.sin(a)))
    c.line((4, 5.5, 24), (4.6, 5.5, 25))
    c.line((4, 5.5, 20), (4, 5.5, 8))
    c.dot((4, 5.5, 7))
    c.dot((3.5, 5.5, 7))
    return c.sprite()


def cradle():
    c = iso(4, 12)
    for y in (2.5, 5.5):
        c.line((1, y, 2), (7, y, 2))
        c.line((0.5, y, 3), (1, y, 2))
        c.line((7, y, 2), (7.5, y, 3))
    c.box(1.5, 2, 3, 6.5, 6, 8, top=p_d25, left=p_vl, right=p_d12)
    c.box(1.5, 2, 8, 3, 6, 12, top=p_d12, left=p_d25)
    return c.sprite()


def desk():
    c = iso(4, 13)
    c.box(0.5, 1, 0, 3, 7, 8, left=p_d25, right=p_d12)
    c.box(6, 1, 0, 7.5, 7, 8, left=p_d25, right=p_d12)
    c.box(0.5, 1, 8, 7.5, 7, 9.5, top=p_paper)
    c.box(2, 2, 9.5, 4.5, 4, 10.5, top=p_hl2)
    # An inkwell with a quill.
    c.box(5, 2.5, 9.5, 6, 3.5, 11, left=p_ink, right=p_ink)
    c.line((5.5, 3, 11), (6.5, 3, 13.5))
    return c.sprite()


def cauldron():
    c = iso(4, 9)
    for (x, y) in ((1.5, 4), (6, 2), (6, 6)):
        c.box(x - 0.4, y - 0.4, 0, x + 0.4, y + 0.4, 3, left=p_ink, right=p_ink)
    c.cyl(4, 4, 3.3, 2, 8, top=p_ink, shades=(p_d25, p_d12, p_paper))
    for p in c.ellipse(4, 4, 2.6, 8):
        c.img.set(p[0], p[1], PAP)
    return c.sprite()


def carpet():
    c = Iso(4, 17)
    reg = c.region([(0.8, 0.8, 0), (7.2, 0.8, 0), (7.2, 7.2, 0), (0.8, 7.2, 0)])
    inner = c.region([(2, 2, 0), (6, 2, 0), (6, 6, 0), (2, 6, 0)])
    for (x, y) in reg:
        edge = any((x + dx, y + dy) not in reg for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)))
        if edge:
            v = INK
        elif (x, y) in inner:
            v = p_d25(x, y)
        else:
            v = p_d50(x, y)
        c.img.set(x, y, v)
    return c.sprite()


def stairs_up():
    c = iso(4, 16)
    n = 5
    for k in range(n):
        y0, y1 = 8 * k / n, 8 * (k + 1) / n
        c.box(0.3, y0, 0, 7.7, y1, 16 - 3 * k, top=p_d12, left=p_paper, right=p_d25)
    return c.sprite()


def stairs_down():
    c = Iso(4, 17)
    outer = c.region([(0.5, 0.5, 0), (7.5, 0.5, 0), (7.5, 7.5, 0), (0.5, 7.5, 0)])
    hole = c.region([(1.5, 0.5, 0), (7.5, 0.5, 0), (7.5, 6.5, 0), (1.5, 6.5, 0)])
    for (x, y) in outer:
        c.img.set(x, y, INK if (x, y) not in hole else PAP)
    # Step edges, going down towards the back wall.
    for y in (5.5, 4, 2.5):
        c.line((1.6, y, 0), (7.4, y, 0))
    for (x, y) in outer:
        if any((x + dx, y + dy) not in outer for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1))):
            c.img.set(x, y, INK)
    return c.sprite()


def gate(side):
    """A portcullis in a doorway. side is the wall the doorway is in."""
    c = iso(4, 30)
    img = c.img
    if side == 'ns':
        y = 4
        pts = [((x, y, 0), (x, y, 28)) for x in (0.8, 2.6, 4.4, 6.2, 7.6)]
        pts += [((0.5, y, z), (7.9, y, z)) for z in (7, 15, 23, 28)]
    else:
        x = 4
        pts = [((x, y, 0), (x, y, 28)) for y in (0.4, 1.8, 3.6, 5.4, 7.2)]
        pts += [((x, 0.1, z), (x, 7.5, z)) for z in (7, 15, 23, 28)]
    for p, q in pts:
        c.line(p, q)
    # Thicken vertical bars, then outline for visibility.
    thick = [(x, y) for y in range(img.h) for x in range(img.w) if img.p[y][x] == INK]
    for (x, y) in thick:
        if img.get(x - 1, y - 1) != INK and img.get(x + 1, y - 1) != INK:
            img.set(x + 1, y, INK)
    c.img = img.outlined()
    return c.sprite()


# ---------------------------------------------------------------------
# Upright things drawn as pixel art.
# ---------------------------------------------------------------------

ARMOUR = [
    "      ##       #",
    "     #..#     #.",
    "    #....#    #.",
    "    #.##.#    #.",
    "    #....#    #.",
    "     #..#     #.",
    "   ########  #..",
    "  #..#..#..# #..",
    "  #.#....#.##...",
    " #..#....#..#..#",
    " #.#.#..#.#.#.# ",
    " #.#..##..#.#.# ",
    " #.#.#..#.#.#.# ",
    " #.#..##..#.#.# ",
    " ##.#....#.##.# ",
    "  #.######.# .# ",
    "  #.#....#.# .# ",
    "   #.#..#.#  .# ",
    "   #.#..#.#  .# ",
    "   #.#..#.#  .# ",
    "   #.#..#.#  .# ",
    "   #.#..#.#  .# ",
    "  #..#..#..# .# ",
    "  ####..####  # ",
    " ############## ",
    " #............# ",
    " ############## ",
]


def candelabra(frame):
    flames = [
        ["  #     #    #  ",
         " #.#   #.#  #.# ",
         " #.#  #..#  #.# ",
         "  #    ##    #  "],
        ["   #   #    #   ",
         "  #.#  #.#  #.# ",
         "  #.#  #.#  #.  ",
         "  ##    #    #  "],
    ][frame]
    body = [
        " ###   ###  ### ",
        " #.#   #.#  #.# ",
        " #.#   #.#  #.# ",
        " #.#   #.#  #.# ",
        " ############## ",
        "  #....#.#....# ",
        "   ####...####  ",
        "       #.#      ",
        "       #.#      ",
        "       #.#      ",
        "       #.#      ",
        "       #.#      ",
        "       #.#      ",
        "       #.#      ",
        "      #...#     ",
        "      #.#.#     ",
        "     #.....#    ",
        "    #########   ",
    ]
    return ascii_sprite(flames + body, 2, bottom=11)


def candle(frame):
    flames = [
        ["       #        ",
         "      #.#       ",
         "      #..#      ",
         "       ##       "],
        ["        #       ",
         "       #.#      ",
         "      #.#       ",
         "       #        "],
    ][frame]
    body = [
        "      ###       ",
        "      #.#       ",
        "      #.#       ",
        "      #.#       ",
        "    #######     ",
        "     #...#      ",
        "      #.#       ",
        "      #.#       ",
        "      #.#       ",
        "      #.#       ",
        "     #...#      ",
        "   #########    ",
    ]
    return ascii_sprite(flames + body, 2, bottom=11)


URN = [
    "    ########    ",
    "    #......#    ",
    "     ######     ",
    "      #..#      ",
    "    ##....##    ",
    "   #........#   ",
    "  #..#.#.#...#  ",
    "  #.#.#.#.#..#  ",
    "  #..........#  ",
    "   #........#   ",
    "    #......#    ",
    "     #....#     ",
    "    ########    ",
    "    #......#    ",
    "    ########    ",
]

GRAVE = [
    "     ######     ",
    "   ##......##   ",
    "  #..........#  ",
    "  #....##....#  ",
    "  #...####...#  ",
    "  #....##....#  ",
    "  #....##....#  ",
    "  #..........#  ",
    "  #.##.#.##..#  ",
    "  #..........#  ",
    "  #..#.##.#..#  ",
    "  #..........#  ",
    "  #..........#  ",
    " ############## ",
    " #............# ",
    " ############## ",
]

TREE = [
    "          #      #    #         ",
    "     #     #    #    #     #    ",
    "      #     #  #    #     #     ",
    "       ##    ##    #    ##      ",
    "   #     #    #   #   ##    #   ",
    "    ##    #   #  #   #    ##    ",
    "      ##   #  # #   #   ##      ",
    "        ##  #.#.#  #  ##        ",
    "          ##.....###            ",
    "            #...#               ",
    "            #...#               ",
    "            #...#               ",
    "            #..#                ",
    "           #...#                ",
    "           #...#                ",
    "           #....#               ",
    "          #......#              ",
    "        ##..#..#..##            ",
    "       #  ##    ##  #           ",
]

STATUE = [
    " #            # ",
    " ##    ##    ## ",
    " #.#  #..#  #.# ",
    " #..# #..# #..# ",
    " #..#  ##  #..# ",
    " #...######...# ",
    "  #..#....#..#  ",
    "  #..#.##.#..#  ",
    "   #.#.##.#.#   ",
    "   ##.#..#.##   ",
    "    #.#..#.#    ",
    "    #......#    ",
    "    #......#    ",
    "    #.#..#.#    ",
    "   #..#..#..#   ",
    "   #..#..#..#   ",
    "   #..#..#..#   ",
    "   #..#..#..#   ",
    "  #...#..#...#  ",
    "  #..........#  ",
    "  ############  ",
    "  #..........#  ",
    "  #.#.#.#.#..#  ",
    "  #..........#  ",
    "  ############  ",
]

MIRROR = [
    "     ######     ",
    "   ##......##   ",
    "  #..######..#  ",
    "  #.#......#.#  ",
    " #.#......#.#.# ",
    " #.#.....#..#.# ",
    " #.#........#.# ",
    " #.#........#.# ",
    " #.#........#.# ",
    " #.#......#.#.# ",
    " #.#.....#..#.# ",
    " #.#........#.# ",
    " #.#........#.# ",
    "  #.#......#.#  ",
    "  #..######..#  ",
    "   ##......##   ",
    "     ##..##     ",
    "      #..#      ",
    "      #..#      ",
    "      #..#      ",
    "    ##....##    ",
    "   #........#   ",
    "   ##########   ",
]

BONES = [
    "          ####                  ",
    "         #....#   #  #          ",
    "         #.#.##   ####          ",
    "          #..#  ##....##        ",
    "   ##  ###.##.##..####.##   ##  ",
    "  #..##....#......#....#..##..# ",
    "   ##.##.#....#...##..#.##.##   ",
    "  #..#  ###.....#....##.#  #..# ",
    "   ##       #####..###     ##   ",
]

HORSE = [
    "                  ###           ",
    "                 #...##         ",
    "                #..#...#        ",
    "               #........#       ",
    "       #########.....###        ",
    "      #...............#         ",
    "     #...............#          ",
    "    ##...............#          ",
    "   #.##.............#           ",
    "   #   #.#......#.#             ",
    "       #.#      #.#             ",
    "       #.#      #.#             ",
    "    ####.#########.####         ",
    "   #...................#        ",
    "    ###################         ",
]

# The items. Each is drawn small, standing at the middle of its tile.
ITEM_ART = {
    'box': [
        "      ##        ",
        "      #.#       ",
        "  ##########    ",
        " #..........#   ",
        "############.#  ",
        "#..........#.#  ",
        "#.########.#.#  ",
        "#.#......#.#.#  ",
        "#.########.#.#  ",
        "#..........#.#  ",
        "#..........##   ",
        "############    ",
    ],
    'locket': [
        "    #    #      ",
        "     #  #       ",
        "      ##        ",
        "     ####       ",
        "   ##....##     ",
        "  #..#..#..#    ",
        "  #........#    ",
        "  #..####..#    ",
        "   #......#     ",
        "    ##..##      ",
        "      ##        ",
    ],
    'letter': [
        "################",
        "#..............#",
        "##............##",
        "#.##........##.#",
        "#...##....##...#",
        "#.....####.....#",
        "#..............#",
        "#..............#",
        "################",
    ],
    'horse': [
        "         ###    ",
        "        #...#   ",
        "       #..#.##  ",
        "       #.....#  ",
        "  ######...##   ",
        " #.........#    ",
        "#..........#    ",
        " #........#     ",
        "  #.#..#.#      ",
        "  #.#  #.#      ",
        " #########      ",
        "#.........#     ",
        " #########      ",
    ],
    'shard': [
        "      #         ",
        "     #.#        ",
        "     #..#       ",
        "    #.#..#      ",
        "    #..#..#     ",
        "   #...#...#    ",
        "   #....#...#   ",
        "  #.....#....#  ",
        "  ############  ",
    ],
    'ring': [
        "     ####       ",
        "    #.##.#      ",
        "     ####       ",
        "   ##....##     ",
        "  #..####..#    ",
        "  #.#    #.#    ",
        "  #.#    #.#    ",
        "  #..####..#    ",
        "   ##....##     ",
        "     ####       ",
    ],
    'rose': [
        "     ###        ",
        "    #.#.#       ",
        "   #.#.#.#      ",
        "    #.#.#       ",
        "     ###        ",
        "      #    ##   ",
        "      #   #..#  ",
        "      #  #..#   ",
        "      # ####    ",
        "      #         ",
        "     ##         ",
        "      #         ",
        "      #         ",
    ],
    'painting': [
        "  ############  ",
        "  #..........#  ",
        "  #.########.#  ",
        "  #.#......#.#  ",
        "  #.#.#..#.#.#  ",
        "  #.#.##.#.#.#  ",
        "  #.#.##.#.#.#  ",
        "  #.#......#.#  ",
        "  #.########.#  ",
        "  #..........#  ",
        "  ############  ",
        "      #  #      ",
        "     #    #     ",
    ],
    'key': [
        "                ",
        "   ####         ",
        "  #....#        ",
        "  #.##.#        ",
        "  #.##.#        ",
        "  #....#        ",
        "   #..######### ",
        "   ###.......#  ",
        "      ######.#  ",
        "          #.#   ",
        "          ###   ",
    ],
}

SPARKLE = [(13, 0), (12, 1), (14, 1), (13, 2)]


def item(name, frame):
    rows = [r.ljust(16) for r in ITEM_ART[name]]
    img = art(rows).outlined()
    # A glint that comes and goes, above the item.
    full = Img(16, img.h + 4)
    for y in range(img.h):
        for x in range(16):
            full.set(x, y + 4, img.p[y][x])
    if frame:
        for (x, y) in SPARKLE:
            full.set(x, y, INK)
    return Sprite(full, -8, 11 - (full.h - 1))


# ---------------------------------------------------------------------
# The wanderer (the player): 24 pixels wide, standing on a 6x6 unit
# footprint. FRONT faces down and to the right, BACK up and to the
# right; the other two directions are mirror images.
# ---------------------------------------------------------------------

# 'X' ink, 'o' shaded ink, '.' paper, ' ' transparent.
FRONT = [
    "          XXXX          ",
    "        XXXXXXXX        ",
    "       XXXXXXXXXX       ",
    "      XXXX....XXXX      ",
    "      XXX......XXo      ",
    "     XXX...X.X..XXo     ",
    "     XXX........XXo     ",
    "     XXXX......XXoo     ",
    "     XXXXX....XXXoo     ",
    "      XXXXXXXXXXoo      ",
    "     XXXXXXXXXXXooo     ",
    "     XXXX.XXXXX.ooo     ",
    "    XXXXX.XXXXX.oooo    ",
    "    XXXX.XXXXXXX.ooo    ",
    "   XXXXX.XXXXXXX.oooo   ",
    "   XXXX.XXXXXXXXX.ooo   ",
    "   XXXX.XXXXXXXXX.ooo   ",
    "  XXXXX.XXXXXXXXX.oooo  ",
    "  XXXX.XXXXXXXXXXX.ooo  ",
    "  XXXX.XXXXXXXXXXX.ooo  ",
    "  XXXX.XXXXXXXXXXX.ooo  ",
    "  XXXXXXXXXXXXXXXXoooo  ",
    "   XXXXXXXXXXXXXXXooo   ",
]

BACK = [
    "          XXXX          ",
    "        XXXXXXXX        ",
    "       XXXXXXXXXo       ",
    "      XXXXXXXXXXoo      ",
    "      XXXXXXXXXXoo      ",
    "     XXXXXXXXXXXooo     ",
    "     XXXXXXXXXXXooo     ",
    "     XXXXXXXXXXXooo     ",
    "     XXXX.....XXooo     ",
    "      XXXXXXXXXXoo      ",
    "     XXXXXXXXXXXooo     ",
    "     XXXXXXX.XXXooo     ",
    "    XXXXXXXX.XXXoooo    ",
    "    XXXXXXXX.XXXoooo    ",
    "   XXXXXXXXX.XXXooooo   ",
    "   XXXXXXXXX.XXXooooo   ",
    "   XXXXXXXXX.XXXooooo   ",
    "  XXXXXXXXXX.XXXoooooo  ",
    "  XXXXXXXXXX.XXXoooooo  ",
    "  XXXXXXXXX...XXoooooo  ",
    "  XXXXXXXX.....Xoooooo  ",
    "  XXXXXXXXXXXXXXoooooo  ",
    "   XXXXXXXXXXXXXooooo   ",
]

LEGS = [
    [   # standing
        "       XXX    XXX       ",
        "       XXX    XXX       ",
        "      XXXX   XXXX       ",
    ],
    [   # stepping
        "      XXX      XXX      ",
        "     XXX        XXX     ",
        "    XXXX        XXXX    ",
    ],
    [   # the other step
        "        XXX  XXX        ",
        "        XXX XXX         ",
        "       XXXXXXXX         ",
    ],
]

LANTERN = [
    "  X  ",
    " XXX ",
    "X...X",
    "X.X.X",
    "X...X",
    " XXX ",
]

PKEY = {'X': INK, 'o': 'shade', '.': PAP, ' ': T}


def wanderer(back, step):
    rows = (BACK if back else FRONT) + LEGS[step]
    img = Img(24, len(rows))
    for y, r in enumerate(rows):
        for x, c in enumerate(r):
            v = PKEY[c]
            if v == 'shade':
                v = p_d50(x, y)
            img.p[y][x] = v
    if not back:
        # The lantern hangs from the right hand, and glows.
        dy = (0, 1, -1)[step]
        for y, r in enumerate(LANTERN):
            for x, c in enumerate(r):
                if c != ' ':
                    img.set(18 + x, 15 + y + dy, PKEY[c])
        img.set(20, 18 + dy, INK)
    return img.outlined()


def player_sprite(img):
    # Footprint 6x6: centre at 2*(3-3) = 0; bottom row at py = 8.
    return Sprite(img, -12, 8 - (img.h - 1))


# ---------------------------------------------------------------------
# The grey lady: drawn translucently, at three strengths so she can
# fade in and out. Only ink is drawn; the mask is transparent elsewhere.
# ---------------------------------------------------------------------

LADY = [
    "         #####          ",
    "        #.....#         ",
    "       #.......#        ",
    "      #..#...#..#       ",
    "      #.........#       ",
    "      #...###...#       ",
    "       #.......#        ",
    "      ##.#####.##       ",
    "     #..#.....#..#      ",
    "    #...#.....#...#     ",
    "    #..#.......#..#     ",
    "   #...#.......#...#    ",
    "   #..#.........#..#    ",
    "   #..#.........#..#    ",
    "    ##...........##     ",
    "     #...........#      ",
    "     #...........#      ",
    "    #.............#     ",
    "    #.............#     ",
    "   #...............#    ",
    "   #...............#    ",
    "  #.................#   ",
    "  #.................#   ",
    "   #..#...#...#..#.#    ",
    "    ##  #   #  ##  #    ",
    "         #              ",
]


def lady(level):
    img = art(LADY)
    out = Img(24, img.h)
    for y in range(img.h):
        for x in range(24):
            v = img.p[y][x]
            if v == INK:
                on = level >= 1 or (x + y) % 2 == 0
            elif v == PAP:
                if level == 2:
                    on = p_d50(x, y) == INK
                elif level == 1:
                    on = p_d25(x, y) == INK
                else:
                    on = p_d12(x, y) == INK
            else:
                on = False
            out.p[y][x] = INK if on else T
    return Sprite(out, -12, 8 - (out.h - 1))
