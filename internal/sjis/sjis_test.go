package sjis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
)

// Load builds a codec from AdvSys.bnd under KFA_DATA, or skips.
func load(t *testing.T) *Codec {
	t.Helper()
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	b, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "USRDIR", "Bnd", "AdvSys.bnd"))
	if err != nil {
		t.Skip(err)
	}
	f, err := bnd.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	var j, u []byte
	for _, e := range f.Entries {
		switch e.Name {
		case "jis2ucs.bin":
			j = e.Data
		case "ucs2jis.bin":
			u = e.Data
		}
	}
	c, err := New(j, u)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRoundTrip(t *testing.T) {
	c := load(t)
	for _, s := range []string{"あ漢ー・「」～①￥", "鍵がかかっている\r\nあけるには鍵が必要だ", "ABC 123", "ｷｬﾝｾﾙ", "#c[FFFF00]警告#c"} {
		b, err := c.Encode(s)
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		if got != s {
			t.Fatalf("%q -> % x -> %q", s, b, got)
		}
	}
	b, _ := c.Encode("あ")
	if b[0] != 0x82 || b[1] != 0xA0 {
		t.Fatalf("あ encodes to % x, want 82 a0", b)
	}
	if _, err := c.Encode("😀"); err == nil {
		t.Fatal("emoji encoded")
	}
}
