// Package font reads and writes CCM glyph tables and reads the glyph
// textures of a font BND. See docs/kfa1/0005-fonts.md.
package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Glyph is one CCM glyph entry.
type Glyph struct {
	U0, V0, U1, V1 float32
	Left           int16 // added to the pen before drawing
	Width          int16 // drawn width
	Advance        int16 // added to the pen after drawing
	Page           int16 // texture = Page / 4, layer = Page % 4
}

// Range maps Shift-JIS codes First..Last to glyphs from Glyph onward.
type Range struct {
	First, Last, Glyph uint32
}

// CCM is a decoded glyph table. Raw keeps the original bytes; Bytes writes
// the glyph entries back into a copy of it.
type CCM struct {
	Raw    []byte
	Ranges []Range
	Glyphs []Glyph
	table  int
}

// ReadCCM decodes a CCM file.
func ReadCCM(b []byte) (*CCM, error) {
	le := binary.LittleEndian
	if len(b) < 0x20 || le.Uint32(b) != 0x00010000 {
		return nil, errors.New("font: not a CCM file")
	}
	nr, ng := int(le.Uint16(b[0x10:])), int(le.Uint16(b[0x12:]))
	ro, gt := int(le.Uint32(b[0x14:])), int(le.Uint32(b[0x18:]))
	if gt+ng*24 > len(b) || ro+nr*12 > len(b) {
		return nil, fmt.Errorf("font: tables do not fit")
	}
	c := &CCM{Raw: b, table: gt}
	for i := range nr {
		e := b[ro+i*12:]
		c.Ranges = append(c.Ranges, Range{le.Uint32(e), le.Uint32(e[4:]), le.Uint32(e[8:])})
	}
	f := func(o int) float32 { return math.Float32frombits(le.Uint32(b[o:])) }
	s := func(o int) int16 { return int16(le.Uint16(b[o:])) }
	for i := range ng {
		o := gt + i*24
		c.Glyphs = append(c.Glyphs, Glyph{f(o), f(o + 4), f(o + 8), f(o + 12), s(o + 16), s(o + 18), s(o + 20), s(o + 22)})
	}
	return c, nil
}

// Index returns the glyph index for a Shift-JIS code.
func (c *CCM) Index(code uint32) (int, bool) {
	for _, r := range c.Ranges {
		if code >= r.First && code <= r.Last {
			return int(r.Glyph + code - r.First), true
		}
	}
	return 0, false
}

// Bytes encodes the CCM with the current glyph entries.
func (c *CCM) Bytes() []byte {
	le := binary.LittleEndian
	out := append([]byte(nil), c.Raw...)
	for i, g := range c.Glyphs {
		o := c.table + i*24
		le.PutUint32(out[o:], math.Float32bits(g.U0))
		le.PutUint32(out[o+4:], math.Float32bits(g.V0))
		le.PutUint32(out[o+8:], math.Float32bits(g.U1))
		le.PutUint32(out[o+12:], math.Float32bits(g.V1))
		le.PutUint16(out[o+16:], uint16(g.Left))
		le.PutUint16(out[o+18:], uint16(g.Width))
		le.PutUint16(out[o+20:], uint16(g.Advance))
		le.PutUint16(out[o+22:], uint16(g.Page))
	}
	return out
}

// TextureSize is the width and height of a glyph texture.
const TextureSize = 512

// Texture is a decoded 8-bit glyph texture.
type Texture struct {
	Pix []byte // TextureSize*TextureSize palette indices, row-major
}

// ReadTexture decodes the swizzled 512x512 index-8 image of a font GIM.
func ReadTexture(gim []byte) (*Texture, error) {
	const imageBlock = 0x480
	le := binary.LittleEndian
	if len(gim) < imageBlock+0x50 || string(gim[:11]) != "MIG.00.1PSP" || le.Uint16(gim[imageBlock:]) != 4 {
		return nil, errors.New("font: unexpected GIM layout")
	}
	hdr := gim[imageBlock+0x10:]
	format, w, h := le.Uint16(hdr[4:]), int(le.Uint16(hdr[8:])), int(le.Uint16(hdr[10:]))
	if format != 5 || w != TextureSize || h != TextureSize {
		return nil, fmt.Errorf("font: GIM format %d %dx%d", format, w, h)
	}
	src := gim[imageBlock+0x50:]
	if len(src) < w*h {
		return nil, errors.New("font: GIM pixel data short")
	}
	t := &Texture{Pix: make([]byte, w*h)}
	i := 0
	// PSP swizzle: 16-byte by 8-row tiles, left to right, top to bottom.
	for by := range h / 8 {
		for bx := range w / 16 {
			for r := range 8 {
				y, x := by*8+r, bx*16
				copy(t.Pix[y*w+x:y*w+x+16], src[i:i+16])
				i += 16
			}
		}
	}
	return t, nil
}

// LayerAlpha returns the alpha for each palette index of layer l (palette l
// of a PLT file).
func LayerAlpha(plt []byte, l int) [256]byte {
	var a [256]byte
	for i := range a {
		a[i] = plt[l*1024+i*4+3]
	}
	return a
}

// InkColumns returns the first and last columns with ink in a glyph cell,
// relative to the cell, and false for a blank cell.
func InkColumns(t *Texture, alpha [256]byte, g Glyph) (first, last int, ok bool) {
	x0, y0 := int(math.Round(float64(g.U0)*TextureSize)), int(math.Round(float64(g.V0)*TextureSize))
	x1, y1 := int(math.Round(float64(g.U1)*TextureSize)), int(math.Round(float64(g.V1)*TextureSize))
	first, last = -1, -1
	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			if alpha[t.Pix[y*TextureSize+x]] != 0 {
				if first < 0 {
					first = x - x0
				}
				last = x - x0
				break
			}
		}
	}
	return first, last, first >= 0
}
