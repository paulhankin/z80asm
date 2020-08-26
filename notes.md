Notes
=====

Labels
------

Idea 1, labels could be of two forms:

  * `label:` a major label
  * `.label` a minor label

Dot labels should be named relative to the previous major label, and using a dotted name implicitly prepends the most recent major label.

So:

    f:
    .j djnz .j

Is equivalent to:

    f:
    f.j: djnz f.j


Alternatives:

    .label
    ..sublabel
    ...subsublabel

or:

    label:
    .sublabel:
    ..subsublabel:

or:

use `{}` to delimit scopes.

    f: {
        j: djnz j
    }

Here there would be no way to refer to the inner `j` label from outside. Inside a scope, labels would be searched from the inside out.

Use of `{}` could also allow namespaces, local consts and so on, and be used for meta constructs like `if` and `macro` and so on.


Consts and labels
-----------------

Currently, consts can refer to labels that are defined after them. This could be a problem if value of the label can depend on the const (for example, if
a section of code is conditionally assembled based on the const changing the amount of code that is assembled before the label).

Probably expressions should be marked tainted if they depend on the value of a label, and such tainted expressions should only be allowed to be used where they can only affect the bytes written but not the number of bytes written.

Alternatives:

* Require that labels are consistent between first and second pass.
* `const` expressions can't depend on labels that haven't been defined yet
* The assembler keeps iterating until it converges or it gives up.


Macros
======

This format seems natural:

    macro name arg0, arg1, arg2 {
        
    }

Instantiated like this:

    name 42, BC, 0x123

The arguments would be parsed as expressions, and then substituted as expressions into the code.

Initially, macros would return no result: they would assemble the instructions they assemble, but that's all.


Conditional assembly
====================

This format seems natural:

    if <expression> {
        ...
    } else if <expression2> {
        ...
    } else {
        ...
    }

Like C or Go, there can be 0 or more `else if` and 0 or 1 `else`.

The problem here is that if `{}` defines scopes for labels, then there's no way for labels to escape.
