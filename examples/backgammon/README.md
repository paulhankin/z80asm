Backgammon
==========

A game of backgammon for the 48K ZX Spectrum, with a computer opponent.

`backgammon.sna` is a ready-built snapshot that can be loaded into any
Spectrum emulator. To rebuild it from the source:

    go run ./cmd/z80asm examples/backgammon/backgammon.z80

Playing
-------

You play yellow, moving from point 24 down to point 1 and bearing off
from your home board in the bottom right. The computer plays black. The
score is kept across games: a single game is 1 point, a gammon 2, and a
backgammon 3. There is no doubling cube.

| Keys            | Action                                   |
|-----------------|------------------------------------------|
| O, 5, 6         | previous choice                          |
| P, 8, 7         | next choice                              |
| SPACE, ENTER, M | select                                   |
| X, 0            | cancel the selected checker              |
| U               | undo all the moves made so far this turn |

On your turn, the cursor only visits checkers that can legally move. Once
you pick one, it only visits legal destinations (if there is just one, the
checker moves straight away). The full rules are enforced, including
having to use both dice if possible, and having to use the larger die if
only one can be used.

How it works
------------

* Each side's board is a 26-byte array from its own point of view
  (0 = borne off, 1-24 = points, 25 = bar), so one rules engine serves both
  players.
* The rules engine searches every sequence of moves for the dice rolled.
  It's used to work out which moves are legal for the human, and by the
  computer player, which evaluates the position at the end of every legal
  sequence and picks the best. The evaluation considers the pip count,
  points held (and runs of consecutive points), blots weighted by the
  number of rolls that hit them and how far back a hit would send them,
  and switches to a simpler race evaluation once there's no contact.
* The game doesn't use the ROM: it has its own font, keyboard reading, and
  an IM 2 interrupt handler for timing.

`gen_gfx.py` generates the font, checker and dice bitmaps in the source.

Tests
-----

`go test ./examples/backgammon` runs the game in a small headless Spectrum
emulator (built on the Z80 core in `z80test/z80`). The tests check the
Z80 rules engine against a reference implementation in Go on thousands of
random positions, check that the computer always makes a legal move, and
play complete games through the user interface with simulated key presses.
Set `BG_SHOTS` to a directory to save screenshots.
