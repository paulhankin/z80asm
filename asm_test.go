package z80asm

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

type ffs map[string]string

func (f ffs) open(filename string) (io.ReadCloser, error) {
	contents, ok := f[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return ioutil.NopCloser(strings.NewReader(contents)), nil
}

func toHex(bs []byte) string {
	var r []string
	for _, b := range bs {
		r = append(r, fmt.Sprintf("%02x", b))
	}
	return strings.Join(r, " ")
}

func TestAsmSnippets(t *testing.T) {
	testcases := []struct {
		fs   ffs
		want []byte
	}{
		{
			fs: ffs{
				"a.asm": "xor a",
			},
			want: b(0xaf),
		},
		{
			fs: ffs{
				"a.asm": "ld bc, 42",
			},
			want: b(0x01, 42, 0),
		},
		{
			fs: ffs{
				"a.asm": "ld hl, 0x4243",
			},
			want: b(0x21, 0x43, 0x42),
		},
		{
			fs: ffs{
				"a.asm": "ld hl, 0x4243; ld bc, 0x1023",
			},
			want: b(0x21, 0x43, 0x42, 0x01, 0x23, 0x10),
		},
		{
			fs: ffs{
				"a.asm": "ld a, 0; ld b, 3; ld h, a; ld l, -2",
			},
			want: b(0x3e, 0, 0x06, 3, 0x67, 0x2e, 254),
		},
		{
			fs: ffs{
				"a.asm": ".label ld hl, label",
			},
			want: b(0x21, 0x00, 0x80),
		},
		{
			fs: ffs{
				"a.asm": ".label push bc; jr label",
			},
			want: b(0xc5, 0x18, 0xfd),
		},
		{
			fs: ffs{
				"a.asm": "rst 0x20",
			},
			want: b(0xe7),
		},
		{
			fs: ffs{
				"a.asm": `db 1, 2, 3, 'h', '\n', '\t', 42`,
			},
			want: b(1, 2, 3, uint8('h'), uint8('\n'), uint8('\t'), 42),
		},
		{
			fs: ffs{
				"a.asm": `rrca ; ret ; di`,
			},
			want: b(0x0f, 0xc9, 0xf3),
		},
		{
			fs: ffs{
				"a.asm": `ld a, (hl)`,
			},
			want: b(0x7e),
		},
		{
			fs: ffs{
				"a.asm": "ld a, (data); .data db 1, 2",
			},
			want: b(0x3a, 0x03, 0x80, 1, 2),
		},
		{
			fs: ffs{
				"a.asm": "out (42), a; out (c), h; in a, (10); in b, (c)",
			},
			want: b(0xd3, 42, 0xed, 0x61, 0xdb, 10, 0xed, 0x40),
		},
		{
			fs: ffs{
				"a.asm": ".loop jr nz, loop",
			},
			want: b(0x20, 0xfe),
		},
	}
	for _, tc := range testcases {
		desc := tc.fs["a.asm"]
		ram := make([]byte, 65536)
		asm, err := NewAssembler(ram)
		if err != nil {
			t.Fatalf("%q: failed to create assembler: %v", desc, err)
		}
		asm.opener = tc.fs.open
		if err := asm.AssembleFile("a.asm"); err != nil {
			t.Errorf("%q: assembler produced error: %v", desc, err)
			continue
		}
		progStart := 0x8000
		for i := 0; i < 65536; i++ {
			if i < progStart || i >= progStart+len(tc.want) {
				if ram[i] != 0 {
					t.Errorf("%q: byte %04x = %02x, want 0", desc, i, ram[i])
				}
			}
		}
		if got := ram[progStart : progStart+len(tc.want)]; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: assembled at %04x\ngot:\n%s\nwant:\n%s", desc, progStart, toHex(got), toHex(tc.want))
		}
	}
}
