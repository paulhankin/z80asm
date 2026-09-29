#!/usr/bin/env python3
"""Generates the graphics, the castle and the text for hollowmere.z80.

Run from this directory:

    python3 gen.py              rewrites the generated parts of hollowmere.z80
    python3 gen.py --preview    also writes sprites.png, a sheet of the sprites

The castle is described below as four floor plans. Each room is placed on
a grid; the generator sizes each room, furnishes it according to its kind
(checking that every door, stair and item can still be reached), and
encodes it for the engine.
"""

import random
import sys
import zlib
import struct

from gfx import *
import objects as O

# ---------------------------------------------------------------------
# Object types. Each has sprites (one, or two for animation), a
# collision box relative to its tile (in world units), flags and some
# things to say when examined.
# ---------------------------------------------------------------------

F_SOLID, F_FLAT, F_ITEM, F_ANIM, F_UP, F_DOWN, F_GATE, F_BELL = 1, 2, 4, 8, 16, 32, 64, 128

TYPES = []      # list of dicts; type id = index + 1


def objtype(name, sprites, box, flags=F_SOLID, texts=(), tiles=(1, 1)):
    TYPES.append(dict(name=name, sprites=sprites, box=box, flags=flags,
                      texts=list(texts), tiles=tiles))


objtype('pillar', [O.pillar()], (1, 1, 7, 7), texts=[
    "SOMEONE HAS SCRATCHED A TALLY OF DAYS INTO THE STONE.",
    "A PILLAR, WORN SMOOTH WHERE HANDS HAVE LEANED."])
objtype('block', [O.block()], (0, 0, 8, 8), texts=["A BLOCK OF GREY STONE, COLD TO THE TOUCH."])
objtype('merlon', [O.low_block()], (0, 0, 8, 8), texts=[
    "BEYOND THE PARAPET, ONLY FOG. NO LAND, NO SKY."])
objtype('table', [O.table()], (0, 0, 8, 8), texts=[
    "THE TABLE IS SET FOR TWO. NEITHER PLACE HAS BEEN USED.",
    "DUST, AND IN THE DUST A FINGER HAS WRITTEN: WAIT."])
objtype('chair', [O.chair()], (2, 2, 6, 6), texts=[
    "A CHAIR, PULLED OUT, AS IF SOMEONE HAS JUST STOOD UP."])
objtype('bed', [O.bed()], (0, 0, 8, 16), tiles=(1, 2), texts=[
    "THE BED IS MADE. THE PILLOW STILL HOLDS THE SHAPE OF A HEAD.",
    "THE SHEETS ARE COLD AND SMELL OF LAVENDER."])
objtype('coffin', [O.coffin()], (1, 0, 7, 16), tiles=(1, 2), texts=[
    "THE LID IS AJAR. YOU DO NOT LOOK INSIDE.",
    "A STONE COFFIN. THE NAME HAS BEEN CHISELLED AWAY."])
objtype('bookcase', [O.bookcase()], (0, 0, 8, 4), texts=[
    "EVERY BOOK HERE HAS THE SAME TITLE: THE LONG WAIT.",
    "A BOOK OF NAMES. THE LAST PAGE HAS BEEN TORN OUT.",
    "ONE BOOK IS MARKED, AT A PAGE ABOUT A RIVER IN WINTER."])
objtype('barrel', [O.barrel()], (1, 1, 7, 7), texts=["EMPTY. IT SMELLS OF OLD WINE."])
objtype('throne', [O.throne()], (1, 1, 7, 7), texts=[
    "THE THRONE IS THICK WITH DUST, EXCEPT FOR THE SEAT."])
objtype('pew', [O.pew()], (0, 2, 16, 7), tiles=(2, 1), texts=[
    "A PRAYER BOOK LIES OPEN AT THE SERVICE FOR THE DEAD."])
objtype('altar', [O.altar()], (1, 1, 15, 7), tiles=(2, 1), texts=[
    "THE CANDLES ON THE ALTAR ARE NEW. NO ONE HAS LIT THEM."])
objtype('well', [O.well()], (0, 0, 8, 8), texts=[
    "YOU DROP A PEBBLE IN. YOU NEVER HEAR IT LAND.",
    "FAR BELOW, SOMETHING GLINTS. OR BLINKS."])
objtype('chest', [O.chest()], (1, 2, 7, 6), texts=[
    "THE CHEST IS FULL OF CHILDREN'S CLOTHES, NEATLY FOLDED.",
    "BLANKETS, AND THE SMELL OF CEDAR."])
objtype('bell', [O.bell()], (1, 2, 7, 6), flags=F_SOLID | F_BELL, texts=[
    "THE GREAT BELL. IT IS STILL SWINGING, VERY SLIGHTLY."])
objtype('clock', [O.clock()], (2, 2, 6, 6), texts=[
    "THE CLOCK HAS STOPPED AT A QUARTER TO THREE."])
objtype('cradle', [O.cradle()], (1, 2, 7, 6), texts=[
    "THE CRADLE ROCKS GENTLY. THERE IS NO DRAUGHT."])
objtype('desk', [O.desk()], (0, 1, 8, 7), texts=[
    "A DIARY. THE LAST ENTRY SAYS ONLY: STILL NO WORD."])
objtype('cauldron', [O.cauldron()], (1, 1, 7, 7), texts=[
    "THE POT IS COLD. SOMETHING WAS COOKED HERE, LONG AGO."])
objtype('carpet', [O.carpet()], (0, 0, 0, 0), flags=F_FLAT)
objtype('stairsup', [O.stairs_up()], (0, 0, 8, 8), flags=F_UP)
objtype('stairsdown', [O.stairs_down()], (1, 1, 7, 7), flags=F_DOWN | F_FLAT)
objtype('gate_ns', [O.gate('ns')], (0, 3, 8, 5), flags=F_SOLID | F_GATE)
objtype('gate_we', [O.gate('we')], (3, 0, 5, 8), flags=F_SOLID | F_GATE)
objtype('armour', [ascii_sprite(O.ARMOUR, 2, bottom=11)], (2, 2, 6, 6), texts=[
    "AN EMPTY SUIT OF ARMOUR. THE VISOR IS DOWN.",
    "THE HELMET IS TURNED TOWARDS YOU. WAS IT BEFORE?"])
objtype('candelabra', [O.candelabra(0), O.candelabra(1)], (3, 3, 5, 5), flags=F_SOLID | F_ANIM, texts=[
    "THE CANDLES BURN, THOUGH NO ONE HAS LIT THEM."])
objtype('candle', [O.candle(0), O.candle(1)], (3, 3, 5, 5), flags=F_SOLID | F_ANIM, texts=[
    "A CANDLE. THE WAX IS STILL SOFT AND WARM."])
objtype('urn', [ascii_sprite(O.URN, 2, bottom=11)], (2, 2, 6, 6), texts=[
    "ASHES. A NAME IS SCRATCHED ON THE LID, TOO FAINT TO READ."])
objtype('grave', [ascii_sprite(O.GRAVE, 2, bottom=10)], (1, 3, 7, 5), texts=[
    "HERE LIES... THE REST IS COVERED IN MOSS.",
    "A NEW STONE, WITH NO NAME ON IT YET.",
    "BELOVED. ALWAYS. THE DATE HAS WORN AWAY."])
objtype('tree', [ascii_sprite(O.TREE, 4, bottom=11)], (2, 2, 6, 6), texts=[
    "A DEAD TREE. TWO INITIALS ARE CARVED IN THE BARK."])
objtype('statue', [ascii_sprite(O.STATUE, 2, bottom=11)], (2, 2, 6, 6), texts=[
    "A STONE ANGEL, WEEPING INTO ITS HANDS."])
