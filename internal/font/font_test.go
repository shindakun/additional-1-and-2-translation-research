package font

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
)

func loadFont(t *testing.T) *bnd.File {
	t.Helper()
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	b, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "USRDIR", "Font", "BitFont_Game00.bnd"))
	if err != nil {
		t.Skip(err)
	}
	f, err := bnd.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func entry(t *testing.T, f *bnd.File, name string) []byte {
	t.Helper()
	for _, e := range f.Entries {
		if e.Name == name {
			return e.Data
		}
	}
	t.Fatalf("%s not in font BND", name)
	return nil
}

func TestCCMRoundTrip(t *testing.T) {
	f := loadFont(t)
	raw := entry(t, f, "Ccm.ccm")
	c, err := ReadCCM(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Glyphs) != 7485 || len(c.Ranges) != 103 {
		t.Fatalf("got %d glyphs, %d ranges", len(c.Glyphs), len(c.Ranges))
	}
	if !bytes.Equal(c.Bytes(), raw) {
		t.Fatal("CCM round trip differs")
	}
	i, ok := c.Index('A')
	if !ok || c.Glyphs[i].Advance != 8 || c.Glyphs[i].Page != 0 {
		t.Fatalf("glyph A: %+v", c.Glyphs[i])
	}
}

func TestProportional(t *testing.T) {
	f := loadFont(t)
	c, err := ReadCCM(entry(t, f, "Ccm.ccm"))
	if err != nil {
		t.Fatal(err)
	}
	var tex [2]*Texture
	for i, n := range []string{"Tex00.gim", "Tex01.gim"} {
		if tex[i], err = ReadTexture(entry(t, f, n)); err != nil {
			t.Fatal(err)
		}
	}
	if n := c.Proportional(tex, entry(t, f, "Plt00.plt"), 1, 4); n != 95 {
		t.Fatalf("changed %d glyphs, want 95", n)
	}
	adv := func(r rune) int16 {
		i, _ := c.Index(uint32(r))
		return c.Glyphs[i].Left + c.Glyphs[i].Advance
	}
	if adv('i') >= adv('m') || adv('m') > 9 || adv('i') < 2 {
		t.Fatalf("pen steps: i=%d m=%d", adv('i'), adv('m'))
	}
	t.Logf("pen steps: i=%d l=%d a=%d m=%d W=%d", adv('i'), adv('l'), adv('a'), adv('m'), adv('W'))
}
