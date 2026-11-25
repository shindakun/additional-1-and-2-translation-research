package font

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
)

// TestPadASCII moves the ASCII glyphs of the boot font to page 7 and checks
// that each moved glyph has the same ink as the original and a blank border.
func TestPadASCII(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	b, err := os.ReadFile(filepath.Join(root, "extract", "kfaI", "PSP_GAME", "USRDIR", "Font", "BitFont_Game12.bnd"))
	if err != nil {
		t.Skip(err)
	}
	f, err := bnd.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ReadCCM(entry(t, f, "Ccm.ccm"))
	if err != nil {
		t.Fatal(err)
	}
	orig, _ := ReadCCM(entry(t, f, "Ccm.ccm"))
	var tex [2]*Texture
	var plts [2][]byte
	for i, n := range []string{"00", "01"} {
		if tex[i], err = ReadTexture(entry(t, f, "Tex"+n+".gim")); err != nil {
			t.Fatal(err)
		}
		plts[i] = entry(t, f, "Plt"+n+".plt")
	}
	before := [2][]byte{bytes.Clone(tex[0].Pix), bytes.Clone(tex[1].Pix)}
	n, err := c.PadASCII(tex, plts, 7)
	if err != nil {
		t.Fatal(err)
	}
	if n != 94 {
		t.Fatalf("moved %d glyphs, want 94", n)
	}
	// Round trip of the texture encoder.
	g1 := entry(t, f, "Tex01.gim")
	back, err := ReadTexture(tex[1].Encode(g1))
	if err != nil || !bytes.Equal(back.Pix, tex[1].Pix) {
		t.Fatal("texture encode round trip failed")
	}
	px := func(pix []byte, x, y int) byte { return pix[y*TextureSize+x] }
	for code := uint32(0x21); code <= 0x7E; code++ {
		i, _ := c.Index(code)
		g, o := c.Glyphs[i], orig.Glyphs[i]
		so := layers(plts[o.Page/4])
		sn := layers(plts[g.Page/4])
		ox, oy := int(math.Round(float64(o.U0)*TextureSize)), int(math.Round(float64(o.V0)*TextureSize))
		nx, ny := int(math.Round(float64(g.U0)*TextureSize)), int(math.Round(float64(g.V0)*TextureSize))
		for y := -1; y <= cellH; y++ {
			for x := -1; x <= cellW; x++ {
				a := sn[px(tex[g.Page/4].Pix, nx+x, ny+y)][g.Page%4]
				if x < 0 || y < 0 || x == cellW || y == cellH {
					if a != 0 {
						t.Fatalf("glyph %#x: border not blank", code)
					}
					continue
				}
				if want := so[px(before[o.Page/4], ox+x, oy+y)][o.Page%4]; a != want {
					t.Fatalf("glyph %#x: ink differs at %d,%d", code, x, y)
				}
			}
		}
	}
	// Other layers of texture 1 keep their states.
	s1 := layers(plts[1])
	for p := range tex[1].Pix {
		for l := range 3 {
			if s1[tex[1].Pix[p]][l] != s1[before[1][p]][l] {
				t.Fatalf("layer %d changed at pixel %d", l, p)
			}
		}
	}
}