objtype('mirror', [ascii_sprite(O.MIRROR, 2, bottom=10)], (2, 3, 6, 5), texts=[
    "THE MIRROR SHOWS THE ROOM BEHIND YOU. IT DOES NOT SHOW YOU."])
objtype('bones', [ascii_sprite(O.BONES, 4, bottom=11)], (1, 2, 7, 6), texts=[
    "BONES, NEATLY STACKED. SOMEONE CARED FOR THEM."])
objtype('rocker', [ascii_sprite(O.HORSE, 4, bottom=11)], (1, 2, 7, 6), texts=[
    "A ROCKING HORSE. IT IS STILL MOVING, VERY SLIGHTLY."])
objtype('item', [], (1, 1, 7, 7), flags=F_ITEM | F_ANIM)

TYPE_ID = {t['name']: i + 1 for i, t in enumerate(TYPES)}

# The eight memories, then the three keys.
ITEMS = ['box', 'locket', 'letter', 'horse', 'shard', 'ring', 'rose', 'painting',
         'key', 'key', 'key']
ITEM_NAMES = ["THE MUSIC BOX", "THE LOCKET", "THE LETTER", "THE WOODEN HORSE",
              "THE MIRROR SHARD", "THE RING", "THE ROSE", "THE PAINTING",
              "THE IRON KEY", "THE BRASS KEY", "THE SILVER KEY"]
MEMORY_TEXT = [
    "A MUSIC BOX. YOU TURN THE KEY AND IT PLAYS THREE NOTES, THEN STOPS. "
    "SOMEONE USED TO HUM THE REST OF THE TUNE. YOU FIND THAT YOU CAN, TOO.",
    "A SILVER LOCKET. INSIDE ARE TWO TINY PORTRAITS: A WOMAN IN GREY, AND "
    "A CHILD. THE CHILD HAS YOUR EYES.",
    "A LETTER, NEVER SENT: 'THE HOUSE IS SO QUIET. I LEAVE A CANDLE IN THE "
    "WINDOW EVERY NIGHT, SO THAT YOU CAN FIND YOUR WAY HOME.'",
    "A WOODEN HORSE, WORN SMOOTH BY SMALL HANDS. YOU KNOW ITS NAME. YOU "
    "GAVE IT ONE, LONG AGO.",
    "A SHARD OF MIRROR. IN IT YOU SEE THE ROOM, THE CANDLES, THE DUST. YOU "
    "TILT IT, LOOKING FOR YOURSELF. YOU ARE NOT THERE.",
    "A WEDDING RING, ENGRAVED 'ALWAYS'. IT WAS LEFT ON A TOMB, AS IF SOMEONE "
    "COULD NOT BEAR TO WEAR IT ANY MORE.",
    "A DRIED ROSE, PRESSED FLAT. SOMEONE PICKED IT FOR A GRAVE, AND THEN "
    "COULD NOT LET IT GO.",
    "A SMALL PAINTING OF A FAMILY ON THE CASTLE STAIRS: A MAN, A WOMAN IN "
    "GREY, AND A CHILD WITH A LANTERN. THE PAINT OF THE CHILD IS NEWER, AS "
    "IF IT WAS ADDED LATER.",
]
KEY_TEXT = [
    "AN IRON KEY, HEAVY AND COLD.",
    "A BRASS KEY, WARM, AS IF SOMEONE HAS JUST PUT IT DOWN.",
    "A SILVER KEY. IT HUMS FAINTLY IN YOUR HAND.",
]
LOCK_NAMES = ["", "IRON", "BRASS", "SILVER"]

INTRO_TEXT = (
    "THE GATES OF HOLLOWMERE STAND OPEN. NO ONE ANSWERS WHEN YOU CALL.|"
    "|YOU DO NOT REMEMBER HOW YOU CAME HERE, ONLY THAT YOU WERE MEANT TO.|"
    "|SOMEWHERE IN THESE EMPTY HALLS ARE EIGHT THINGS THAT WERE ONCE "
    "LOVED. FIND THEM, AND PERHAPS YOU WILL REMEMBER.")
ENDING_TEXT = [
    "YOU TAKE THE ROPE AND PULL. THE GREAT BELL SPEAKS, ONCE, AND THE "
    "WHOLE CASTLE SEEMS TO LISTEN.",
    "FROM THE BELFRY WINDOW YOU SEE A LIGHT IN THE GATEHOUSE, AND A WOMAN "
    "IN GREY, HOLDING A CANDLE. SHE HAS BEEN WAITING FOR A VERY LONG TIME.",
    "YOU REMEMBER NOW. THE RIVER, THE COLD, THE LANTERN GOING OUT. YOU "
    "WERE NEVER LOST. YOU WERE ONLY LATE.",
    "YOU GO DOWN TO MEET HER.||AND HOLLOWMERE, AT LAST, IS QUIET.||"
    "        THE END",
]

WALL_TEXTS = {
    'window': ["OUTSIDE, FOG. THE MOON HAS NOT MOVED ALL NIGHT.",
               "YOUR BREATH MISTS THE GLASS. SOMETHING WRITES IN IT."],
    'moon': ["A FULL MOON, WHITE AND VERY STILL."],
    'portrait': ["A WOMAN IN GREY. SHE LOOKS AS IF SHE IS LISTENING.",
                 "A MAN IN BLACK. HIS EYES HAVE BEEN PAINTED CLOSED.",
                 "A CHILD WITH A LANTERN. THE FACE IS FAMILIAR."],
    'torch': ["THE FLAME DOES NOT FLICKER. IT GIVES NO WARMTH."],
    'banner': ["A BANNER: A SWAN ABOVE A BROKEN KEY."],
    'shield': ["AN OLD SHIELD. THE MOTTO READS: WE WAIT."],
    'web': ["A COBWEB. THE SPIDER IS LONG GONE."],
    'hearth': ["THE ASHES IN THE HEARTH ARE STILL WARM."],
    'crack': ["A CRACK IN THE WALL. COLD AIR BREATHES THROUGH IT."],
}
NOTHING_TEXTS = [
    "THERE IS NOTHING HERE BUT DUST.",
    "YOU HEAR ONLY YOUR OWN BREATHING.",
    "NOTHING. THE SILENCE PRESSES IN.",
    "YOUR LANTERN'S LIGHT SHOWS NOTHING NEW.",
]
AMBIENT_TEXTS = [
    "SOMEWHERE, A DOOR CLOSES.",
    "FOOTSTEPS OVERHEAD. THEN NOTHING.",
    "A DRAUGHT, AND THE SMELL OF ROSES.",
    "YOU THINK YOU HEAR YOUR NAME.",
    "A CLOCK STRIKES, FAR AWAY.",
    "THE CASTLE SETTLES AND CREAKS.",
    "SOMEONE IS HUMMING. IT STOPS.",
    "WATER DRIPS SOMEWHERE IN THE DARK.",
    "A CHILD LAUGHS, SOMEWHERE BELOW.",
    "THE AIR GROWS COLD, THEN WARM AGAIN.",
    "A WINDOW RATTLES IN ITS FRAME.",
    "YOU FEEL YOU ARE BEING WATCHED.",
    "A SOFT KNOCK. THEN SILENCE.",
    "YOUR LANTERN DIMS, THEN STEADIES.",
    "SOMEWHERE, SOMEONE IS WAITING.",
    "DUST FALLS FROM THE CEILING.",
]
GHOST_TEXTS = [
    "SHE IS GONE.",
    "WAS SOMEONE THERE?",
    "A CHILL PASSES THROUGH YOU.",
    "SHE FADES LIKE BREATH ON GLASS.",
]
MSG = {
    'locked': "LOCKED. IT WILL NEED THE {} KEY.",
    'unlock': "THE {} KEY TURNS. THE GATE RISES.",
    'sealed': "THE DOOR WILL NOT OPEN. NOT YET. NOT UNTIL YOU REMEMBER.",
    'unseal': "THE DOOR OPENS BY ITSELF.",
    'belfry': "THE BELL ROPE HANGS WITHIN REACH.",
    'allmem': "SOMEWHERE HIGH ABOVE, A GREAT BELL BEGINS TO SWING.",
    'stairup': "YOU CLIMB THE STAIRS.",
    'stairdown': "YOU GO DOWN THE STAIRS.",
    'count': "OF 8 REMEMBERED",
}
FLOOR_NAMES = ["THE CRYPTS", "THE GROUND FLOOR", "THE UPPER FLOOR", "THE TOWER"]

