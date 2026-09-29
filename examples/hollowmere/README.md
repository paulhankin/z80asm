Hollowmere
==========

A quiet game of exploration for the 48K ZX Spectrum, in isometric 3D.

The gates of Hollowmere stand open, and nobody answers when you call. You
wander its empty rooms by lantern light: great halls, a chapel, crypts,
bedchambers, a nursery, a library, battlements and a bell tower, 91 rooms
over four floors. There is no combat, and nothing can hurt you. Somewhere
in the castle are eight things that were once loved; each one you find
tells you a little more of what happened here. Find them all, and climb
to the belfry.

`hollowmere.sna` is a ready-built snapshot that can be loaded into any
Spectrum emulator. To rebuild it from the source:

    go run ./cmd/z80asm examples/hollowmere/hollowmere.z80

Playing
-------

| Keys            | Action                                     |
|-----------------|--------------------------------------------|
| Q, 7            | walk north (up and to the right)           |
| A, 6            | walk south (down and to the left)          |
| O, 5            | walk west (up and to the left)             |
| P, 8            | walk east (down and to the right)          |
| SPACE, ENTER    | look at whatever is next to you            |
| M               | a map of the rooms you've seen on this floor |

Hold two keys together to walk straight across the screen. Walk into an
item to pick it up. Some doors have iron gates: you'll need to find their
keys. Stairs lead up and down between floors. Look at things; the castle
has plenty to say, and you may not be as alone as you think.

How it works
------------

* Each room is a floor of up to 7x7 tiles with two back walls. The screen
  is redrawn a rectangle at a time: the background pieces (wall sections,
  floor tiles and the front edges of the floor), then the objects, player
  and ghost sorted by depth, are drawn into a 2K buffer with masked
  sprites, and then copied to the screen. Only rectangles that change are
  redrawn, and pieces outside them are skipped quickly.
* Walls, furniture and floors are byte aligned. The player moves smoothly
  (two pixels across per step), so their sprite is shifted and mirrored as
  needed, and kept in a cache for the direction they face. The ghost is
  drawn with a mask that lets the room show through her.
* Collision is against the floor, the doorways and each object's box, in
  world units. Walking slightly off-line into a doorway nudges you into it.
* The game doesn't use the ROM: it has its own font, keyboard reading and
  beeper sounds, and an IM 2 interrupt handler for timing. The code sits
  above 32K, out of the memory that the display slows down.

`gen.py` (with `gfx.py` and `objects.py`) generates the graphics, rooms
and text in `hollowmere.z80`: run `python3 gen.py` in this directory to
regenerate them. The castle is laid out as four floor plans in `gen.py`.
The furniture is drawn from isometric boxes and cylinders, or from pixel
art. Each room is furnished according to its kind, with a check that every
doorway, staircase and item can still be reached. `python3 gen.py
--preview` also writes `sprites.png`, a sheet of all the sprites.

Tests
-----

`go test ./examples/hollowmere` runs the game in a small headless Spectrum
emulator (built on the Z80 core in `z80test/z80`):

* `TestPlaythrough` plays the whole game through the keyboard, with a bot
  that plans its routes from the game's memory. It collects every item,
  opens every gate, climbs to the belfry, rings the bell and watches the
  ending, and fails if the game ever writes over its own code.
* `TestAllRooms` draws every room.
* `TestWalkSpeed` measures how fast the game runs and profiles it.
* `TestLayout` checks that the program fits below its buffers, and
  `TestSnapshot` that `hollowmere.sna` is up to date.

Set `HM_SHOTS` to a directory to save screenshots, including a contact
sheet of every room on each floor.
