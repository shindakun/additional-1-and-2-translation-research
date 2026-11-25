package gim

import (
	"bytes"
	"image/color"
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

func TestColorAt(t *testing.T) {
	cases := []struct {
		f    int
		v    uint16
		want color.NRGBA
	}{
		{RGBA5551, 0x94A6, color.NRGBA{49, 41, 41, 255}},
		{RGBA5551, 0x7FFF, color.NRGBA{255, 255, 255, 0}},
		{RGBA5650, 0xFFFF, color.NRGBA{255, 255, 255, 255}},
		{RGBA5650, 0x07E0, color.NRGBA{0, 255, 0, 255}},
		{RGBA4444, 0x8F21, color.NRGBA{17, 34, 255, 136}},
	}
	for _, c := range cases {
		b := []byte{byte(c.v), byte(c.v >> 8)}
		if got := colorAt(c.f, b, 0); got != c.want {
			t.Errorf("format %d %#04x: got %v, want %v", c.f, c.v, got, c.want)
		}
	}
}

// TestEncodeRoundTrip encodes every disc GIM from its own decoded image.
// Direct formats give the original bytes; indexed formats give the same
// decoded image (a palette can hold one color twice).
func TestEncodeRoundTrip(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	var files [][]byte
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
			if bytes.HasPrefix(e.Data, magic[:11]) {
				files = append(files, e.Data)
			}
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, "img", "kfaI", "*.gim"))
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, b)
	}
	exact := 0
	for i, b := range files {
		g, err := Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Encode(b, g.Image)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(out, b) {
			exact++
			continue
		}
		if g.Format != Index4 && g.Format != Index8 {
			t.Fatalf("file %d format %d: bytes differ", i, g.Format)
		}
		h, err := Decode(out)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(h.Image.Pix, g.Image.Pix) {
			t.Fatalf("file %d: decoded image differs", i)
		}
	}
	t.Logf("%d GIMs, %d byte-identical", len(files), exact)
}