# ---------------------------------------------------------------------
# The castle. Four floors, each a grid of up to 7x7 rooms. Letters are
# rooms (see ROOM_KINDS); '-' and '|' are doorways between neighbours;
# a digit is a locked door (1 iron, 2 brass, 3 silver, 4 sealed until
# all eight memories are found).
# ---------------------------------------------------------------------

FLOORS = [
    # 0: the crypts
    [
        "O-C-X-Z-X-C-O",
        "|   | | |   |",
        "X   C-X-C   X",
        "|   |   |   |",
        "C-C-C   C-C-C",
        "  |       |  ",
        "  V-C-W-C-O  ",
    ],
    # 1: the ground floor
    [
        "O-O P1P T-T  ",
        "|   |     |  ",
        "R-R-C-H-H-H-D",
        "    |     | |",
        "L-L-C-Y-Y-C D",
        "| |   | |   |",
        "S L-C-Y-Y-C-K",
        "  2   |   | |",
        "  Z   C   C K",
        "      |   | |",
        "  A-C-G-C-C V",
    ],
    # 2: the upper floor
    [
        "B-C-W-W-W-C-B",
        "|   |     |  ",
        "N   C     I  ",
        "|   |     |  ",
        "C-C-C-M-C-C-B",
        "  |   |   |  ",
        "  S   B   C-B",
        "  |       3  ",
        "  Z       E  ",
    ],
    # 3: the tower
    [
        "             ",
        "             ",
        "          U  ",
        "          4  ",
        "        E-E-E",
        "        |   |",
        "        E-t-E",
        "          |  ",
        "          t  ",
    ],
]

# Stairs: (lower room, upper room), each as (floor, x, y).
STAIRS = [
    ((0, 3, 0), (1, 3, 0)),     # crypt stair up to the chapel
    ((1, 1, 4), (2, 1, 4)),     # stair hall up to the upper floor
    ((2, 5, 4), (3, 5, 4)),     # tower door up to the tower stair
]

START = (1, 3, 5)

ITEM_PLACES = [
    (2, 3, 2),  # music box: music room
    (0, 6, 0),  # locket: ossuary
    (2, 1, 3),  # letter: study
    (2, 0, 1),  # horse: nursery
    (1, 4, 0),  # mirror shard: throne room
    (0, 3, 1),  # ring: tomb
    (1, 0, 1),  # rose: graveyard
    (3, 5, 3),  # painting: bell-ringer's room
    (1, 6, 4),  # iron key: kitchen
    (0, 3, 3),  # brass key: drowned chapel
    (2, 6, 3),  # silver key: a bedchamber
]

# Names for each kind of room, by floor; rooms take them in turn.
NAMES = {
    (0, 'Z'): ["THE CRYPT STAIR"],
    (0, 'X'): ["THE FIRST TOMB", "THE LADY'S TOMB", "A SEALED TOMB", "THE OLD TOMB",
               "THE KNIGHTS' TOMB", "A NAMELESS TOMB"],
    (0, 'O'): ["THE OSSUARY", "THE CHARNEL HOUSE", "THE BONE CHAPEL"],
    (0, 'V'): ["THE WINE CELLAR"],
    (0, 'W'): ["THE DROWNED CHAPEL"],
    (0, 'C'): ["A DAMP TUNNEL", "THE BONE ROAD", "A DRIPPING TUNNEL", "THE LOW PASSAGE",
               "A SILENT TUNNEL", "THE WORM WAY", "A COLD TUNNEL", "THE BLIND PASSAGE",
               "A NARROW TUNNEL", "THE DEAD END", "A DARK TUNNEL", "THE LONG DARK"],
    (1, 'G'): ["THE GATEHOUSE"],
    (1, 'Y'): ["THE COURTYARD", "THE WELL COURT", "THE SOUTH COURT", "THE EMPTY COURT"],
    (1, 'O'): ["THE OLD ORCHARD", "THE WITHERED GARDEN"],
    (1, 'R'): ["THE GRAVEYARD", "THE CHURCHYARD"],
    (1, 'P'): ["THE CHAPEL", "THE CHANCEL"],
    (1, 'T'): ["THE THRONE ROOM", "THE AUDIENCE HALL"],
    (1, 'H'): ["THE GREAT HALL", "THE HIGH TABLE", "THE MINSTRELS' END"],
    (1, 'D'): ["THE BANQUETING HALL", "THE SUPPER ROOM"],
    (1, 'K'): ["THE KITCHEN", "THE SCULLERY", "THE BAKEHOUSE"],
    (1, 'V'): ["THE PANTRY"],
    (1, 'A'): ["THE ARMOURY"],
    (1, 'L'): ["THE LIBRARY", "THE READING ROOM", "THE MAP ROOM", "THE LIBRARY STACKS"],
    (1, 'S'): ["THE STEWARD'S ROOM"],
    (1, 'Z'): ["THE STAIR HALL"],
    (1, 'C'): ["A COLD PASSAGE", "THE LANTERN WALK", "A LONG CORRIDOR", "THE SERVANTS' WAY",
               "A DRAUGHTY PASSAGE", "THE WEST PASSAGE", "A NARROW PASSAGE", "THE EAST PASSAGE",
               "THE GUARD WALK", "A DIM CORRIDOR", "THE CLOISTER"],
    (2, 'Z'): ["THE UPPER STAIR"],
    (2, 'B'): ["THE BLUE BEDCHAMBER", "THE LORD'S CHAMBER", "THE GUEST ROOM",
               "THE LADY'S CHAMBER", "THE EMPTY BEDROOM"],
    (2, 'N'): ["THE NURSERY"],
    (2, 'W'): ["THE LONG GALLERY", "THE GALLERY OF FACES", "THE GALLERY'S END"],
    (2, 'M'): ["THE MUSIC ROOM"],
    (2, 'S'): ["THE STUDY"],
    (2, 'I'): ["THE LINEN ROOM"],
    (2, 'E'): ["THE TOWER DOOR"],
    (2, 'C'): ["THE UPPER LANDING", "A QUIET CORRIDOR", "THE PORTRAIT WALK",
               "A CREAKING PASSAGE", "THE NIGHT CORRIDOR", "A NARROW LANDING",
               "THE WEST LANDING", "A HUSHED PASSAGE", "THE EAST LANDING"],
    (3, 't'): ["THE TOWER STAIR", "THE BELL-RINGER'S ROOM"],
    (3, 'E'): ["THE WEST TURRET", "THE BATTLEMENTS", "THE EAST TURRET",
               "THE WINDY WALK", "THE NORTH WALK"],
    (3, 'U'): ["THE BELFRY"],
}

# ---------------------------------------------------------------------
# Rooms.
# ---------------------------------------------------------------------

DIRS = {'N': (0, -1), 'E': (1, 0), 'S': (0, 1), 'W': (-1, 0)}
OPP = {'N': 'S', 'S': 'N', 'E': 'W', 'W': 'E'}

