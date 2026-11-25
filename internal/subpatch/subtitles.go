package subpatch

import (
	"errors"
	"fmt"
	"math"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/mips"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/prx"
)

// Line is one subtitle line: ASCII text drawn from Start up to, but not
// including, End: microseconds since the first frame of the movie. X and Y are screen pixels; Align
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

// outline lists the draw passes: black at the eight one-pixel offsets, then
// white. A single offset shadow shows as a second, shifted font.
var outline = func() []struct {
	color  uint32
	dx, dy float32
} {
	type pass = struct {
		color  uint32
		dx, dy float32
	}
	var p []pass
	for _, d := range [][2]float32{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
		p = append(p, pass{0xFF000000, d[0], d[1]})
	}
	return append(p, pass{0xFFFFFFFF, 0, 0})
}()

// recordWords is the size of one line record: start, end, x, y, the text
// offset from the code base, the alignment and the movie key.
const recordWords = 7

// Subtitles patches boot to draw lines over movies. A frame counter starts
// at 0 for each movie (the exit hook resets it). Every frame, each line
// whose range holds the clock is drawn in white with a one-pixel black
// outline. A line with a Movie key shows only over that movie.
// The opening
// at boot has no font, so it shows nothing (docs/0010).
//
// With bootFont, a fourth hook loads Font/BitFont_Game12.bnd before the
// opening at boot starts, and the exit hook removes it after that movie, so
// the title loads its own font as before (docs/0010).
func Subtitles(boot []byte, lines []Line, bootFont bool) ([]byte, error) {
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
	if bootFont && (img.Word(a.BootCall) != mips.Jal(a.BootTarget) || img.RelocAt(a.BootCall) != prx.R26) {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.BootCall)
	}

	const codeWords = 352
	const frame = 56
	data := a.Cave + codeWords*4
	counter, curKey, loaded, bndPtr, startT, elapsed := data, data+4, data+8, data+12, data+16, data+20
	passes := data + 24
	table := passes + uint32(len(outline)*12)
	fontPath := table + uint32(len(lines)*recordWords*4)
	const fontName = "$(AppDataRoot)Font/BitFont_Game12.bnd\x00"

	b := &builder{p: mips.Program{Base: a.Cave}}
	b.p.Emit(
		mips.Addiu(mips.SP, mips.SP, -frame),
		mips.Sw(mips.RA, mips.SP, 32),
		mips.Sw(mips.S0, mips.SP, 36),
		mips.Sw(mips.S1, mips.SP, 40),
		mips.Sw(mips.S2, mips.SP, 44),
		mips.Sw(mips.S3, mips.SP, 48),
		mips.Sw(mips.S4, mips.SP, 52),
		mips.Bal(1),
		mips.Nop(),
	)
	b.base = b.p.PC()
	b.p.Emit(mips.Or(mips.S0, mips.RA, mips.Zero))
	b.call(a.MovieDraw) // a0 still holds the movie object

	// Clock: the counter word is 0 until the first frame of a movie; then
	// the start time is stored and elapsed = now - start, in microseconds.
	// System time follows the emulated clock and the movie audio, so the
	// lines stay in step when frames are dropped.
	b.addr(mips.T1, counter)
	b.p.Emit(mips.Lw(mips.T2, mips.T1, 0))
	started := b.p.Branch(mips.Bne, mips.T2, mips.Zero)
	b.call(a.SysTime)
	b.addr(mips.T1, startT)
	b.p.Emit(mips.Sw(mips.V0, mips.T1, 0))
	b.addr(mips.T1, counter)
	b.p.Emit(mips.Addiu(mips.T2, mips.Zero, 1), mips.Sw(mips.T2, mips.T1, 0))
	started()
	b.call(a.SysTime)
	b.addr(mips.T1, startT)
	b.p.Emit(mips.Lw(mips.T2, mips.T1, 0), mips.Subu(mips.T2, mips.V0, mips.T2))
	b.addr(mips.T1, elapsed)
	b.p.Emit(mips.Sw(mips.T2, mips.T1, 0))
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
	b.addr(mips.T2, elapsed)
	b.p.Emit(
		mips.Lw(mips.T2, mips.T2, 0),
		mips.Lw(mips.T0, mips.S1, 0),
		mips.Lw(mips.T1, mips.S1, 4),
		mips.Sltu(mips.T3, mips.T2, mips.T0), // counter < start
	)
	early := b.p.Branch(mips.Bne, mips.T3, mips.Zero)
	b.p.Emit(mips.Sltu(mips.T3, mips.T2, mips.T1)) // counter < end
	late := b.p.Branch(mips.Beq, mips.T3, mips.Zero)
	b.p.Emit(mips.Lw(mips.T0, mips.S1, 24))
	anyMovie := b.p.Branch(mips.Beq, mips.T0, mips.Zero)
	b.addr(mips.T1, curKey)
	b.p.Emit(mips.Lw(mips.T1, mips.T1, 0))
	other := b.p.Branch(mips.Bne, mips.T0, mips.T1)
	anyMovie()
	b.p.Emit(mips.Lw(mips.A0, mips.S1, 20))
	b.call(a.SetAlign)
	// Outline: black at the eight one-pixel offsets, then white at the
	// position. The passes table holds color, dx, dy (floats) for each pass.
	b.addr(mips.S3, passes)
	b.p.Emit(mips.Addiu(mips.S4, mips.Zero, int32(len(outline))))
	passLoop := len(b.p.Words)
	b.p.Emit(mips.Lw(mips.A0, mips.S3, 0))
	b.call(a.SetColor)
	b.p.Emit(
		mips.Lw(mips.T0, mips.S1, 8),
		mips.Mtc1(mips.T0, 12),
		mips.Lw(mips.T0, mips.S1, 12),
		mips.Mtc1(mips.T0, 13),
		mips.Lw(mips.T0, mips.S3, 4),
		mips.Mtc1(mips.T0, 14),
		mips.AddS(12, 12, 14),
		mips.Lw(mips.T0, mips.S3, 8),
		mips.Mtc1(mips.T0, 14),
		mips.AddS(13, 13, 14),
		mips.Lw(mips.A0, mips.S1, 16),
		mips.Addu(mips.A0, mips.A0, mips.S0),
	)
	b.call(a.DrawText)
	b.p.Emit(
		mips.Addiu(mips.S3, mips.S3, 12),
		mips.Addiu(mips.S4, mips.S4, -1),
	)
	passBack := len(b.p.Words)
	b.p.Emit(mips.Bne(mips.S4, mips.Zero, int32(passLoop-passBack-1)), mips.Nop())
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
		mips.Lw(mips.S3, mips.SP, 48),
		mips.Lw(mips.S4, mips.SP, 52),
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
	b.p.Emit(mips.Sw(mips.Zero, mips.T1, 0))
	// Remove the boot font once: 2 means loaded by the patch, 3 removed.
	b.addr(mips.T1, loaded)
	b.p.Emit(mips.Lw(mips.T2, mips.T1, 0), mips.Addiu(mips.T3, mips.Zero, 2))
	notOurs := b.p.Branch(mips.Bne, mips.T2, mips.T3)
	b.p.Emit(mips.Addiu(mips.T2, mips.Zero, 3), mips.Sw(mips.T2, mips.T1, 0))
	b.addr(mips.T1, a.FontPtr)
	b.p.Emit(mips.Lw(mips.A0, mips.T1, 0), mips.Or(mips.A1, mips.Zero, mips.Zero))
	b.call(a.DelFont)
	b.addr(mips.T1, a.FontPtr)
	b.p.Emit(mips.Lw(mips.A0, mips.T1, 0))
	b.p.LoadConst(mips.T1, 0x401C0) // slot 0 binder inside CFontMng
	b.p.Emit(mips.Addu(mips.A0, mips.A0, mips.T1))
	b.call(a.BinderDelete)
	notOurs()
	b.p.Emit(
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

	// Fourth entry, from the boot function: load the game font, then start
	// movie000 as the original call does.
	var bootEntry uint32
	if bootFont {
		bootEntry = b.p.PC()
		b.p.Emit(
			mips.Addiu(mips.SP, mips.SP, -frame),
			mips.Sw(mips.RA, mips.SP, 32),
			mips.Sw(mips.S0, mips.SP, 36),
			mips.Sw(mips.A0, mips.SP, 40),
			mips.Bal(1),
			mips.Nop(),
		)
		b.base = b.p.PC()
		b.p.Emit(mips.Or(mips.S0, mips.RA, mips.Zero))
		b.addr(mips.A1, bndPtr)
		b.p.Emit(mips.Sw(mips.Zero, mips.A1, 0))
		b.addr(mips.A0, a.FileSystem)
		b.addr(mips.A2, fontPath)
		b.p.Emit( // the remaining arguments as CBinder::LoadFile passes them
			mips.Or(mips.A3, mips.Zero, mips.Zero),
			mips.Addiu(mips.T0, mips.Zero, 1),
			mips.Or(mips.T1, mips.Zero, mips.Zero),
			mips.Or(mips.T2, mips.Zero, mips.Zero),
			mips.Or(mips.T3, mips.Zero, mips.Zero),
			mips.Sw(mips.Zero, mips.SP, 0),
		)
		b.call(a.AllocRead)
		b.addr(mips.T1, bndPtr)
		b.p.Emit(mips.Lw(mips.A2, mips.T1, 0))
		noData := b.p.Branch(mips.Beq, mips.A2, mips.Zero)
		b.addr(mips.T1, a.FontPtr)
		b.p.Emit(mips.Lw(mips.A0, mips.T1, 0))
		noMng := b.p.Branch(mips.Beq, mips.A0, mips.Zero)
		b.p.Emit(
			mips.Or(mips.A1, mips.Zero, mips.Zero),
			mips.Addiu(mips.A3, mips.Zero, 1),
			mips.Or(mips.T0, mips.Zero, mips.Zero),
			mips.Or(mips.T1, mips.Zero, mips.Zero),
			mips.Or(mips.T2, mips.Zero, mips.Zero),
			mips.Or(mips.T3, mips.Zero, mips.Zero),
		)
		b.call(a.SetFont)
		b.addr(mips.T1, loaded)
		b.p.Emit(mips.Addiu(mips.T2, mips.Zero, 2), mips.Sw(mips.T2, mips.T1, 0))
		noData()
		noMng()
		b.p.Emit(mips.Lw(mips.A0, mips.SP, 40))
		b.call(a.BootTarget)
		b.p.Emit(
			mips.Lw(mips.RA, mips.SP, 32),
			mips.Lw(mips.S0, mips.SP, 36),
			mips.Jr(mips.RA),
			mips.Addiu(mips.SP, mips.SP, frame),
		)
	}

	if len(b.p.Words) > codeWords {
		return nil, fmt.Errorf("subpatch: code is %d words, room for %d", len(b.p.Words), codeWords)
	}
	for len(b.p.Words) < codeWords {
		b.p.Emit(mips.Nop())
	}

	b.p.Emit(0, 0, 0, 0, 0, 0) // started, movie key, boot font state, font BND pointer, start time, elapsed
	for _, o := range outline {
		b.p.Emit(o.color, math.Float32bits(o.dx), math.Float32bits(o.dy))
	}
	text := []byte(fontName)
	at := map[string]uint32{} // equal texts share one copy
	for _, l := range lines {
		off, ok := at[l.Text]
		if !ok {
			off = fontPath + uint32(len(text)) - mainBase
			at[l.Text] = off
			text = append(text, l.Text...)
			text = append(text, 0)
		}
		b.p.Emit(l.Start, l.End,
			math.Float32bits(l.X), math.Float32bits(l.Y),
			off, l.Align, l.Movie)
	}
	code := append(b.p.Bytes(), text...)
	if a.Cave+uint32(len(b.p.Words)*4) != fontPath {
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
	if bootFont {
		img.SetWord(a.BootCall, mips.Jal(bootEntry))
	}
	return out, nil
}
