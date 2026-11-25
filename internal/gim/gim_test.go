package gim

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
)

// TestDiscGIMs decodes every GIM in the menu and system BNDs.
func TestDiscGIMs(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	n := 0
	formats := map[int]int{}
	for _, name := range []string{"AdvMenu.bnd", "AdvSys.bnd"} {
		b, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "USRDIR", "Bnd", name))
		if err != nil {
			t.Skip(err)
		}
		f, err := bnd.Read(b)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range f.Entries {
			if !bytes.HasPrefix(e.Data, magic[:11]) {
				continue
			}
			img, err := Decode(e.Data)
			if err != nil {
				t.Fatalf("%s/%s: %v", name, e.Name, err)
			}
			if img.Width == 0 || img.Height == 0 {
				t.Fatalf("%s/%s: empty", name, e.Name)
			}
			formats[img.Format]++
			n++
		}
	}
	t.Logf("%d GIMs decoded, formats %v", n, formats)
}
