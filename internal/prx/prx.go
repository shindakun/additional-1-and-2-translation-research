// Package prx reads and edits the relocation table of a PSP PRX (the plain
// BOOT.BIN). See docs/kfa1/0010-subtitle-patch.md.
package prx

import (
	"encoding/binary"
	"errors"
)

// Relocation types used by the King's Field executables.
const (
	RNone = 0
	R32   = 2
	R26   = 4
	RHi16 = 5
	RLo16 = 6
)

const relSectionType = 0x700000A0

// Reloc is one Elf32_Rel entry and its position in the file.
type Reloc struct {
	FileOff int // position of the entry in the file
	Offset  uint32
	Type    byte
}

// Image is a PRX held as its file bytes.
type Image struct {
	Raw []byte
	// TextFile is the file offset of address 0 (the start of segment 0).
	TextFile int
	// TextSize is the size of .text.
	TextSize int
	Relocs   []Reloc
}

// Open parses section headers and relocation sections.
func Open(b []byte) (*Image, error) {
	le := binary.LittleEndian
	if len(b) < 0x34 || string(b[1:4]) != "ELF" {
		return nil, errors.New("prx: not an ELF file")
	}
	shoff, shnum := int(le.Uint32(b[0x20:])), int(le.Uint16(b[0x30:]))
	img := &Image{Raw: b, TextFile: -1}
	for i := range shnum {
		s := b[shoff+i*40:]
		typ, addr, off, size := le.Uint32(s[4:]), le.Uint32(s[12:]), int(le.Uint32(s[16:])), int(le.Uint32(s[20:]))
		switch {
		case typ == 1 && addr == 0 && size > 0 && img.TextFile < 0:
			img.TextFile, img.TextSize = off, size
		case typ == relSectionType:
			for k := 0; k+8 <= size; k += 8 {
				info := le.Uint32(b[off+k+4:])
				img.Relocs = append(img.Relocs, Reloc{FileOff: off + k, Offset: le.Uint32(b[off+k:]), Type: byte(info)})
			}
		}
	}
	if img.TextFile < 0 {
		return nil, errors.New("prx: no .text at address 0")
	}
	return img, nil
}

// Word returns the 32-bit word at a virtual address.
func (img *Image) Word(addr uint32) uint32 {
	return binary.LittleEndian.Uint32(img.Raw[img.TextFile+int(addr):])
}

// SetWord writes a 32-bit word at a virtual address.
func (img *Image) SetWord(addr, w uint32) {
	binary.LittleEndian.PutUint32(img.Raw[img.TextFile+int(addr):], w)
}

// Write copies bytes to a virtual address.
func (img *Image) Write(addr uint32, p []byte) {
	copy(img.Raw[img.TextFile+int(addr):], p)
}

// Neutralize sets every relocation that targets [lo, hi) to R_MIPS_NONE and
// returns how many it changed. The loader then leaves those words alone.
func (img *Image) Neutralize(lo, hi uint32) int {
	n := 0
	for i, r := range img.Relocs {
		if r.Offset >= lo && r.Offset < hi && r.Type != RNone {
			info := binary.LittleEndian.Uint32(img.Raw[r.FileOff+4:])
			binary.LittleEndian.PutUint32(img.Raw[r.FileOff+4:], info&^0xFF)
			img.Relocs[i].Type = RNone
			n++
		}
	}
	return n
}

// RelocAt returns the relocation type applied at an address, or RNone.
func (img *Image) RelocAt(addr uint32) byte {
	for _, r := range img.Relocs {
		if r.Offset == addr {
			return r.Type
		}
	}
	return RNone
}

// Reuse turns one relocation that Neutralize switched off in [lo, hi) into
// a relocation of type typ at addr, with offset and target in segment 0.
// This adds a relocation without growing the table, for example for a jal
// written where the original instruction had none.
func (img *Image) Reuse(lo, hi, addr uint32, typ byte) error {
	for i, r := range img.Relocs {
		if r.Type == RNone && r.Offset >= lo && r.Offset < hi {
			binary.LittleEndian.PutUint32(img.Raw[r.FileOff:], addr)
			binary.LittleEndian.PutUint32(img.Raw[r.FileOff+4:], uint32(typ))
			img.Relocs[i].Offset, img.Relocs[i].Type = addr, typ
			return nil
		}
	}
	return errors.New("prx: no free relocation in the range")
}
