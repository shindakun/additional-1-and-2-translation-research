// Package subpatch builds the movie subtitle patch for BOOT.BIN.
// See docs/0010-subtitle-patch.md.
package subpatch

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/mips"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/prx"
)

// Addrs holds the addresses one BOOT.BIN needs.
type Addrs struct {
	DrawCall      uint32 // the jal to CAcffMovie::Draw in CSimpleMovieScene::Draw
	MovieDraw     uint32 // CAcffMovie::Draw
	Cave, CaveEnd uint32 // unreferenced function used as code space
	FontPtr       uint32 // word holding the current font
	SetCurrent    uint32 // FsFontSetCurrent
	SetColor      uint32 // FsFontSetBaseColor
	SetAlign      uint32 // FsFontSetAlignment
	DrawText      uint32 // FsFontDrawText
	SetScale      uint32 // FsFontSetScale
	Begin, End    uint32 // FsFontBegin, FsFontEnd: queue glyphs, then draw them
	DrawState     uint32 // word FsFontInit sets; zero until the first font loads
	GuDisable     uint32 // sceGuDisable
	Sprintf       uint32 // sprintf
}

// known maps BOOT.BIN SHA-1 to its addresses.
var known = map[string]Addrs{
	"e62c3559fc94e9c917081bdf5bbf8dd5713ef1ae": { // Additional I
		DrawCall: 0x3EB30, MovieDraw: 0xB35D0,
		Cave: 0xF232C, CaveEnd: 0xF232C + 6552,
		FontPtr: 0x1262F4, SetCurrent: 0x14C1C, SetColor: 0x14EA0,
		SetAlign: 0x15168, DrawText: 0x15288, SetScale: 0x150E8,
		Begin: 0x168F4, End: 0x16958, GuDisable: 0xC4B4C, Sprintf: 0xFE6F0,
		DrawState: 0x1415FC,
	},
}

// AddrsFor returns the addresses for a known BOOT.BIN.
func AddrsFor(boot []byte) (Addrs, error) {
	sum := sha1.Sum(boot)
	a, ok := known[hex.EncodeToString(sum[:])]
	if !ok {
		return a, errors.New("subpatch: unknown BOOT.BIN")
	}
	return a, nil
}

// builder emits position-independent code: s0 holds the run-time address
// of base, so a link-time address x is at s0 + (x - base).
type builder struct {
	p    mips.Program
	base uint32
}

// addr puts the run-time address of a link-time address in reg.
func (b *builder) addr(reg, linkAddr uint32) {
	b.p.LoadConst(reg, linkAddr-b.base)
	b.p.Emit(mips.Addu(reg, reg, mips.S0))
}

// call calls a link-time address through t9.
func (b *builder) call(target uint32) {
	b.addr(mips.T9, target)
	b.p.Emit(mips.Jalr(mips.T9), mips.Nop())
}

// floatArg loads a float constant into $fN.
func (b *builder) floatArg(f int, v float32) {
	b.p.LoadConst(mips.T0, math.Float32bits(v))
	b.p.Emit(mips.Mtc1(mips.T0, uint32(f)))
}

