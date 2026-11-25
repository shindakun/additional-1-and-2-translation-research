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

func TestSubtitles(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	boot, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "SYSDIR", "BOOT.BIN"))
	if err != nil {
		t.Skip(err)
	}
	if _, err := Subtitles(boot, []Line{{Start: 1, End: 2, Text: "100%"}}); err == nil {
		t.Fatal("percent sign accepted")
	}
	if _, err := Subtitles(boot, []Line{{Start: 1, End: 2, Text: "é"}}); err == nil {
		t.Fatal("non-ASCII accepted")
	}
	lines := []Line{
		{Start: 10, End: 20, X: 240, Y: 256, Align: AlignCenter, Text: "first"},
		{Start: 15, End: 30, X: 8, Y: 100, Align: AlignLeft, Text: "second"},
	}
	out, err := Subtitles(boot, lines)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := AddrsFor(boot)
	img, err := prx.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	if img.Word(a.DrawCall) != mips.Jal(a.Cave) {
		t.Fatal("draw hook not set")
	}
	// The table follows the code and the counter.
	rec := a.Cave + 224*4 + 8
	if img.Word(rec) != 10 || img.Word(rec+4) != 20 || img.Word(rec+28) != AlignCenter {
		t.Fatalf("first record: %d %d %#x", img.Word(rec), img.Word(rec+4), img.Word(rec+28))
	}
	if img.Word(rec+36) != 15 || img.Word(rec+36+28) != AlignLeft {
		t.Fatal("second record wrong")
	}
	for i := uint32(0); i < 64; i++ { // a long list still fits the code space
		lines = append(lines, Line{Start: i, End: i + 1, Text: "a line of fifty characters for the size check ok."})
	}
	if _, err := Subtitles(boot, lines); err != nil {
		t.Fatal(err)
	}
}

func TestMovieKey(t *testing.T) {
	for _, c := range []struct {
		name string
		want uint32
	}{
		{"disc0:/PSP_GAME/USRDIR/Movie/movie000.pmf", 0x303030},
		{"movie011.pmf", 0x303131},
		{"x.pmf", 0},
		{"none", 0},
	} {
		if got := MovieKey(c.name); got != c.want {
			t.Errorf("%s: got %#x, want %#x", c.name, got, c.want)
		}
	}
}