FLOOR_STYLES = ['flag', 'check', 'wood', 'earth', 'dark']

BRIGHT = 0x40
BLUE, RED, MAGENTA, GREEN, CYAN, YELLOW, WHITE = 1, 2, 3, 4, 5, 6, 7


class Room:
    def __init__(self, floor, x, y, kind):
        self.floor, self.x, self.y, self.kind = floor, x, y, kind
        self.exits = {}         # dir -> (room, lock)
        self.objs = []          # (type name, tx, ty, extra)
        self.walls = {}         # ('n'|'w', index) -> kind
        self.stairs = None      # ('up'|'down', target room)
        self.item = None
        self.id = None

    def __repr__(self):
        return "%s%s" % (self.kind, (self.floor, self.x, self.y))


def parse_floors():
    rooms = {}
    for f, plan in enumerate(FLOORS):
        for r in range(0, len(plan), 2):
            line = plan[r]
            for c in range(0, len(line), 2):
                ch = line[c]
                if ch != ' ':
                    rooms[(f, c // 2, r // 2)] = Room(f, c // 2, r // 2, ch)

        def link(a, b, d, ch):
            if a not in rooms or b not in rooms:
                raise SystemExit("bad doorway on floor %d between %s and %s" % (f, a, b))
            lock = int(ch) if ch.isdigit() else 0
            rooms[a].exits[d] = (rooms[b], lock)
            rooms[b].exits[OPP[d]] = (rooms[a], lock)
        for r, line in enumerate(plan):
            for c, ch in enumerate(line):
                if ch == ' ' or (r % 2 == 0 and c % 2 == 0):
                    continue
                if r % 2 == 0:
                    link((f, (c - 1) // 2, r // 2), (f, (c + 1) // 2, r // 2), 'E', ch)
                else:
                    link((f, c // 2, (r - 1) // 2), (f, c // 2, (r + 1) // 2), 'S', ch)
    return rooms


# Size (in tiles), colour and floor style of each kind of room.
KINDS = {
    'G': ((5, 5), WHITE | BRIGHT, 'flag'),
    'Y': ((7, 7), WHITE, 'flag'),
    'O': ((7, 7), GREEN | BRIGHT, 'earth'),
    'R': ((7, 7), GREEN, 'earth'),
    'P': ((7, 7), YELLOW, 'check'),
    'T': ((7, 7), YELLOW | BRIGHT, 'check'),
    'H': ((7, 7), WHITE | BRIGHT, 'check'),
    'D': ((7, 5), YELLOW, 'wood'),
    'K': ((5, 5), WHITE, 'flag'),
    'V': ((5, 5), YELLOW, 'flag'),
    'A': ((5, 5), CYAN, 'flag'),
    'L': ((7, 5), GREEN | BRIGHT, 'wood'),
    'S': ((5, 5), CYAN | BRIGHT, 'wood'),
    'Z': ((5, 5), WHITE | BRIGHT, 'flag'),
    'X': ((5, 5), MAGENTA | BRIGHT, 'dark'),
    'W': ((7, 7), CYAN, 'dark'),
    'B': ((5, 5), CYAN | BRIGHT, 'wood'),
    'N': ((5, 5), YELLOW | BRIGHT, 'wood'),
    'M': ((7, 5), MAGENTA | BRIGHT, 'check'),
    'I': ((5, 3), WHITE, 'wood'),
    'E': ((5, 5), WHITE | BRIGHT, 'flag'),
    't': ((5, 5), MAGENTA | BRIGHT, 'flag'),
    'U': ((5, 5), WHITE | BRIGHT, 'flag'),
    'C': (None, WHITE, 'flag'),
}
# Per-floor overrides.
FLOOR_KINDS = {
    (0, 'C'): (None, BLUE | BRIGHT, 'dark'),
    (0, 'O'): ((5, 5), WHITE, 'dark'),
    (0, 'V'): ((5, 5), MAGENTA, 'dark'),
    (0, 'Z'): ((5, 5), BLUE | BRIGHT, 'dark'),
    (2, 'C'): (None, CYAN, 'wood'),
    (2, 'W'): ((7, 3), YELLOW | BRIGHT, 'check'),
    (2, 'E'): ((5, 5), WHITE, 'flag'),
    (3, 'E'): ((5, 5), WHITE, 'flag'),
}


def kind_info(room):
    return FLOOR_KINDS.get((room.floor, room.kind), KINDS[room.kind])


def door_tile(room, d):
    """The tile inside the room that a doorway on side d opens from."""
    W, D = room.W, room.D
    return {'N': (W // 2, 0), 'S': (W // 2, D - 1), 'W': (0, D // 2), 'E': (W - 1, D // 2)}[d]


class Furnisher:
    """Places objects in a room, keeping every doorway, stair and item
    reachable from every other."""

    def __init__(self, room, rng):
        self.r, self.rng = room, rng
        self.used = set()       # tiles with something on them
        self.solid = set()      # tiles that block walking
        self.keep = set()       # tiles that must stay free
        self.targets = []       # tiles that must stay reachable
        for d in room.exits:
            tx, ty = door_tile(room, d)
            self.keep.add((tx, ty))
            self.targets.append((tx, ty))
            inward = {'N': (0, 1), 'S': (0, -1), 'W': (1, 0), 'E': (-1, 0)}[d]
            self.keep.add((tx + inward[0], ty + inward[1]))

    def free(self, tx, ty):
        return 0 <= tx < self.r.W and 0 <= ty < self.r.D and (tx, ty) not in self.used

    def tiles_of(self, name, tx, ty):
        tw, td = TYPES[TYPE_ID[name] - 1]['tiles']
        return [(tx + i, ty + j) for i in range(tw) for j in range(td)]

    def reachable(self, solid):
        if not self.targets:
            return True
        start = self.targets[0]
        seen = {start}
        todo = [start]
        while todo:
            x, y = todo.pop()
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                q = (x + dx, y + dy)
                if 0 <= q[0] < self.r.W and 0 <= q[1] < self.r.D and q not in seen and q not in solid:
                    seen.add(q)
                    todo.append(q)
        return all(t in seen for t in self.targets)

    def place(self, name, tx, ty, extra=0, force=False):
        ts = self.tiles_of(name, tx, ty)
        if not force:
            if any(not self.free(*t) or t in self.keep for t in ts):
                return False
        flags = TYPES[TYPE_ID[name] - 1]['flags']
        blocks = flags & F_SOLID and not flags & F_GATE
        if blocks and not force:
            if not self.reachable(self.solid | set(ts)):
                return False
        self.used |= set(ts)
        if blocks:
            self.solid |= set(ts)
        self.r.objs.append((name, tx, ty, extra))
        return True

    def some(self, name, spots, n=None):
        spots = [s for s in spots]
        self.rng.shuffle(spots)
        k = 0
        for (x, y) in spots:
            if n is not None and k >= n:
                break
            if self.place(name, x, y):
                k += 1
        return k

    def free_tile(self, prefer_inner=True):
        W, D = self.r.W, self.r.D
        cands = [(x, y) for x in range(W) for y in range(D)
                 if self.free(x, y) and (x, y) not in self.keep]
        self.rng.shuffle(cands)
        if prefer_inner:
            cands.sort(key=lambda t: min(t[0], t[1], W - 1 - t[0], D - 1 - t[1]) == 0)
        return cands[0] if cands else None

    # Convenient sets of tiles.
    def perimeter(self):
        W, D = self.r.W, self.r.D
        return [(x, y) for x in range(W) for y in range(D) if x in (0, W - 1) or y in (0, D - 1)]

    def back(self):
        W, D = self.r.W, self.r.D
        return [(x, 0) for x in range(W)] + [(0, y) for y in range(1, D)]

    def front(self):
        W, D = self.r.W, self.r.D
        return [(x, D - 1) for x in range(W)] + [(W - 1, y) for y in range(D - 1)]

    def corners(self):
        W, D = self.r.W, self.r.D
        return [(0, 0), (W - 1, 0), (0, D - 1), (W - 1, D - 1)]

    def inner(self):
        W, D = self.r.W, self.r.D
        return [(x, y) for x in range(1, W - 1) for y in range(1, D - 1)]


def furnish(room, rng):
    f = Furnisher(room, rng)
    W, D = room.W, room.D
    k = room.kind
    fl = room.floor

    # Locked doors get gates.
    for d, (other, lock) in room.exits.items():
        if lock:
            tx, ty = door_tile(room, d)
            f.place('gate_ns' if d in 'NS' else 'gate_we', tx, ty, lock, force=True)

    # Stairs go against the back (north) wall, with a free tile in front.
    if room.stairs:
        way, target = room.stairs
        xs = [x for x in range(W) if (x, 0) not in f.keep and (x, 1) not in f.keep]
        xs.sort(key=lambda x: -x)
        tx = xs[0]
        f.place('stairsup' if way == 'up' else 'stairsdown', tx, 0, 0, force=True)
        f.keep.add((tx, 1))
        f.targets.append((tx, 1))
        room.stair_tile = tx
        if way == 'up':
            room.walls[('n', tx)] = 'arch'

    if room.item is not None:
        t = f.free_tile()
        f.place('item', t[0], t[1], room.item, force=True)
        f.targets.append(t)

    def walls(kinds, n):
        segs = [('n', i) for i in range(W)] + [('w', i) for i in range(D)]
        segs = [s for s in segs if s not in room.walls]
        rng.shuffle(segs)
        for s in segs[:n]:
            room.walls[s] = rng.choice(kinds)

    if k == 'C':
        if fl == 0:
            f.some(rng.choice(['urn', 'candle', 'bones']), f.perimeter(), rng.randint(0, 2))
            walls(['crack', 'web', 'torch', 'crack'], rng.randint(1, 3))
        elif fl == 2:
            f.some(rng.choice(['armour', 'candelabra', 'statue', 'clock', 'chest']), f.perimeter(), rng.randint(1, 2))
            walls(['portrait', 'window', 'moon', 'torch', 'portrait'], rng.randint(2, 4))
        else:
            f.some(rng.choice(['armour', 'candelabra', 'urn', 'statue', 'barrel']), f.perimeter(), rng.randint(1, 3))
            walls(['torch', 'window', 'banner', 'shield', 'portrait', 'web', 'moon'], rng.randint(2, 4))
    elif k == 'G':
        f.some('armour', [(1, 1), (3, 1)])
        f.some('block', [(0, 4), (4, 4), (0, 3), (4, 3)], 2)
        walls(['shield', 'torch', 'banner'], 4)
    elif k == 'Y':
        if not f.place('well', 3, 3):
            f.place('statue', 3, 3)
        f.some('tree', [(1, 1), (5, 1), (1, 5), (5, 5)], rng.randint(1, 2))
        f.some('statue', [(1, 3), (5, 3), (3, 1), (3, 5)], rng.randint(0, 1))
        walls(['window', 'web', 'crack', 'moon'], 4)
    elif k == 'O':
        if fl == 0:
            f.some('bones', f.perimeter(), 4)
            f.some('urn', f.perimeter(), 3)
            f.some('candle', f.inner(), 1)
            walls(['web', 'crack'], 4)
        else:
            f.some('tree', [(x, y) for x in (1, 3, 5) for y in (1, 3, 5)], 6)
            f.some('statue', f.perimeter(), 1)
            walls(['crack', 'web', 'moon'], 3)
    elif k == 'R':
        f.some('grave', [(x, y) for x in (1, 2, 4, 5) for y in (1, 3, 5)], 8)
        f.some('tree', f.corners(), 1)
        walls(['crack', 'moon', 'web'], 3)
    elif k == 'P':
        f.some('altar', [(2, 1), (3, 1)], 1)
        f.some('pew', [(x, y) for x in (1, 4) for y in (3, 5)])
        f.some('candle', [(1, 1), (5, 1), (0, 2), (6, 2)], 3)
        walls(['window', 'window', 'moon'], 6)
    elif k == 'T':
        f.place('throne', 3, 1)
        for y in range(2, 7):
            f.place('carpet', 3, y)
        f.some('pillar', [(1, 1), (5, 1), (1, 4), (5, 4)])
        f.some('candelabra', [(2, 1), (4, 1)])
        walls(['banner', 'shield', 'banner', 'window'], 6)
    elif k == 'H':
        f.some('pillar', [(1, 1), (5, 1), (1, 5), (5, 5)])
        for x in (2, 3, 4):
            f.place('table', x, 3)
        f.some('chair', [(2, 2), (4, 2), (2, 4), (4, 4), (3, 2), (3, 4)], 4)
        f.some('candelabra', [(3, 1), (3, 5)], 1)
        walls(['banner', 'hearth', 'shield', 'window', 'torch'], 6)
    elif k == 'D':
        for x in (2, 3, 4):
            f.place('table', x, 2)
        f.some('chair', [(x, y) for x in (2, 3, 4) for y in (1, 3)], 4)
        f.some('candelabra', [(0, 0), (6, 0), (0, 4), (6, 4)], 2)
        walls(['portrait', 'hearth', 'window'], 5)
    elif k == 'K':
        f.some('table', [(2, 2), (2, 3)], 1)
        f.some('cauldron', [(1, 0), (3, 0), (0, 1)], 1)
        f.some('barrel', f.perimeter(), rng.randint(2, 3))
        walls(['hearth', 'web', 'window'], 3)
    elif k == 'V':
        f.some('barrel', f.perimeter(), rng.randint(5, 8))
        f.some('barrel', f.inner(), 2)
        walls(['web', 'crack'], 2)
    elif k == 'A':
        f.some('armour', f.back(), 4)
        f.some('block', f.inner(), 1)
        walls(['shield', 'shield', 'banner'], 5)
    elif k == 'L':
        f.some('bookcase', [(x, 0) for x in range(W)], W)
        f.some('bookcase', [(x, 2) for x in (1, 2, 4, 5)])
        f.some('desk', [(x, 4) for x in (1, 5)], 1)
        f.some('candle', f.front(), 1)
        walls(['window', 'portrait'], 2)
    elif k == 'S':
        f.place('desk', 2, 2)
        f.some('chair', [(2, 3), (1, 2)], 1)
        f.some('bookcase', [(x, 0) for x in range(W)], 2)
        f.some('clock', f.perimeter(), 1)
        f.some('candle', f.perimeter(), 1)
        walls(['portrait', 'window', 'hearth'], 3)
    elif k == 'Z':
        f.some('candelabra', f.perimeter(), 2)
        f.some('statue' if fl else 'urn', f.perimeter(), 1)
        walls(['portrait', 'torch', 'window'] if fl else ['web', 'crack', 'torch'], 3)
    elif k == 'X':
        f.some('coffin', [(1, 1), (3, 1), (1, 2), (3, 2)], rng.randint(1, 2))
        f.some('candle', f.perimeter(), 2)
        f.some('urn', f.perimeter(), 1)
        f.some('pillar', f.corners(), 1)
        walls(['web', 'crack', 'torch'], 3)
    elif k == 'W':
        f.some('pillar', [(1, 1), (5, 1), (1, 5), (5, 5), (1, 3), (5, 3)], 5)
        f.some('pew', [(2, 4), (2, 5)], 1)
        f.some('altar', [(2, 1)], 1)
        walls(['crack', 'web', 'window'], 5)
    elif k == 'B':
        f.some('bed', [(1, 1), (3, 1), (1, 0), (3, 0)], 1)
        f.some('chest', f.front(), 1)
        f.some(rng.choice(['mirror', 'clock', 'candle']), f.perimeter(), 1)
        f.some('candle', f.perimeter(), 1)
        walls(['portrait', 'window', 'hearth', 'moon'], 3)
    elif k == 'N':
        f.some('cradle', [(1, 1), (3, 1)], 1)
        f.some('rocker', [(1, 3), (3, 3), (2, 3)], 1)
        f.some('chest', f.perimeter(), 1)
        f.some('candle', f.perimeter(), 1)
        walls(['window', 'portrait', 'moon'], 3)
    elif k == 'M':
        f.some('clock', [(0, 0), (6, 0)], 1)
        f.some('chair', [(x, y) for x in (2, 3, 4) for y in (2, 3)], 4)
        f.some('mirror', f.perimeter(), 1)
        f.some('candelabra', f.corners(), 2)
        walls(['portrait', 'window', 'moon'], 4)
    elif k == 'I':
        f.some('chest', f.perimeter(), 4)
        walls(['web', 'window'], 2)
    elif k == 'E':
        if fl == 3:
            f.some('merlon', f.front(), rng.randint(2, 4))
            walls(['moon', 'window', 'crack'], 3)
        else:
            f.some('armour', f.perimeter(), 2)
            walls(['torch', 'shield'], 3)
    elif k == 't':
        f.some('candle', f.perimeter(), 2)
        f.some('chest', f.perimeter(), 1)
        walls(['window', 'moon', 'web'], 3)
    elif k == 'U':
        f.place('bell', 2, 2)
        f.some('candle', f.corners(), 2)
        walls(['moon', 'window'], 5)
    for s, kind in list(room.walls.items()):
        pass


def build():
    rooms = parse_floors()
    order = sorted(rooms.values(), key=lambda r: (r.floor, r.y, r.x))
    for i, r in enumerate(order):
        r.id = i
    assert len(order) < 255
    for lo, hi in STAIRS:
        rooms[lo].stairs = ('up', rooms[hi])
        rooms[hi].stairs = ('down', rooms[lo])
    for i, p in enumerate(ITEM_PLACES):
        assert rooms[p].item is None
        rooms[p].item = i
    counters = {}
    rng = random.Random(1987)
    for r in order:
        size, attr, style = kind_info(r)
        if size is None:
            ds = set(r.exits)
            if ds <= {'E', 'W'}:
                size = (7, 3)
            elif ds <= {'N', 'S'}:
                size = (3, 7)
            else:
                size = (5, 5)
        r.W, r.D = size
        r.attr, r.style = attr, FLOOR_STYLES.index(style)
        names = NAMES[(r.floor, r.kind)]
        n = counters.get((r.floor, r.kind), 0)
        counters[(r.floor, r.kind)] = n + 1
        r.name = names[n % len(names)]
        for d in r.exits:
            if d in 'NW':
                tx, ty = door_tile(r, d)
                r.walls[('n', tx) if d == 'N' else ('w', ty)] = 'arch'
    for r in order:
        furnish(r, rng)
    return rooms, order


# ---------------------------------------------------------------------
# Text.
# ---------------------------------------------------------------------

def wrap(text, width):
    lines = []
    for para in text.split('|'):
        line = ''
        for w in para.split(' '):
            if not w:
                continue
            if line and len(line) + 1 + len(w) > width:
                lines.append(line)
                line = w
            else:
                line = line + ' ' + w if line else w
        lines.append(line)
    return lines


class Texts:
    def __init__(self):
        self.items = []
        self.index = {}

    def add(self, text, width=32, maxlines=2):
        lines = wrap(text, width)
        assert len(lines) <= maxlines, (text, lines)
        s = '|'.join(lines)
        for c in s:
            assert c in "|" or 32 <= ord(c) <= 90, (c, text)
        if s in self.index:
            return self.index[s]
        self.items.append(s)
        self.index[s] = len(self.items) - 1
        return self.index[s]


# ---------------------------------------------------------------------
# Output.
# ---------------------------------------------------------------------

class Out:
    def __init__(self):
        self.lines = []

    def __call__(self, s=''):
        self.lines.append(s)

    def bytes(self, bs, per=16):
        bs = list(bs)
        for i in range(0, len(bs), per):
            self("    db " + ", ".join("0x%02x" % (b & 0xff) for b in bs[i:i + per]))

    def sprite(self, label, spr):
        self(label + ":")
        enc = spr.encode()
        self("    db %d, %d, 0x%02x, 0x%02x" % tuple(enc[:4]))
        self.bytes(enc[4:])


def dstring(s):
    assert '"' not in s and '\\' not in s
    return '    ds "%s" ; db 0' % s


FONT = {
' ': ["........"]*8,
'!': ["........","...XX...","...XX...","...XX...","...XX...","........","...XX...","........"],
"'": ["........","...XX...","...XX...","..XX....","........","........","........","........"],
',': ["........","........","........","........","........","...XX...","...XX...","..XX...."],
'-': ["........","........","........","........",".XXXXXX.","........","........","........"],
'.': ["........","........","........","........","........","...XX...","...XX...","........"],
'/': ["........",".....XX.","....XX..","...XX...","..XX....",".XX.....","........","........"],
'0': ["........","..XXXX..",".XX..XX.",".XX.XXX.",".XXX.XX.",".XX..XX.","..XXXX..","........"],
'1': ["........","...XX...","..XXX...","...XX...","...XX...","...XX...",".XXXXXX.","........"],
'2': ["........","..XXXX..",".XX..XX.","....XX..","...XX...","..XX....",".XXXXXX.","........"],
'3': ["........","..XXXX..",".XX..XX.","....XX..",".....XX.",".XX..XX.","..XXXX..","........"],
'4': ["........","....XX..","...XXX..","..XXXX..",".XX.XX..",".XXXXXX.","....XX..","........"],
'5': ["........",".XXXXXX.",".XX.....",".XXXXX..",".....XX.",".XX..XX.","..XXXX..","........"],
'6': ["........","..XXXX..",".XX.....",".XXXXX..",".XX..XX.",".XX..XX.","..XXXX..","........"],
'7': ["........",".XXXXXX.",".....XX.","....XX..","...XX...","...XX...","...XX...","........"],
'8': ["........","..XXXX..",".XX..XX.","..XXXX..",".XX..XX.",".XX..XX.","..XXXX..","........"],
'9': ["........","..XXXX..",".XX..XX.",".XX..XX.","..XXXXX.",".....XX.","..XXXX..","........"],
':': ["........","........","...XX...","...XX...","........","...XX...","...XX...","........"],
'?': ["........","..XXXX..",".XX..XX.","....XX..","...XX...","........","...XX...","........"],
'A': ["........","..XXXX..",".XX..XX.",".XX..XX.",".XXXXXX.",".XX..XX.",".XX..XX.","........"],
'B': ["........",".XXXXX..",".XX..XX.",".XXXXX..",".XX..XX.",".XX..XX.",".XXXXX..","........"],
'C': ["........","..XXXX..",".XX..XX.",".XX.....",".XX.....",".XX..XX.","..XXXX..","........"],
'D': ["........",".XXXX...",".XX.XX..",".XX..XX.",".XX..XX.",".XX.XX..",".XXXX...","........"],
'E': ["........",".XXXXXX.",".XX.....",".XXXXX..",".XX.....",".XX.....",".XXXXXX.","........"],
'F': ["........",".XXXXXX.",".XX.....",".XXXXX..",".XX.....",".XX.....",".XX.....","........"],
'G': ["........","..XXXX..",".XX.....",".XX.XXX.",".XX..XX.",".XX..XX.","..XXXXX.","........"],
'H': ["........",".XX..XX.",".XX..XX.",".XXXXXX.",".XX..XX.",".XX..XX.",".XX..XX.","........"],
'I': ["........",".XXXXXX.","...XX...","...XX...","...XX...","...XX...",".XXXXXX.","........"],
'J': ["........","....XXX.",".....XX.",".....XX.",".....XX.",".XX..XX.","..XXXX..","........"],
'K': ["........",".XX..XX.",".XX.XX..",".XXXX...",".XXXX...",".XX.XX..",".XX..XX.","........"],
'L': ["........",".XX.....",".XX.....",".XX.....",".XX.....",".XX.....",".XXXXXX.","........"],
'M': ["........",".XX...XX",".XXX.XXX",".XXXXXXX",".XX.X.XX",".XX...XX",".XX...XX","........"],
'N': ["........",".XX..XX.",".XXX.XX.",".XXXXXX.",".XX.XXX.",".XX..XX.",".XX..XX.","........"],
'O': ["........","..XXXX..",".XX..XX.",".XX..XX.",".XX..XX.",".XX..XX.","..XXXX..","........"],
'P': ["........",".XXXXX..",".XX..XX.",".XX..XX.",".XXXXX..",".XX.....",".XX.....","........"],
'Q': ["........","..XXXX..",".XX..XX.",".XX..XX.",".XX.XXX.",".XX.XX..","..XX.XX.","........"],
'R': ["........",".XXXXX..",".XX..XX.",".XX..XX.",".XXXXX..",".XX.XX..",".XX..XX.","........"],
'S': ["........","..XXXX..",".XX.....","..XXXX..",".....XX.",".XX..XX.","..XXXX..","........"],
'T': ["........",".XXXXXX.","...XX...","...XX...","...XX...","...XX...","...XX...","........"],
'U': ["........",".XX..XX.",".XX..XX.",".XX..XX.",".XX..XX.",".XX..XX.","..XXXX..","........"],
'V': ["........",".XX..XX.",".XX..XX.",".XX..XX.",".XX..XX.","..XXXX..","...XX...","........"],
'W': ["........",".XX...XX",".XX...XX",".XX.X.XX",".XXXXXXX",".XXX.XXX",".XX...XX","........"],
'X': ["........",".XX..XX.","..XXXX..","...XX...","..XXXX..",".XX..XX.",".XX..XX.","........"],
'Y': ["........",".XX..XX.",".XX..XX.","..XXXX..","...XX...","...XX...","...XX...","........"],
'Z': ["........",".XXXXXX.","....XX..","...XX...","..XX....",".XX.....",".XXXXXX.","........"],
# '<' is a key icon, '=' a memory (a small flame), '>' a filled diamond.
'<': ["........","........",".XX.....","X..XXXXX",".XX..X.X","........","........","........"],
'=': ["...X....","..XX....","..XXX...",".XXXXX..",".XX.XX..","..XXX...","........","........"],
'>': ["...X....","..XXX...",".XXXXX..","XXXXXXX.",".XXXXX..","..XXX...","...X....","........"],
}


def font_bytes():
    out = []
    for code in range(32, 91):
        rows = FONT.get(chr(code), ["........"] * 8)
        for r in rows:
            v = 0
            for c in r:
                v = v * 2 + (c == 'X')
            out.append(v)
    return out


def generate():
    rooms, order = build()
    texts = Texts()
    consts, data = Out(), Out()

    consts("// Object types.")
    for i, t in enumerate(TYPES):
        consts("const T_%s = %d" % (t['name'].upper(), i + 1))
    consts("const NTYPES = %d" % len(TYPES))
    consts("const NROOMS = %d" % len(order))
    consts("const START_ROOM = %d" % rooms[START].id)
    consts("const BELFRY = %d" % [r.id for r in order if r.kind == 'U'][0])
    consts("const NITEMS = %d" % len(ITEMS))
    consts("const WALL_H = %d" % WALL_H)
    for i, k in enumerate(WALL_KINDS):
        consts("const W_%s = %d" % (k.upper(), i))
    consts("const PLAYER_ROWS = %d" % O.wanderer(0, 0).h)
    consts("const NAMBIENT = %d" % len(AMBIENT_TEXTS))
    consts("const NGHOSTTXT = %d" % len(GHOST_TEXTS))
    consts("const NNOTHING = %d" % len(NOTHING_TEXTS))

    # --- Sprites.
    data("// Object types: flags, first text, number of texts, sprite,")
    data("// second animation frame (or 0), collision box x0, y0, x1, y1.")
    data("types:")
    for i, t in enumerate(TYPES):
        tid = [texts.add(s) for s in t['texts']]
        first = tid[0] if tid else 0
        for j, x in enumerate(tid):
            assert x == first + j
        s1 = 'spr_%s_0' % t['name'] if t['sprites'] else '0'
        s2 = 'spr_%s_1' % t['name'] if len(t['sprites']) > 1 else '0'
        data("    db 0x%02x, %d, %d ; dw %s, %s ; db %d, %d, %d, %d   // %d %s" % (
            (t['flags'], first, len(tid), s1, s2) + tuple(t['box']) + (i + 1, t['name'])))
    data()
    for t in TYPES:
        for j, s in enumerate(t['sprites']):
            data.sprite('spr_%s_%d' % (t['name'], j), s)
    data()
    data("// Items: two animation frames each.")
    data("item_sprites:")
    for i, name in enumerate(ITEMS):
        data("    dw spr_item_%s_0, spr_item_%s_1" % (name, name))
    for name in sorted(set(ITEMS)):
        for fr in (0, 1):
            data.sprite('spr_item_%s_%d' % (name, fr), O.item(name, fr))
    data()

    data("// Floor tiles for each floor style (two, for alternate tiles).")
    tiles = {s: O.floor_tile(s) for s in FLOOR_STYLES}
    data("floor_tiles:")
    for s in FLOOR_STYLES:
        data("    dw tile_%s, tile_%s" % (s, 'check2' if s == 'check' else s))
    for s in FLOOR_STYLES:
        data.sprite('tile_' + s, tiles[s])
    data.sprite('tile_check2', O.floor_tile('check', 1))
    data.sprite('rim_s', Sprite(O.rim('s'), 0, 0))
    data.sprite('rim_e', Sprite(O.rim('e'), 0, 0))
    data()
    data("// Back wall pieces (for the north wall; the west wall uses mirror images).")
    data("wall_sprites:")
    for k in WALL_KINDS:
        data("    dw wall_%s" % k)
    for k in WALL_KINDS:
        data.sprite('wall_' + k, wall_sprite(wall_segment('n', WALL_DESIGNS[k])))
    # Vertical lines for the corner and ends of the walls.
    for name, bit in (('vline_l', 0x80), ('vline_r', 0x01)):
        data(name + ":")
        data("    db 1, %d, 0, 0" % (WALL_H + 1))
        data.bytes([0xff ^ bit, bit] * (WALL_H + 1))
    data()
    data("// Texts said when looking at each kind of wall decoration.")
    data("wall_texts:")
    for k in WALL_KINDS:
        tid = [texts.add(s) for s in WALL_TEXTS.get(k, [])]
        data("    db %d, %d   // %s" % (tid[0] if tid else 0, len(tid), k))
    data()

    data("// The wanderer: front stand/step/step, then back stand/step/step.")
    data("player_frames:")
    names = []
    for back in (0, 1):
        for step in (0, 1, 2):
            names.append('spr_player_%d_%d' % (back, step))
    data("    dw " + ", ".join(names))
    for back in (0, 1):
        for step in (0, 1, 2):
            img = O.wanderer(back, step)
            data.sprite('spr_player_%d_%d' % (back, step), O.player_sprite(img))
    data()
    data("// The grey lady, faint to strong.")
    data("lady_frames:")
    data("    dw spr_lady_0, spr_lady_1, spr_lady_2")
    for lv in range(3):
        data.sprite('spr_lady_%d' % lv, O.lady(lv))
    data()

    # --- Rooms.
    names = Texts()
    data("// Rooms. Each: size (W<<4 | D, in tiles), attribute, floor style,")
    data("// name, exits N E S W (0xff for none), map position")
    data("// (x | y<<3 | floor<<6), then W north-wall and D west-wall pieces,")
    data("// then objects (type, x | y<<4, extra) ending with 0.")
    data("room_table:")
    for r in order:
        data("    dw room_%d" % r.id)
    for r in order:
        nid = names.add(r.name, 22, 1)
        ex = [r.exits[d][0].id if d in r.exits else 0xff for d in 'NESW']
        data("room_%d:   // %s %s" % (r.id, r.name, (r.floor, r.x, r.y)))
        data("    db 0x%02x, 0x%02x, %d, %d, %d, %d, %d, %d, 0x%02x" % (
            (r.W << 4 | r.D, r.attr, r.style, nid) + tuple(ex) + (r.x | r.y << 3 | r.floor << 6,)))
        ws = [WALL_KINDS.index(r.walls.get(('n', i), 'plain')) for i in range(r.W)]
        ws += [WALL_KINDS.index(r.walls.get(('w', i), 'plain')) for i in range(r.D)]
        data("    db " + ", ".join(str(w) for w in ws))
        for (name, tx, ty, extra) in r.objs:
            if name in ('stairsup', 'stairsdown'):
                extra = r.stairs[1].id
            data("    db T_%s, 0x%02x, %d" % (name.upper(), tx | ty << 4, extra))
        data("    db 0")
    data()
    data("visited:")
    data.bytes([0] * len(order))
    data("room_names:")
    for i in range(len(names.items)):
        data("    dw rname_%d" % i)
    for i, s in enumerate(names.items):
        data("rname_%d:" % i + dstring(s)[3:])
    data()

    # --- Other text.
    ids = {}
    ids['nothing'] = [texts.add(s) for s in NOTHING_TEXTS]
    ids['ambient'] = [texts.add(s) for s in AMBIENT_TEXTS]
    ids['ghost'] = [texts.add(s) for s in GHOST_TEXTS]
    for key in ('nothing', 'ambient', 'ghost'):
        v = ids[key]
        assert v == list(range(v[0], v[0] + len(v))), key
        consts("const TX_%s = %d" % (key.upper(), v[0]))
    for k, s in MSG.items():
        if '{}' in s:
            for lock in (1, 2, 3):
                consts("const TX_%s%d = %d" % (k.upper(), lock, texts.add(s.format(LOCK_NAMES[lock]))))
        else:
            consts("const TX_%s = %d" % (k.upper(), texts.add(s)))
    for i, s in enumerate(KEY_TEXT):
        consts("const TX_KEY%d = %d" % (i, texts.add(s)))
    assert len(texts.items) < 256

    data("texts:")
    for i in range(len(texts.items)):
        data("    dw text_%d" % i)
    for i, s in enumerate(texts.items):
        data("text_%d:" % i + dstring(s)[3:])
    data()

    # Long texts, shown in a panel 26 characters wide.
    panels = Out()
    data("item_names:")
    data("    dw " + ", ".join("iname_%d" % i for i in range(len(ITEM_NAMES))))
    for i, s in enumerate(ITEM_NAMES):
        data("iname_%d:" % i + dstring(s)[3:])
    data("memory_texts:")
    data("    dw " + ", ".join("mtext_%d" % i for i in range(len(MEMORY_TEXT))))
    for i, s in enumerate(MEMORY_TEXT):
        lines = wrap(s, 26)
        assert len(lines) <= 8, s
        data("mtext_%d:" % i + dstring('|'.join(lines))[3:])
    lines = wrap(INTRO_TEXT, 26)
    assert len(lines) <= 14
    data("intro_text:" + dstring('|'.join(lines))[3:])
    data("ending_texts:")
    data("    dw " + ", ".join("etext_%d" % i for i in range(len(ENDING_TEXT))))
    for i, s in enumerate(ENDING_TEXT):
        lines = wrap(s, 26)
        assert len(lines) <= 8, s
        data("etext_%d:" % i + dstring('|'.join(lines))[3:])
    consts("const NENDING = %d" % len(ENDING_TEXT))
    data("floor_names:")
    data("    dw " + ", ".join("fname_%d" % i for i in range(len(FLOOR_NAMES))))
    for i, s in enumerate(FLOOR_NAMES):
        data("fname_%d:" % i + dstring(s)[3:])
    data()
    data("// Font: characters 32 to 90, 8 bytes each.")
    data("font:")
    data.bytes(font_bytes(), 8)

    stats = dict(rooms=len(order), texts=len(texts.items))
    return consts.lines, data.lines, stats, rooms, order


def splice(path, marker, lines):
    src = open(path).read().split('\n')
    begin = "// ---- BEGIN GENERATED %s (gen.py) ----" % marker
    end = "// ---- END GENERATED %s ----" % marker
    i, j = src.index(begin), src.index(end)
    src[i + 1:j] = lines
    open(path, 'w').write('\n'.join(src))


def write_png(path, rows):
    h, w = len(rows), len(rows[0])
    raw = b''.join(b'\x00' + bytes(sum(([c, c, c] for c in r), [])) for r in rows)

    def chunk(t, d):
        c = struct.pack('>I', len(d)) + t + d
        return c + struct.pack('>I', zlib.crc32(t + d) & 0xffffffff)
    png = b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
    png += chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b'')
    open(path, 'wb').write(png)


def preview():
    sprs = []
    for t in TYPES:
        sprs += t['sprites']
    for name in sorted(set(ITEMS)):
        sprs.append(O.item(name, 1))
    for back in (0, 1):
        for step in (0, 1, 2):
            sprs.append(O.player_sprite(O.wanderer(back, step)))
    for lv in range(3):
        sprs.append(O.lady(lv))
    for k in WALL_KINDS:
        sprs.append(wall_sprite(wall_segment('n', WALL_DESIGNS[k])))
        sprs.append(wall_sprite(wall_segment('w', WALL_DESIGNS[k])))
    for s in FLOOR_STYLES:
        sprs.append(O.floor_tile(s))
    W = 640
    x = y = rowh = 0
    placed = []
    for s in sprs:
        if x + s.img.w + 4 > W:
            x, y, rowh = 0, y + rowh + 4, 0
        placed.append((x, y, s))
        x += s.img.w + 4
        rowh = max(rowh, s.img.h)
    H = y + rowh
    scale = 2
    rows = [[60] * (W * scale) for _ in range(H * scale)]
    for (ox, oy, s) in placed:
        for j in range(s.img.h):
            for i in range(s.img.w):
                v = s.img.p[j][i]
                c = {T: 60, INK: 230, PAP: 0}[v]
                for dy in range(scale):
                    for dx in range(scale):
                        rows[(oy + j) * scale + dy][(ox + i) * scale + dx] = c
    write_png('sprites.png', rows)


if __name__ == '__main__':
    consts, data, stats, rooms, order = generate()
    splice('hollowmere.z80', 'CONSTS', consts)
    splice('hollowmere.z80', 'DATA', data)
    print("rooms: %(rooms)d, texts: %(texts)d" % stats, file=sys.stderr)
    if '--preview' in sys.argv:
        preview()
