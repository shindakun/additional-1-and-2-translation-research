// Package gim decodes PSP GIM images: direct 16- and 32-bit formats and 4-
// and 8-bit indexed formats with a palette, in normal or swizzled order.
package gim

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
)

// Pixel formats.
const (
	RGBA5650 = 0
	RGBA5551 = 1
	RGBA4444 = 2
	RGBA8888 = 3
	Index4   = 4
	Index8   = 5
)

var magic = []byte("MIG.00.1PSP\x00")

// Image is one decoded picture.
type Image struct {
	Format, Order int
	Width, Height int
	PaletteFormat int
	ImageOffset   int // file offset of the pixel data
	PaletteOffset int // file offset of the palette data, or -1
	Palette       []color.NRGBA
	Image         *image.NRGBA
}

type block struct {
	id         uint16
	off, size  int
	dataOffset int
}

func blocks(b []byte, off, end int) ([]block, error) {
	var out []block
	for off < end {
		if off+16 > len(b) {
			return nil, errors.New("gim: block header past end")
		}
		id := binary.LittleEndian.Uint16(b[off:])
		size := int(binary.LittleEndian.Uint32(b[off+4:]))
		data := int(binary.LittleEndian.Uint32(b[off+12:]))
		if id == 0 && size == 0 {
			break // zero padding after the last block
		}
		if size < 16 || off+size > len(b) {
			return nil, fmt.Errorf("gim: bad block size %#x at %#x", size, off)
		}
		out = append(out, block{id, off, size, off + data})
		off += size
	}
	return out, nil
}

// header is the image or palette header at the start of a block's data.
type header struct {
	format, order, width, height int
	pixelStart                   int
}

func readHeader(b []byte, at int) header {
	le := binary.LittleEndian
	return header{
		format:     int(le.Uint16(b[at+4:])),
		order:      int(le.Uint16(b[at+6:])),
		width:      int(le.Uint16(b[at+8:])),
		height:     int(le.Uint16(b[at+10:])),
		pixelStart: int(le.Uint32(b[at+0x1C:])),
	}
}

func bitsPerPixel(f int) int {
	switch f {
	case RGBA5650, RGBA5551, RGBA4444:
		return 16
	case RGBA8888:
		return 32
	case Index4:
		return 4
	case Index8:
		return 8
	}
	return 0
}

// strideBytes is the row length in bytes: rows are padded to 16 bytes.
func strideBytes(f, w int) int {
	return (w*bitsPerPixel(f)/8 + 15) &^ 15
}

func colorAt(f int, b []byte, i int) color.NRGBA {
	le := binary.LittleEndian
	switch f {
	case RGBA8888:
		return color.NRGBA{b[i*4], b[i*4+1], b[i*4+2], b[i*4+3]}
	case RGBA5650:
		v := int(le.Uint16(b[i*2:]))
		return color.NRGBA{scale(v, 5), scale(v>>5, 6), scale(v>>11, 5), 255}
	case RGBA5551:
		v := int(le.Uint16(b[i*2:]))
		return color.NRGBA{scale(v, 5), scale(v>>5, 5), scale(v>>10, 5), uint8(255 * (v >> 15))}
	case RGBA4444:
		v := int(le.Uint16(b[i*2:]))
		return color.NRGBA{scale(v, 4), scale(v>>4, 4), scale(v>>8, 4), scale(v>>12, 4)}
	}
	return color.NRGBA{}
}

// scale expands the low bits of v to 0-255.
func scale(v, bits int) uint8 {
	m := 1<<bits - 1
	return uint8((v & m) * 255 / m)
}

// unswizzle turns swizzled pixel data (16-byte by 8-row tiles) into rows.
func unswizzle(src []byte, stride, height int) []byte {
	out := make([]byte, stride*height)
	rowsOfTiles := (height + 7) / 8
	i := 0
	for ty := range rowsOfTiles {
		for tx := 0; tx < stride/16; tx++ {
			for r := range 8 {
				y := ty*8 + r
				if y < height && i+16 <= len(src) {
					copy(out[y*stride+tx*16:], src[i:i+16])
				}
				i += 16
			}
		}
	}
	return out
}

