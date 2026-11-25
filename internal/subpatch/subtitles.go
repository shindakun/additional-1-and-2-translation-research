package subpatch

import (
	"errors"
	"fmt"
	"math"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/mips"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/prx"
)

// Line is one subtitle line: ASCII text drawn from frame Start up to, but
// not including, frame End of a movie. X and Y are screen pixels; Align
// says which part of the line X marks.
type Line struct {
	Start, End uint32
	X, Y       float32
	Align      uint32 // FsFontSetAlignment value
	Movie      uint32 // MovieKey of the movie file, or 0 for every movie
	Text       string
}

// MovieKey packs the three characters before the extension of a movie file
// name ("movie000.pmf" gives "000") into a word, as the patch does with the
// path that CSimpleMovieScene::SetMovieFile receives.
func MovieKey(name string) uint32 {
	dot := -1
	for i := range len(name) {
		if name[i] == '.' {
			dot = i
		}
	}
	if dot < 3 {
		return 0
	}
	return uint32(name[dot-3])<<16 | uint32(name[dot-2])<<8 | uint32(name[dot-1])
}

// FsFontSetAlignment values for horizontal alignment (bits 4-5), from a
// headless test: 0x00 starts the text at X, 0x10 ends it at X, 0x20 centers
// it on X.
const (
	AlignLeft   = 0x00
	AlignRight  = 0x10
	AlignCenter = 0x20
)

// recordWords is the size of one line record: start, end, x, y, shadow x,
// shadow y, the text offset from the code base, the alignment and the movie
// key.
const recordWords = 9

