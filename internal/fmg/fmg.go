// Package fmg reads and writes FMG message tables. See docs/kfa1/0013-fmg.md.
package fmg

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Group maps a run of IDs to consecutive string indexes.
type Group struct {
	Index, First, Last int32
}

// File is a parsed FMG. Strings holds raw Shift-JIS bytes by index; nil means
// no string (offset 0).
type File struct {
	head    []byte // bytes before the group table
	Groups  []Group
	Strings [][]byte
}

// Read parses an FMG.
func Read(b []byte) (*File, error) {
	le := binary.LittleEndian
	if len(b) < 0x1C || le.Uint32(b) != 0x10000 {
		return nil, errors.New("fmg: bad magic")
	}
	groups, count, table := int(le.Uint32(b[0xC:])), int(le.Uint32(b[0x10:])), int(le.Uint32(b[0x14:]))
	if 0x1C+groups*12 > len(b) || table+count*4 > len(b) || table < 0x1C+groups*12 {
		return nil, errors.New("fmg: tables past end")
	}
	f := &File{head: append([]byte(nil), b[:0x1C]...)}
	for g := range groups {
		o := 0x1C + g*12
		f.Groups = append(f.Groups, Group{int32(le.Uint32(b[o:])), int32(le.Uint32(b[o+4:])), int32(le.Uint32(b[o+8:]))})
	}
	for i := range count {
		o := int(le.Uint32(b[table+i*4:]))
		if o == 0 {
			f.Strings = append(f.Strings, nil)
			continue
		}
		end := o
		for end < len(b) && b[end] != 0 {
			end++
		}
		if end >= len(b) {
			return nil, fmt.Errorf("fmg: string %d not terminated", i)
		}
		f.Strings = append(f.Strings, append([]byte{}, b[o:end]...))
	}
	return f, nil
}

// ID returns the message ID of string index i, or -1.
func (f *File) ID(i int) int {
	for _, g := range f.Groups {
		if int(g.Index) <= i && i <= int(g.Index)+int(g.Last-g.First) {
			return int(g.First) + i - int(g.Index)
		}
	}
	return -1
}

// Bytes builds the file: header, group table, offset table, then the strings
// in index order, each with a zero byte, padded to 16 bytes.
func (f *File) Bytes() []byte {
	le := binary.LittleEndian
	table := 0x1C + len(f.Groups)*12
	out := make([]byte, table+len(f.Strings)*4)
	copy(out, f.head)
	for g, gr := range f.Groups {
		o := 0x1C + g*12
		le.PutUint32(out[o:], uint32(gr.Index))
		le.PutUint32(out[o+4:], uint32(gr.First))
		le.PutUint32(out[o+8:], uint32(gr.Last))
	}
	for i, s := range f.Strings {
		if s == nil {
			continue
		}
		le.PutUint32(out[table+i*4:], uint32(len(out)))
		out = append(out, s...)
		out = append(out, 0)
	}
	for len(out)%16 != 0 {
		out = append(out, 0)
	}
	le.PutUint32(out[4:], uint32(len(out)))
	le.PutUint32(out[0xC:], uint32(len(f.Groups)))
	le.PutUint32(out[0x10:], uint32(len(f.Strings)))
	le.PutUint32(out[0x14:], uint32(table))
	return out
}
