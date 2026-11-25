package subpatch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/mips"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/prx"
)

func TestCounter(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	boot, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "SYSDIR", "BOOT.BIN"))
	if err != nil {
		t.Skip(err)
	}
	out, err := Counter(boot, 16, 16, false)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := AddrsFor(boot)
	img, err := prx.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	if img.Word(a.DrawCall) != mips.Jal(a.Cave) {
		t.Fatal("hook not installed")
	}
	for _, r := range img.Relocs {
		if r.Offset >= a.Cave && r.Offset < a.CaveEnd && r.Type != prx.RNone {
			t.Fatalf("relocation %d still active at %#x", r.Type, r.Offset)
		}
	}
	// Outside the code space and the relocation table, nothing changes but
	// the hooked jal.
	orig, _ := prx.Open(append([]byte(nil), boot...))
	diff := 0
	for i := range boot {
		if boot[i] != out[i] {
			off := i - orig.TextFile
			inCave := off >= int(a.Cave) && off < int(a.CaveEnd)
			inHook := (off >= int(a.DrawCall) && off < int(a.DrawCall)+4) ||
				(off >= int(a.ExitCall) && off < int(a.ExitCall)+4)
			if !inCave && !inHook {
				diff++
			}
		}
	}
	if diff != 61 { // one byte per neutralized relocation entry
		t.Fatalf("%d bytes changed outside the code space, want 61", diff)
	}
	if !bytes.Contains(out, []byte("F %d")) {
		t.Fatal("format string missing")
	}
}
