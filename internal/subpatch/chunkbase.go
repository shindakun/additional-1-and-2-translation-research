package subpatch

import (
	"errors"
	"fmt"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/mips"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/prx"
)

// ChunkBase patches the main.dat chunk loader so a chunk can move in the
// file without changing its pointers (docs/kfa1/0004). The loader reads a chunk
// at the offset in its table entry (word 2) and also uses that offset as the
// base of every pointer in the chunk. After the patch, the base comes from
// word 3 of the entry when word 3 is not zero; word 3 is zero on the disc,
// so unmoved chunks load as before.
//
// Two hooks, each a jal to a stub in the second code space. The hooked
// words had no relocation, so the patch reuses two relocations of the code
// space (prx.Reuse) for the new jal instructions:
//
//   - TableLoad (lw s2, 8(a0)): the stub loads s2 the same way and saves
//     the base, word 3 or else word 2, in a data word. The next
//     instruction, lw s3, 0(a0), runs in the delay slot; it does not use
//     s2.
//   - BaseStore (sw s2, 0x2C4(s0)): the stub stores the saved base instead.
//     The next instruction runs in the delay slot and the stub returns to
//     the instruction after it.
//
// Both hooks overwrite ra; the loader saves ra in its prologue and the
// instruction after the second hook is itself a jal. The stubs use t0-t2;
// the loader writes them before it reads them after both hooks.
func ChunkBase(boot []byte) ([]byte, error) {
	a, err := AddrsFor(boot)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), boot...)
	img, err := prx.Open(out)
	if err != nil {
		return nil, err
	}
	if img.Word(a.TableLoad) != mips.Lw(18, mips.A0, 8) {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.TableLoad)
	}
	if img.Word(a.BaseStore) != mips.Sw(18, mips.S0, 0x2C4) {
		return nil, fmt.Errorf("subpatch: unexpected instruction at %#x", a.BaseStore)
	}

	b := &builder{p: mips.Program{Base: a.Cave2}}
	const words = 32
	base := a.Cave2 + words*4

	// Stub A: s2 = entry[2]; base = entry[3] or entry[2].
	loadEntry := b.p.PC()
	b.p.Emit(
		mips.Or(mips.T2, mips.RA, mips.Zero),
		mips.Lw(18, mips.A0, 8),
		mips.Lw(mips.T0, mips.A0, 12),
	)
	set := b.p.Branch(mips.Bne, mips.T0, mips.Zero)
	b.p.Emit(mips.Or(mips.T0, 18, mips.Zero))
	set()
	// The data word sits after the code; take its run-time address from
	// the program counter.
	b.p.Emit(
		mips.Bal(1),
		mips.Nop(),
	)
	pcA := b.p.PC()
	b.p.LoadConst(mips.T1, base-pcA)
	b.p.Emit(
		mips.Addu(mips.T1, mips.T1, mips.RA),
		mips.Sw(mips.T0, mips.T1, 0),
		mips.Jr(mips.T2),
		mips.Nop(),
	)

	// Stub B: store the saved base.
	storeBase := b.p.PC()
	b.p.Emit(
		mips.Or(mips.T2, mips.RA, mips.Zero),
		mips.Bal(1),
		mips.Nop(),
	)
	pcB := b.p.PC()
	b.p.LoadConst(mips.T1, base-pcB)
	b.p.Emit(
		mips.Addu(mips.T1, mips.T1, mips.RA),
		mips.Lw(mips.T0, mips.T1, 0),
		mips.Sw(mips.T0, mips.S0, 0x2C4),
		mips.Jr(mips.T2),
		mips.Nop(),
	)
	if len(b.p.Words) > words {
		return nil, fmt.Errorf("subpatch: chunk-base code is %d words, room for %d", len(b.p.Words), words)
	}
	for len(b.p.Words) < words {
		b.p.Emit(mips.Nop())
	}
	b.p.Emit(0) // saved base
	code := b.p.Bytes()
	if a.Cave2+uint32(len(code)) > a.Cave2End {
		return nil, errors.New("subpatch: chunk-base code does not fit")
	}
	img.Neutralize(a.Cave2, a.Cave2End)
	img.Write(a.Cave2, code)
	// The hooked words had no relocation; a jal needs one. Reuse two of the
	// relocations the code space no longer needs.
	for _, h := range []uint32{a.TableLoad, a.BaseStore} {
		if err := img.Reuse(a.Cave2, a.Cave2End, h, prx.R26); err != nil {
			return nil, err
		}
	}
	img.SetWord(a.TableLoad, mips.Jal(loadEntry))
	img.SetWord(a.BaseStore, mips.Jal(storeBase))
	return out, nil
}
