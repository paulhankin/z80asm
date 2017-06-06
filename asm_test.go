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

func testFailureSnippet(t *testing.T, fs ffs, mustContain string) {
	desc := fs["a.asm"]
	ram := make([]byte, 65536)
	asm, err := NewAssembler(ram)
	if err != nil {
		t.Fatalf("%q: failed to create assembler: %v", desc, err)
	}
	asm.opener = fs.open
	err = asm.AssembleFile("a.asm")
	if err == nil {
		t.Errorf("%q: assembler succeeded, expected match %q", desc, mustContain)
		return
	}
	if !strings.Contains(err.Error(), mustContain) {
		t.Errorf("%q: failure %q does not match %q", desc, err.Error(), mustContain)
	}
}

func testSnippet(t *testing.T, org int, fs ffs, want []byte) {
	desc := fs["a.asm"]
	ram := make([]byte, 65536)
	asm, err := NewAssembler(ram)
	if err != nil {
		t.Fatalf("%q: failed to create assembler: %v", desc, err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Errorf("%q: assembler produced error: %v", desc, err)
		return
	}
	for i := 0; i < 65536; i++ {
		if i < org || i >= org+len(want) {
			if ram[i] != 0 {
				t.Errorf("%q: byte %04x = %02x, want 0", desc, i, ram[i])
			}
		}
	}
	if got := ram[org : org+len(want)]; !reflect.DeepEqual(got, want) {
		t.Errorf("%q: assembled at %04x\ngot:\n%s\nwant:\n%s", desc, org, toHex(got), toHex(want))
	}
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
				"a.asm": `dw 1, 2, 256`,
			},
			want: b(1, 0, 2, 0, 0, 1),
		},
		{
			fs: ffs{
				"a.asm": `ds "hello\n"`,
			},
			want: []byte("hello\n"),
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
		testSnippet(t, 0x8000, tc.fs, tc.want)
	}
}

func TestParseErrors(t *testing.T) {
	testCases := []struct {
		asm     string
		wantErr string // partial match
	}{
		{"xor a, b", "no suitable"},
		{"ld hl, (42", ")"},
		{"ld a, (1+2*3", ")"},
		{"ld a, 2+3+", "EOF"},
		{"ld a, 1 ld b, 2", "unexpected Ident"},
		{"ld b, (123)", "no suitable"},
		{"xor a,", "unexpected trailing ,"},
		{"xor missing", "label"},
		{"ld hl, 6/(4-4)", "zero"},
		{"ld hl, 6%(4-4)", "zero"},
	}
	for _, tc := range testCases {
		fs := ffs{"a.asm": tc.asm}
		testFailureSnippet(t, fs, tc.wantErr)
	}
}

func TestIntExpressions(t *testing.T) {
	testCases := []struct {
		expr string // An arithmetic operation.
		want uint16
	}{
		{"1+2", 3},
		{"7*4", 28},
		{"1-2", 65536 - 1},
		{"8/4", 2},
		{"-1+2", 1},
		{"1+2*3", 7},
		{"2*3+4", 10},
		{"2*(3+4)", 14},
		{"(1+2)*3", 9},
		{"8*8*8", 512},
		{"label+1", 0x6001},
		{"label/2", 0x3000},
		{"label*2", 0xc000},
		{"label+1*2", 0x6002},
		{"label*2+1", 0xc001},
		{"label-1", 0x5fff},
		{"label+label", 0xc000},
		{"16>>2", 4},
		{"2<<3", 16},
		{"1==2", 0},
		{"1+2==4-1", 1},
		{"1!=2", 1},
		{"1+2!=4-1", 0},
		{"10+20<10+30", 1},
		{"10+20<10+20", 0},
		{"10+20<=10+20", 1},
		{"10+20>10+30", 0},
		{"10+20>10+20", 0},
		{"10+20>=10+20", 1},
		{"10+30>10+20", 1},
		{"(16>>4)&1", 1},
		{"(16>>4)&3", 1},
		{"(7>>1)&2", 2},
		{"5|12", 13},
		{"5&12", 4},
		{"7%3", 1},
		{"^10", 65536 - 11},
		{"^0", 65535},
		{"^0+1", 0},
		{"!(1==0)", 1},
		{"!(0==0)", 0},
		{"-(1+2)", 65536 - 3},
		{"7&^2", 5},
		{"42||badness", 42},
		{"42&&43", 43},
		{"42||43", 42},
		{"0&&badness", 0},
		{"1&&0||3", 3},
		{"1&&2||3", 2},
		{"0&&2||3", 3},
		{"1==2 && 2==2", 0},
		{"1==2 || 2==2", 1},
		{"1==2 || !(2==2)", 0},
	}
	for _, tc := range testCases {
		fs := ffs{
			"a.asm": fmt.Sprintf("org 0x6000 ; .label ld hl, %s", tc.expr),
		}
		want := b(0x21, byte(tc.want%256), byte(tc.want/256))
		testSnippet(t, 0x6000, fs, want)
	}
}
