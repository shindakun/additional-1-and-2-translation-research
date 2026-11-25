// Package images lists the images of a main.dat chunk through its image
// table. See docs/0012-images.md.
package images

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Image types in the table.
const (
	TypeJPEG = 0
	TypeGIM  = 2
	TypeBMP  = 3
)

// Ext returns a file extension for an image type.
func Ext(t uint32) string {
	switch t {
	case TypeJPEG:
		return "jpg"
	case TypeGIM:
		return "gim"
	case TypeBMP:
		return "bmp"
	}
	return "bin"
}

var magic = map[uint32][]byte{
	TypeJPEG: {0xFF, 0xD8},
	TypeGIM:  []byte("MIG.00.1PSP"),
	TypeBMP:  []byte("BM"),
}

// Entry is one image table entry.
type Entry struct {
	Index         int
	Type          uint32
	Off, Size     int // offset in the chunk data, size in bytes
	Width, Height int
	TableOff      int // offset of the table entry in the chunk data
}

// List reads the image table of a decrypted chunk. base is the chunk's file
// offset in main.dat. The table is at the block header +0x18; its length is
// the u16 at +0x0A (GIM count) plus the u16 at +0x0E (other images).
func List(chunk []byte, base uint32) ([]Entry, error) {
	le := binary.LittleEndian
	ptr := le.Uint32(chunk[0x18:])
	if ptr == 0xFFFFFFFF {
		return nil, nil
	}
	count := int(le.Uint16(chunk[0x0A:])) + int(le.Uint16(chunk[0x0E:]))
	t := int(ptr - base)
	var out []Entry
	for k := range count {
		e := t + k*16
		if e+16 > len(chunk) {
			return nil, fmt.Errorf("images: table entry %d past end", k)
		}
		typ, size, fo := le.Uint32(chunk[e:]), int(le.Uint32(chunk[e+4:])), le.Uint32(chunk[e+8:])
		off := int(fo - base)
		if fo < base || off+size > len(chunk) {
			return nil, fmt.Errorf("images: entry %d outside the chunk", k)
		}
		m, ok := magic[typ]
		if !ok || !bytes.HasPrefix(chunk[off:], m) {
			return nil, fmt.Errorf("images: entry %d type %d has wrong magic", k, typ)
		}
		out = append(out, Entry{k, typ, off, size, int(le.Uint16(chunk[e+12:])), int(le.Uint16(chunk[e+14:])), e})
	}
	return out, nil
}

// Replace writes data over image e in chunk, fills the rest of the old space
// with zero bytes and sets the table size. data must not be longer than the
// original image, because the chunk layout does not move.
func Replace(chunk []byte, e Entry, data []byte) error {
	if len(data) > e.Size {
		return fmt.Errorf("images: entry %d: %d bytes, space for %d", e.Index, len(data), e.Size)
	}
	m := magic[e.Type]
	if !bytes.HasPrefix(data, m) {
		return fmt.Errorf("images: entry %d: new data is not type %d", e.Index, e.Type)
	}
	copy(chunk[e.Off:], data)
	clear(chunk[e.Off+len(data) : e.Off+e.Size])
	binary.LittleEndian.PutUint32(chunk[e.TableOff+4:], uint32(len(data)))
	return nil
}