// Counter patches boot to draw "F <n>" over every movie frame, where n
// counts draw calls. It shows that the hook runs and measures the rate.
func Counter(boot []byte, x, y float32) ([]byte, error) {
	a, err := AddrsFor(boot)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), boot...)
	img, err := prx.Open(out)
	if err != nil {
		return nil, err
	}
	if img.Word(a.DrawCall) != mips.Jal(a.MovieDraw) || img.RelocAt(a.DrawCall) != prx.R26 {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.DrawCall)
	}

	b := &builder{p: mips.Program{Base: a.Cave}}
	const frame = 48
	b.p.Emit(
		mips.Addiu(mips.SP, mips.SP, -frame),
		mips.Sw(mips.RA, mips.SP, 32),
		mips.Sw(mips.S0, mips.SP, 36),
		mips.Bal(1), // ra = address of the instruction after the delay slot
		mips.Nop(),
	)
	b.base = b.p.PC()
	b.p.Emit(mips.Or(mips.S0, mips.RA, mips.Zero))

	// Data follows the code; reserve its addresses first.
	const codeWords = 192
	data := a.Cave + codeWords*4
	counter, buf, format := data, data+4, data+36

	b.call(a.MovieDraw) // a0 still holds the movie object

	b.addr(mips.T1, counter)
	b.p.Emit(
		mips.Lw(mips.A2, mips.T1, 0),
		mips.Addiu(mips.A2, mips.A2, 1),
		mips.Sw(mips.A2, mips.T1, 0),
	)
	b.addr(mips.A0, buf)
	b.addr(mips.A1, format)
	b.call(a.Sprintf)

	// No font is loaded during the first opening movie: FsFontInit has not
	// set the draw state, and FsFontBegin would write through a null pointer.
	b.addr(mips.T1, a.DrawState)
	b.p.Emit(mips.Lw(mips.T0, mips.T1, 0))
	skip := len(b.p.Words)
	b.p.Emit(mips.Beq(mips.T0, mips.Zero, 0), mips.Nop()) // offset set below

	// The sequence CAdvMenuPrinter uses: set the font, begin, scale, draw,
	// end. FsFontDrawText only queues glyphs; FsFontEnd draws them.
	b.addr(mips.T1, a.FontPtr)
	b.p.Emit(mips.Lw(mips.A0, mips.T1, 0))
	b.call(a.SetCurrent)
	b.call(a.Begin)
	b.floatArg(12, 1)
	b.floatArg(13, 1)
	b.call(a.SetScale)
	b.p.Emit(mips.Or(mips.A0, mips.Zero, mips.Zero))
	b.call(a.SetAlign)
	// Shadow first, then the text: black one pixel down and right, white on top.
	for _, pass := range []struct {
		color uint32
		dx    float32
	}{{0xFF000000, 1}, {0xFFFFFFFF, 0}} {
		b.p.LoadConst(mips.A0, pass.color)
		b.call(a.SetColor)
		b.floatArg(12, x+pass.dx)
		b.floatArg(13, y+pass.dx)
		b.addr(mips.A0, buf)
		b.call(a.DrawText)
	}
	b.call(a.End)
	for _, state := range []int32{10, 7, 1} { // as CAdvMenuPrinter::End
		b.p.Emit(mips.Addiu(mips.A0, mips.Zero, state))
		b.call(a.GuDisable)
	}

	b.p.Words[skip] = mips.Beq(mips.T0, mips.Zero, int32(len(b.p.Words)-skip-1))
	b.p.Emit(
		mips.Lw(mips.RA, mips.SP, 32),
		mips.Lw(mips.S0, mips.SP, 36),
		mips.Jr(mips.RA),
		mips.Addiu(mips.SP, mips.SP, frame),
	)
	if len(b.p.Words) > codeWords {
		return nil, fmt.Errorf("subpatch: code is %d words, room for %d", len(b.p.Words), codeWords)
	}
	for len(b.p.Words) < codeWords {
		b.p.Emit(mips.Nop())
	}
	b.p.Emit(0)                      // counter
	b.p.Emit(make([]uint32, 8)...)   // sprintf buffer, 32 bytes
	b.p.Words = append(b.p.Words, 0) // format string follows
	code := b.p.Bytes()
	code = append(code[:len(code)-4], []byte("F %d\x00\x00\x00\x00")...)
	if a.Cave+uint32(len(code)) > a.CaveEnd {
		return nil, errors.New("subpatch: patch does not fit the code space")
	}

	img.Neutralize(a.Cave, a.CaveEnd)
	img.Write(a.Cave, code)
	img.SetWord(a.DrawCall, mips.Jal(a.Cave))
	return out, nil
}
