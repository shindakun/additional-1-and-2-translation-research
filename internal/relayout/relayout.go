// Package relayout moves main.dat chunks that need more room to the end of
// the file. See docs/kfa1/0004-text.md.
package relayout

import (
	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
)

// Move relocates chunk unit u to the end of f with n bytes of room. The
// pointers in the chunk keep their values: the chunk table entry keeps the
// old offset as the pointer base, and BOOT.BIN needs subpatch.ChunkBase so
// the loader uses it.
func Move(f *maindat.File, units []maindat.Plain, u, n int) error {
	_, err := f.Relocate(units, u, n)
	return err
}
