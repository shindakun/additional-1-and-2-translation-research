package font

import (
	"errors"
	"fmt"
	"math"
)

// Encode writes t back into a copy of the font GIM it came from, in the
// same swizzled layout that ReadTexture reads.
func (t *Texture) Encode(gim []byte) []byte {
	const imageBlock = 0x480
	out := append([]byte(nil), gim...)
	dst := out[imageBlock+0x50:]
	i := 0
	for by := range TextureSize / 8 {
		for bx := range TextureSize / 16 {
			for r := range 8 {
				y, x := by*8+r, bx*16
				copy(dst[i:i+16], t.Pix[y*TextureSize+x:y*TextureSize+x+16])
				i += 16
			}
		}
	}
	return out
}

// layers returns, for each palette index, the alpha of the four layers
// (palettes 0-3 of a PLT file). Each index stands for one combination.
func layers(plt []byte) [256][4]byte {
	var s [256][4]byte
	for l := range 4 {
		a := LayerAlpha(plt, l)
		for i := range s {
			s[i][l] = a[i]
		}
	}
	return s
}

// padCell is the slot size for a moved glyph: the 8x16 cell and a blank
// pixel on every side.
const (
	cellW, cellH = 8, 16
	slotW, slotH = cellW + 2, cellH + 2
)

// PadASCII moves the printable ASCII glyphs (0x21-0x7E) to free slots on
// page dst, each with a blank pixel border, and points their texture
// coordinates there. The ASCII cells in the original atlas touch the next
// glyph, so linear filtering at a raised render resolution draws slivers of
// the neighbor; a blank border removes them. Each pixel keeps the state of
// the other three layers. plts[i] is the PLT file of texture i. It returns
// the number of glyphs moved.
func (c *CCM) PadASCII(tex [2]*Texture, plts [2][]byte, dst int) (int, error) {
	if dst < 0 || dst > 7 {
		return 0, fmt.Errorf("font: page %d", dst)
	}
	dt, dl := tex[dst/4], dst%4
	dstStates := layers(plts[dst/4])
	inverse := map[[4]byte]byte{}
	for i := 255; i >= 0; i-- { // lowest index wins
		inverse[dstStates[i]] = byte(i)
	}
	// Free slots: the destination layer is blank over the whole slot.
	used := make([]bool, TextureSize*TextureSize)
	free := func(x, y int) bool {
		for yy := y; yy < y+slotH; yy++ {
			for xx := x; xx < x+slotW; xx++ {
				p := yy*TextureSize + xx
				if used[p] || dstStates[dt.Pix[p]][dl] != 0 {
					return false
				}
			}
		}
		return true
	}
	sx, sy := 0, 0
	next := func() (int, int, bool) {
		for ; sy+slotH <= TextureSize; sy, sx = sy+1, 0 {
			for ; sx+slotW <= TextureSize; sx++ {
				if free(sx, sy) {
					for yy := sy; yy < sy+slotH; yy++ {
						for xx := sx; xx < sx+slotW; xx++ {
							used[yy*TextureSize+xx] = true
						}
					}
					return sx, sy, true
				}
			}
		}
		return 0, 0, false
	}

	n := 0
	for code := uint32(0x21); code <= 0x7E; code++ {
		i, ok := c.Index(code)
		if !ok || i >= len(c.Glyphs) {
			continue
		}
		g := &c.Glyphs[i]
		if g.Page < 0 || g.Page > 7 || int(g.Page) == dst {
			continue
		}
		x0 := int(math.Round(float64(g.U0) * TextureSize))
		y0 := int(math.Round(float64(g.V0) * TextureSize))
		x1 := int(math.Round(float64(g.U1) * TextureSize))
		y1 := int(math.Round(float64(g.V1) * TextureSize))
		if x1-x0 != cellW || y1-y0 != cellH {
			return n, fmt.Errorf("font: glyph %#x is %dx%d, want %dx%d", code, x1-x0, y1-y0, cellW, cellH)
		}
		st, sl := tex[g.Page/4], int(g.Page%4)
		srcStates := layers(plts[g.Page/4])
		x, y, ok := next()
		if !ok {
			return n, errors.New("font: no free space on the destination page")
		}
		for yy := range cellH {
			for xx := range cellW {
				a := srcStates[st.Pix[(y0+yy)*TextureSize+x0+xx]][sl]
				p := (y+1+yy)*TextureSize + x + 1 + xx
				s := dstStates[dt.Pix[p]]
				s[dl] = a
				idx, ok := inverse[s]
				if !ok {
					return n, fmt.Errorf("font: no palette index for layer states %v", s)
				}
				dt.Pix[p] = idx
			}
		}
		g.U0 = float32(x+1) / TextureSize
		g.V0 = float32(y+1) / TextureSize
		g.U1 = float32(x+1+cellW) / TextureSize
		g.V1 = float32(y+1+cellH) / TextureSize
		g.Page = int16(dst)
		n++
	}
	return n, nil
}
