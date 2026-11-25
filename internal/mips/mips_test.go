package mips

import "testing"

// Expected words come from the Additional I BOOT.BIN (docs/0010-subtitle-patch.md).
func TestEncodings(t *testing.T) {
	for _, c := range []struct {
		name      string
		got, want uint32
	}{
		{"addiu sp,sp,-0x20", Addiu(SP, SP, -0x20), 0x27BDFFE0},
		{"sw ra,0x10(sp)", Sw(RA, SP, 0x10), 0xAFBF0010},
		{"jal 0xB35D0", Jal(0xB35D0), 0x0C02CD74},
		{"addiu a0,a0,0x130", Addiu(A0, A0, 0x130), 0x24840130},
		{"lw ra,0x10(sp)", Lw(RA, SP, 0x10), 0x8FBF0010},
		{"jr ra", Jr(RA), 0x03E00008},
		{"or s0,a3,zero", Or(S0, A3, Zero), 0x00E08025},
		{"lui a1,0x13", Lui(A1, 0x13), 0x3C050013},
		{"jalr t9", Jalr(T9), 0x0320F809},
		{"mtc1 t0,$f12", Mtc1(T0, 12), 0x44886000},
		{"bal +1", Bal(1), 0x04110001},
		{"beq s2,zero,+0x3b", Beq(18, Zero, 0x3B), 0x1240003B},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %#08x, want %#08x", c.name, c.got, c.want)
		}
	}
}