// Decode reads the first picture of a GIM file.
func Decode(b []byte) (*Image, error) {
	if len(b) < 16 || string(b[:12]) != string(magic) {
		return nil, errors.New("gim: bad magic")
	}
	// The root block's size gives the end of the GIM; data can follow it.
	end := len(b)
	if len(b) >= 24 {
		end = min(end, 16+int(binary.LittleEndian.Uint32(b[20:])))
	}
	root, err := blocks(b, 16, end)
	if err != nil {
		return nil, err
	}
	var img, pal *block
	var walk func([]block) error
	walk = func(bs []block) error {
		for i := range bs {
			switch bs[i].id {
			case 2, 3:
				sub, err := blocks(b, bs[i].dataOffset, bs[i].off+bs[i].size)
				if err != nil {
					return err
				}
				if err := walk(sub); err != nil {
					return err
				}
			case 4:
				if img == nil {
					img = &bs[i]
				}
			case 5:
				if pal == nil {
					pal = &bs[i]
				}
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	if img == nil {
		return nil, errors.New("gim: no image block")
	}
	ih := readHeader(b, img.dataOffset)
	out := &Image{Format: ih.format, Order: ih.order, Width: ih.width, Height: ih.height,
		ImageOffset: img.dataOffset + ih.pixelStart, PaletteOffset: -1}
	if bitsPerPixel(ih.format) == 0 {
		return nil, fmt.Errorf("gim: pixel format %d not supported", ih.format)
	}
	stride := strideBytes(ih.format, ih.width)
	alignedH := (ih.height + 7) &^ 7
	if out.ImageOffset+stride*ih.height > len(b) {
		return nil, errors.New("gim: pixel data past end")
	}
	px := b[out.ImageOffset:min(len(b), out.ImageOffset+stride*alignedH)]
	if ih.order == 1 {
		px = unswizzle(px, stride, ih.height)
	}

	var palette []color.NRGBA
	if ih.format == Index4 || ih.format == Index8 {
		if pal == nil {
			return nil, errors.New("gim: indexed image without palette")
		}
		ph := readHeader(b, pal.dataOffset)
		out.PaletteFormat, out.PaletteOffset = ph.format, pal.dataOffset+ph.pixelStart
		for i := 0; i < ph.width*max(ph.height, 1); i++ {
			palette = append(palette, colorAt(ph.format, b[out.PaletteOffset:], i))
		}
	}

	out.Palette = palette
	dst := image.NewNRGBA(image.Rect(0, 0, ih.width, ih.height))
	for y := range ih.height {
		row := px[y*stride:]
		for x := range ih.width {
			var c color.NRGBA
			switch ih.format {
			case Index8:
				if int(row[x]) < len(palette) {
					c = palette[row[x]]
				}
			case Index4:
				v := row[x/2] >> (4 * uint(x&1)) & 15
				if int(v) < len(palette) {
					c = palette[v]
				}
			default:
				c = colorAt(ih.format, row, x)
			}
			dst.SetNRGBA(x, y, c)
		}
	}
	out.Image = dst
	return out, nil
}

// swizzle is the inverse of unswizzle.
func swizzle(src []byte, stride, height int) []byte {
	out := make([]byte, len(src))
	i := 0
	for ty := 0; ty < (height+7)/8; ty++ {
		for tx := 0; tx < stride/16; tx++ {
			for r := range 8 {
				y := ty*8 + r
				if y < height && i+16 <= len(out) {
					copy(out[i:i+16], src[y*stride+tx*16:])
				}
				i += 16
			}
		}
	}
	return out
}

func unscale(c uint8, bits int) int {
	m := 1<<bits - 1
	return (int(c)*m + 127) / 255
}

func pack(f int, c color.NRGBA) uint32 {
	switch f {
	case RGBA5650:
		return uint32(unscale(c.R, 5) | unscale(c.G, 6)<<5 | unscale(c.B, 5)<<11)
	case RGBA5551:
		a := 0
		if c.A >= 128 {
			a = 1
		}
		return uint32(unscale(c.R, 5) | unscale(c.G, 5)<<5 | unscale(c.B, 5)<<10 | a<<15)
	case RGBA4444:
		return uint32(unscale(c.R, 4) | unscale(c.G, 4)<<4 | unscale(c.B, 4)<<8 | unscale(c.A, 4)<<12)
	}
	return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16 | uint32(c.A)<<24
}

func nearest(pal []color.NRGBA, c color.NRGBA) int {
	best, bd := 0, -1
	for i, p := range pal {
		dr, dg, db, da := int(p.R)-int(c.R), int(p.G)-int(c.G), int(p.B)-int(c.B), int(p.A)-int(c.A)
		if d := dr*dr + dg*dg + db*db + 2*da*da; bd < 0 || d < bd {
			best, bd = i, d
		}
	}
	return best
}

// Encode writes img into a copy of the GIM file orig. The format, swizzle and
// palette stay; indexed images use the nearest palette color. img must have
// the GIM's size. Row padding keeps the original bytes.
func Encode(orig []byte, img image.Image) ([]byte, error) {
	g, err := Decode(orig)
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != g.Width || img.Bounds().Dy() != g.Height {
		return nil, fmt.Errorf("gim: image is %v, want %dx%d", img.Bounds().Size(), g.Width, g.Height)
	}
	var pal []color.NRGBA
	if g.Format == Index4 || g.Format == Index8 {
		pal = g.Palette
	}
	stride := strideBytes(g.Format, g.Width)
	alignedH := (g.Height + 7) &^ 7
	end := min(len(orig), g.ImageOffset+stride*alignedH)
	out := append([]byte(nil), orig...)
	px := append([]byte(nil), orig[g.ImageOffset:end]...)
	if g.Order == 1 {
		px = unswizzle(px, stride, alignedH)[:len(px)]
	}
	b := img.Bounds()
	le := binary.LittleEndian
	for y := range g.Height {
		row := px[y*stride:]
		for x := range g.Width {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			switch g.Format {
			case Index8:
				row[x] = byte(nearest(pal, c))
			case Index4:
				v := byte(nearest(pal, c))
				sh := 4 * uint(x&1)
				row[x/2] = row[x/2]&^(15<<sh) | v<<sh
			case RGBA8888:
				le.PutUint32(row[x*4:], pack(g.Format, c))
			default:
				le.PutUint16(row[x*2:], uint16(pack(g.Format, c)))
			}
		}
	}
	if g.Order == 1 {
		px = swizzle(px, stride, alignedH)
	}
	copy(out[g.ImageOffset:end], px)
	return out, nil
}
