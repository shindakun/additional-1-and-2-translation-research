package images

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
)

func TestListDiscs(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	for _, g := range []string{"kfaI", "kfaII"} {
		t.Run(g, func(t *testing.T) {
			base := filepath.Join(root, "extract", g, "PSP_GAME")
			boot, err := os.ReadFile(filepath.Join(base, "SYSDIR", "BOOT.BIN"))
			if err != nil {
				t.Skip(err)
			}
			raw, err := os.ReadFile(filepath.Join(base, "USRDIR", "GameData", "01_UmdTitle", "main.dat"))
			if err != nil {
				t.Fatal(err)
			}
			ks, err := maindat.KeysFromBoot(boot)
			if err != nil {
				t.Fatal(err)
			}
			f, err := maindat.Open(raw, ks)
			if err != nil {
				t.Fatal(err)
			}
			_, units, err := f.Unpack()
			if err != nil {
				t.Fatal(err)
			}
			count := map[uint32]int{}
			for _, u := range units {
				if !strings.HasPrefix(u.Name, "chunks/") {
					continue
				}
				list, err := List(u.Data, uint32(u.Off))
				if err != nil {
					t.Fatalf("%s: %v", u.Name, err)
				}
				for _, e := range list {
					count[e.Type]++
				}
			}
			t.Logf("images by type: %v", count)
			if count[TypeJPEG] == 0 || count[TypeGIM] == 0 {
				t.Fatalf("too few images: %v", count)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	// A chunk with a 16-byte image table entry at 0x40 and a JPEG at 0x50.
	chunk := make([]byte, 0x80)
	le := func(o int, v uint32) {
		chunk[o], chunk[o+1], chunk[o+2], chunk[o+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	base := uint32(0x1000)
	le(0x18, base+0x40)
	chunk[0x0E] = 1
	le(0x44, 0x20)
	le(0x48, base+0x50)
	copy(chunk[0x50:], bytes.Repeat([]byte{0xFF, 0xD8, 0xAA}, 11)[:0x20])
	list, err := List(chunk, base)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
	if err := Replace(chunk, list[0], make([]byte, 0x21)); err == nil {
		t.Fatal("longer data accepted")
	}
	if err := Replace(chunk, list[0], []byte{0, 1}); err == nil {
		t.Fatal("wrong magic accepted")
	}
	if err := Replace(chunk, list[0], []byte{0xFF, 0xD8, 0xFF, 0xD9}); err != nil {
		t.Fatal(err)
	}
	got, err := List(chunk, base)
	if err != nil || got[0].Size != 4 {
		t.Fatalf("after replace: %v %v", got, err)
	}
	if !bytes.Equal(chunk[0x50:0x54], []byte{0xFF, 0xD8, 0xFF, 0xD9}) || bytes.ContainsFunc(chunk[0x54:0x70], func(r rune) bool { return r != 0 }) {
		t.Fatalf("data not written or not zero-filled: % x", chunk[0x50:0x70])
	}
}
