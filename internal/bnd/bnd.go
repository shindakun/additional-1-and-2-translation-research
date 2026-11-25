// Package bnd reads and writes FromSoftware BND containers as used by the
// Adventure Player engine. See docs/kfa1/0001-initial-findings.md.
package bnd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Version values seen on the King's Field Additional discs.
const (
	VersionNamed   = 0xFFFF // entries carry names; data aligned to 0x800
	VersionUnnamed = 0x633  // entries carry IDs only; data aligned to 0x10
)

const (
	headerSize = 0x20
	entrySize  = 0x10
)

var magic = []byte("BND\x00")

// Entry is one file in a BND.
type Entry struct {
	ID   uint32
	Name string // empty in the unnamed variant
	Data []byte
}

// File is a decoded BND.
type File struct {
	Version uint32
	Flag    uint32
	// Reserved holds header bytes 0x14-0x1F. They are constant per variant
	// but not understood, so they are carried through unchanged.
	Reserved [12]byte
	Entries  []Entry
}

// Align returns the data alignment for the file's variant.
func (f *File) Align() int {
	if f.Version == VersionNamed {
		return 0x800
	}
	return 0x10
}

// Read decodes a BND.
func Read(b []byte) (*File, error) {
	if len(b) < headerSize || !bytes.Equal(b[:4], magic) {
		return nil, errors.New("bnd: bad magic")
	}
	le := binary.LittleEndian
	f := &File{Version: le.Uint32(b[4:]), Flag: le.Uint32(b[8:])}
	size, count := le.Uint32(b[0xC:]), le.Uint32(b[0x10:])
	copy(f.Reserved[:], b[0x14:0x20])
	if int(size) != len(b) {
		return nil, fmt.Errorf("bnd: header size %#x, data %#x", size, len(b))
	}
	if headerSize+int(count)*entrySize > len(b) {
		return nil, fmt.Errorf("bnd: %d entries do not fit", count)
	}
	for i := 0; i < int(count); i++ {
		e := b[headerSize+i*entrySize:]
		id, off, n, nameOff := le.Uint32(e), le.Uint32(e[4:]), le.Uint32(e[8:]), le.Uint32(e[12:])
		if uint64(off)+uint64(n) > uint64(len(b)) {
			return nil, fmt.Errorf("bnd: entry %d out of range", i)
		}
		ent := Entry{ID: id, Data: b[off : off+n]}
		if nameOff != 0 {
			end := bytes.IndexByte(b[nameOff:], 0)
			if end < 0 {
				return nil, fmt.Errorf("bnd: entry %d name not terminated", i)
			}
			ent.Name = string(b[nameOff : int(nameOff)+end])
		}
		f.Entries = append(f.Entries, ent)
	}
	return f, nil
}

func align(n, a int) int { return (n + a - 1) / a * a }

// Bytes encodes the BND. Names follow the entry table in entry order; data
// follows the names, each entry aligned to the variant's alignment, with
// zero padding. This reproduces every BND on both discs byte for byte.
func (f *File) Bytes() []byte {
	le := binary.LittleEndian
	a := f.Align()
	count := len(f.Entries)
	nameOffs := make([]int, count)
	p := headerSize + count*entrySize
	for i, e := range f.Entries {
		if e.Name != "" {
			nameOffs[i] = p
			p += len(e.Name) + 1
		}
	}
	dataOffs := make([]int, count)
	p = align(p, a)
	for i, e := range f.Entries {
		if i > 0 {
			p = align(p, a)
		}
		dataOffs[i] = p
		p += len(e.Data)
	}
	out := make([]byte, p)
	copy(out, magic)
	le.PutUint32(out[4:], f.Version)
	le.PutUint32(out[8:], f.Flag)
	le.PutUint32(out[0xC:], uint32(len(out)))
	le.PutUint32(out[0x10:], uint32(count))
	copy(out[0x14:0x20], f.Reserved[:])
	for i, e := range f.Entries {
		t := out[headerSize+i*entrySize:]
		le.PutUint32(t, e.ID)
		le.PutUint32(t[4:], uint32(dataOffs[i]))
		le.PutUint32(t[8:], uint32(len(e.Data)))
		le.PutUint32(t[12:], uint32(nameOffs[i]))
		if e.Name != "" {
			copy(out[nameOffs[i]:], e.Name)
		}
		copy(out[dataOffs[i]:], e.Data)
	}
	return out
}