// Subtitles patches boot to draw lines over movies. A frame counter starts
// at 0 for each movie (the exit hook resets it). Every frame, each line
// whose range holds the counter is drawn in white with a black shadow one
// pixel down and right. A line with a Movie key shows only over that movie.
// The opening
// at boot has no font, so it shows nothing (docs/0010).
func Subtitles(boot []byte, lines []Line) ([]byte, error) {
	a, err := AddrsFor(boot)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		for _, c := range []byte(l.Text) {
			if c < 0x20 || c >= 0x80 || c == '%' {
				return nil, fmt.Errorf("subpatch: line %q: ASCII without %% only", l.Text)
			}
		}
	}
	out := append([]byte(nil), boot...)
	img, err := prx.Open(out)
	if err != nil {
		return nil, err
	}
	if img.Word(a.DrawCall) != mips.Jal(a.MovieDraw) || img.RelocAt(a.DrawCall) != prx.R26 {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.DrawCall)
	}
	if img.Word(a.ExitCall) != mips.Jal(a.ExitTarget) || img.RelocAt(a.ExitCall) != prx.R26 {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.ExitCall)
	}
	if img.Word(a.FileCall) != mips.Jal(a.FileTarget) || img.RelocAt(a.FileCall) != prx.R26 {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.FileCall)
	}

	const codeWords = 224
	const frame = 56
	data := a.Cave + codeWords*4
	counter, curKey, table := data, data+4, data+8
	strings := table + uint32(len(lines)*recordWords*4)

	b := &builder{p: mips.Program{Base: a.Cave}}
	b.p.Emit(
		mips.Addiu(mips.SP, mips.SP, -frame),
		mips.Sw(mips.RA, mips.SP, 32),
		mips.Sw(mips.S0, mips.SP, 36),
		mips.Sw(mips.S1, mips.SP, 40),
		mips.Sw(mips.S2, mips.SP, 44),
		mips.Bal(1),
		mips.Nop(),
	)
	b.base = b.p.PC()
	b.p.Emit(mips.Or(mips.S0, mips.RA, mips.Zero))
	b.call(a.MovieDraw) // a0 still holds the movie object

	b.addr(mips.T1, counter)
	b.p.Emit(
		mips.Lw(mips.T2, mips.T1, 0),
		mips.Addiu(mips.T2, mips.T2, 1),
		mips.Sw(mips.T2, mips.T1, 0),
	)
	// No font yet (the opening at boot): draw nothing.
	b.addr(mips.T1, a.DrawState)
	b.p.Emit(mips.Lw(mips.T0, mips.T1, 0))
	noFont := b.p.Branch(mips.Beq, mips.T0, mips.Zero)

	b.addr(mips.T1, a.FontPtr)
	b.p.Emit(mips.Lw(mips.A0, mips.T1, 0))
	b.call(a.SetCurrent)
	b.call(a.Begin)
	b.floatArg(12, 1)
	b.floatArg(13, 1)
	b.call(a.SetScale)

	b.addr(mips.S1, table)
	b.p.LoadConst(mips.S2, uint32(len(lines)))
	loop := len(b.p.Words)
	done := b.p.Branch(mips.Beq, mips.S2, mips.Zero)
	b.addr(mips.T2, counter)
	b.p.Emit(
		mips.Lw(mips.T2, mips.T2, 0),
		mips.Lw(mips.T0, mips.S1, 0),
		mips.Lw(mips.T1, mips.S1, 4),
		mips.Sltu(mips.T3, mips.T2, mips.T0), // counter < start
	)
	early := b.p.Branch(mips.Bne, mips.T3, mips.Zero)
	b.p.Emit(mips.Sltu(mips.T3, mips.T2, mips.T1)) // counter < end
	late := b.p.Branch(mips.Beq, mips.T3, mips.Zero)
	b.p.Emit(mips.Lw(mips.T0, mips.S1, 32))
	anyMovie := b.p.Branch(mips.Beq, mips.T0, mips.Zero)
	b.addr(mips.T1, curKey)
	b.p.Emit(mips.Lw(mips.T1, mips.T1, 0))
	other := b.p.Branch(mips.Bne, mips.T0, mips.T1)
	anyMovie()
	b.p.Emit(mips.Lw(mips.A0, mips.S1, 28))
	b.call(a.SetAlign)
	for _, pass := range []struct {
		color uint32
		xy    int32
	}{{0xFF000000, 16}, {0xFFFFFFFF, 8}} {
		b.p.LoadConst(mips.A0, pass.color)
		b.call(a.SetColor)
		b.p.Emit(
			mips.Lw(mips.T0, mips.S1, pass.xy),
			mips.Mtc1(mips.T0, 12),
			mips.Lw(mips.T0, mips.S1, pass.xy+4),
			mips.Mtc1(mips.T0, 13),
			mips.Lw(mips.A0, mips.S1, 24),
			mips.Addu(mips.A0, mips.A0, mips.S0),
		)
		b.call(a.DrawText)
	}
	early()
	late()
	other()
	b.p.Emit(
		mips.Addiu(mips.S1, mips.S1, recordWords*4),
		mips.Addiu(mips.S2, mips.S2, -1),
	)
	back := len(b.p.Words)
	b.p.Emit(mips.Beq(mips.Zero, mips.Zero, int32(loop-back-1)), mips.Nop())
	done()
	b.call(a.End)
	for _, state := range []int32{10, 7, 1} { // as CAdvMenuPrinter::End
		b.p.Emit(mips.Addiu(mips.A0, mips.Zero, state))
		b.call(a.GuDisable)
	}
	noFont()
	b.p.Emit(
		mips.Lw(mips.RA, mips.SP, 32),
		mips.Lw(mips.S0, mips.SP, 36),
		mips.Lw(mips.S1, mips.SP, 40),
		mips.Lw(mips.S2, mips.SP, 44),
		mips.Jr(mips.RA),
		mips.Addiu(mips.SP, mips.SP, frame),
	)
	mainBase := b.base

	// Second entry, from CSimpleMovieScene::Exit: run the original call and
	// reset the counter, so every movie starts at frame 0.
	exitEntry := b.p.PC()
	b.p.Emit(
		mips.Addiu(mips.SP, mips.SP, -frame),
		mips.Sw(mips.RA, mips.SP, 32),
		mips.Sw(mips.S0, mips.SP, 36),
		mips.Bal(1),
		mips.Nop(),
	)
	b.base = b.p.PC()
	b.p.Emit(mips.Or(mips.S0, mips.RA, mips.Zero))
	b.call(a.ExitTarget) // a0 and a1 are the caller's
	b.addr(mips.T1, counter)
	b.p.Emit(
		mips.Sw(mips.Zero, mips.T1, 0),
		mips.Lw(mips.RA, mips.SP, 32),
		mips.Lw(mips.S0, mips.SP, 36),
		mips.Jr(mips.RA),
		mips.Addiu(mips.SP, mips.SP, frame),
	)

	// Third entry, from CSimpleMovieScene::SetMovieFile: store the key of the
	// path in a2, then jump to the original target with the caller's
	// arguments and return address.
	fileEntry := b.p.PC()
	b.p.Emit(
		mips.Addiu(mips.SP, mips.SP, -16),
		mips.Sw(mips.RA, mips.SP, 0),
		mips.Sw(mips.S0, mips.SP, 4),
		mips.Bal(1),
		mips.Nop(),
	)
	b.base = b.p.PC()
	b.p.Emit(
		mips.Or(mips.S0, mips.RA, mips.Zero),
		mips.Or(mips.T0, mips.A2, mips.Zero),
		mips.Or(mips.T1, mips.Zero, mips.Zero),
	)
	scan := len(b.p.Words)
	b.p.Emit(mips.Lbu(mips.T2, mips.T0, 0))
	scanned := b.p.Branch(mips.Beq, mips.T2, mips.Zero)
	b.p.Emit(mips.Addiu(mips.T3, mips.Zero, '.'))
	notDot := b.p.Branch(mips.Bne, mips.T2, mips.T3)
	b.p.Emit(mips.Or(mips.T1, mips.T0, mips.Zero))
	notDot()
	b.p.Emit(mips.Addiu(mips.T0, mips.T0, 1))
	back0 := len(b.p.Words)
	b.p.Emit(mips.Beq(mips.Zero, mips.Zero, int32(scan-back0-1)), mips.Nop())
	scanned()
	b.p.Emit(mips.Or(mips.T2, mips.Zero, mips.Zero))
	noDot := b.p.Branch(mips.Beq, mips.T1, mips.Zero)
	b.p.Emit(
		mips.Lbu(mips.T2, mips.T1, -3),
		mips.Sll(mips.T2, mips.T2, 16),
		mips.Lbu(mips.T3, mips.T1, -2),
		mips.Sll(mips.T3, mips.T3, 8),
		mips.Or(mips.T2, mips.T2, mips.T3),
		mips.Lbu(mips.T3, mips.T1, -1),
		mips.Or(mips.T2, mips.T2, mips.T3),
	)
	noDot()
	b.addr(mips.T1, curKey)
	b.p.Emit(mips.Sw(mips.T2, mips.T1, 0))
	b.addr(mips.T9, a.FileTarget)
	b.p.Emit(
		mips.Lw(mips.RA, mips.SP, 0),
		mips.Lw(mips.S0, mips.SP, 4),
		mips.Jr(mips.T9),
		mips.Addiu(mips.SP, mips.SP, 16),
	)

	if len(b.p.Words) > codeWords {
		return nil, fmt.Errorf("subpatch: code is %d words, room for %d", len(b.p.Words), codeWords)
	}
	for len(b.p.Words) < codeWords {
		b.p.Emit(mips.Nop())
	}

	b.p.Emit(0, 0) // counter, movie key
	var text []byte
	for _, l := range lines {
		b.p.Emit(l.Start, l.End,
			math.Float32bits(l.X), math.Float32bits(l.Y),
			math.Float32bits(l.X+1), math.Float32bits(l.Y+1),
			strings+uint32(len(text))-mainBase, l.Align, l.Movie)
		text = append(text, l.Text...)
		text = append(text, 0)
	}
	code := append(b.p.Bytes(), text...)
	if a.Cave+uint32(len(b.p.Words)*4) != strings {
		return nil, errors.New("subpatch: data layout mismatch")
	}
	if a.Cave+uint32(len(code)) > a.CaveEnd {
		return nil, fmt.Errorf("subpatch: %d bytes, code space is %d", len(code), a.CaveEnd-a.Cave)
	}
	img.Neutralize(a.Cave, a.CaveEnd)
	img.Write(a.Cave, code)
	img.SetWord(a.DrawCall, mips.Jal(a.Cave))
	img.SetWord(a.ExitCall, mips.Jal(exitEntry))
	img.SetWord(a.FileCall, mips.Jal(fileEntry))
	return out, nil
}
