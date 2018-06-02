Z80 Assembler
=============

This repository contains a z80 assembler, both as a command-line tool, and as a library.
It currently is somewhat limited, both in assembler features (for example, there's no
macros), and can currently only output ZX-Spectrum .sna files. But the assembler does
implement the full (standard) z80 instruction set.

The code is MIT licensed, and the details can be found in LICENSE.txt.

Syntax
======

Assembler instructions are case-insensitive, with destination registers (or addresses)
appearing before source when applicable. Hex numbers can be used, and are written with
a `0x` prefix.

For example:

    ld a, 42
    jp label
    ld b, 0x10

Multiple instructions may be written on one line, by separating them by `;`.

    ld a, 42 ; inc a

Indirection uses regular brackets `()`. For example:

    ld hl, (123)
    sub (hl)
    ld (hl), d

Indirection via `ix` and `iy` use syntax like this: `(ix+4)`. For example:

    ld (ix+4), 42

Comments use `//` or `/* ... */` (which don't nest). For example, this code generates the single instuction `ld a, (de)`:

    /* The next two instructions are commented out.
    ld a, 42
    ld b, 42 */
    ld a, (de) // This is a comment

Labels are written as a dot followed by a name, and must appear at the start of a line. For example:

    .label
      djnz label

A special label `.main` defines the entrypoint for the code.

Where applicable, constants may be expressions written in C (or equivalently go) syntax. For example:

    ld a, 4+10

There are several assembler directives: `org` which speficies where to assemble, and `db`, `dw`, `ds`
which allow literal bytes, words (16 bits, written low-byte first), and strings. For example:

    org 0x9000
    db 1, 2, 3
    dw 0x1234
    ds "hello\n"

This causes the following bytes to be generated at and beyond memory location `0x9000`:

    1, 2, 3, 0x34, 0x12, 'h', 'e', 'l', 'l', 'o', 0x0a

If you want the length of a string (for example as an 8-bit value), you can use label arithmetic. Note that it is fine to refer to labels before they appear:

    db endhello - hello
    .hello ds "hello\n"
    .endhello

This generates the bytes: `6, 'h', 'e', 'l', 'l', 'o', 0x0a`.
