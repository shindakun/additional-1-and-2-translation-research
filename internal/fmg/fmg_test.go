package fmg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
)

func TestRoundTripDiscs(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	for _, g := range []string{"kfaI", "kfaII"} {
		b, err := os.ReadFile(filepath.Join(root, "extract", g, "PSP_GAME", "USRDIR", "Bnd", "AdvSys.bnd"))
		if err != nil {
			t.Skip(err)
		}
		c, err := bnd.Read(b)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, e := range c.Entries {
			if !strings.HasSuffix(e.Name, ".fmg") {
				continue
			}
			f, err := Read(e.Data)
			if err != nil {
				t.Fatalf("%s %s: %v", g, e.Name, err)
			}
			if !bytes.Equal(f.Bytes(), e.Data) {
				t.Fatalf("%s %s: rebuild differs", g, e.Name)
			}
			if f.ID(len(f.Strings)-1) < 0 {
				t.Fatalf("%s %s: last string has no ID", g, e.Name)
			}
			n++
		}
		if n == 0 {
			t.Fatalf("%s: no FMG", g)
		}
	}
}

func TestGrow(t *testing.T) {
	f := &File{head: []byte{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Groups: []Group{{0, 100, 102}}, Strings: [][]byte{[]byte("a"), nil, []byte("bc")}}
	g, err := Read(f.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	g.Strings[0] = []byte("a much longer string")
	h, err := Read(g.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(h.Strings[0]) != "a much longer string" || h.Strings[1] != nil || string(h.Strings[2]) != "bc" {
		t.Fatalf("got %q", h.Strings)
	}
	if h.ID(2) != 102 || h.ID(3) != -1 {
		t.Fatalf("IDs %d %d", h.ID(2), h.ID(3))
	}
}
