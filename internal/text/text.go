// Package text finds the translatable strings in main.dat units and writes
// translations back in place. See docs/kfa1/0004-text.md.
package text

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/sjis"
)

const none = 0xFFFFFFFF

// Kind of a slot.
const (
	KindText  = "text"  // NUL-terminated string reached through a text header
	KindName  = "name"  // 0x18-byte name field of a command record
	KindMenu  = "menu"  // 0x1D-byte name field of a system menu record
	KindFlag  = "flag"  // 0x18-byte name field of a flag record (items, stats)
	KindBlock = "block" // 0x20-byte name field of a block header; the save data title
)

// nameField is the size of a command name field; menuField is the size of
// a system menu name field, at record +3.
const (
	nameField  = 0x18
	menuField  = 0x1D
	blockField = 0x20
)

// Slot is one place that holds a string.
type Slot struct {
	Unit  int    // index into the unit list
	Off   int    // offset of the string in the unit's plain data
	Max   int    // bytes available, without the terminating NUL
	Kind  string // KindText or KindName
	Where string // location label, for example c067/s00956/t03
	JA    string // current text, UTF-8, line breaks as \n
	Ptrs  []int  // offsets in the unit of the pointers to a text string
}

type unitView struct {
	data []byte
	base uint32 // file offset of data[0]
}

func (u unitView) u32(off uint32) uint32 {
	return binary.LittleEndian.Uint32(u.data[off-u.base:])
}

func (u unitView) in(addr uint32, n int) bool {
	return addr >= u.base && int(addr-u.base)+n <= len(u.data)
}

// Extract lists every global and scene text and every global and scene
// command name.
func Extract(units []maindat.Plain, c *sjis.Codec) ([]Slot, error) {
	var slots []Slot
	seen := map[[2]int]int{} // unit and offset to slot index + 1
	// add records the string at addr. ptr is the unit offset of the pointer
	// that reaches it, or -1 for a fixed field.
	add := func(ui int, v unitView, addr uint32, kind, where string, ptr int) error {
		if !v.in(addr, 1) {
			return fmt.Errorf("text: %s points outside its unit", where)
		}
		off := int(addr - v.base)
		if i := seen[[2]int{ui, off}]; i != 0 {
			if ptr >= 0 {
				slots[i-1].Ptrs = append(slots[i-1].Ptrs, ptr)
			}
			return nil
		}
		seen[[2]int{ui, off}] = len(slots) + 1
		var raw []byte
		budget := 0
		switch kind {
		case KindName, KindFlag:
			raw = bytes.TrimRight(v.data[off:off+nameField], "\x00")
			budget = nameField - 1
		case KindMenu:
			raw = bytes.TrimRight(v.data[off:off+menuField], "\x00")
			budget = menuField - 1
		case KindBlock:
			raw = v.data[off : off+blockField]
			if z := bytes.IndexByte(raw, 0); z >= 0 {
				raw = raw[:z]
			}
			budget = blockField - 1
		default:
			end := bytes.IndexByte(v.data[off:], 0)
			if end < 0 {
				return fmt.Errorf("text: %s not terminated", where)
			}
			raw, budget = v.data[off:off+end], end
		}
		s, err := c.Decode(raw)
		if err != nil {
			return fmt.Errorf("text: %s: %w", where, err)
		}
		var ptrs []int
		if ptr >= 0 {
			ptrs = []int{ptr}
		}
		slots = append(slots, Slot{ui, off, budget, kind, where, strings.ReplaceAll(s, "\r\n", "\n"), ptrs})
		return nil
	}
	table := func(ui int, v unitView, hdr uint32, prefix string) error {
		if hdr == none || !v.in(hdr, 8) {
			return nil
		}
		n, t := v.u32(hdr), v.u32(hdr+4)
		if t == none {
			return nil
		}
		for k := uint32(0); k < n; k++ {
			if p := v.u32(t + 4*k); p != none {
				if err := add(ui, v, p, KindText, fmt.Sprintf("%s/t%04d", prefix, k), int(t+4*k-v.base)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	names := func(ui int, v unitView, count, recs uint32, prefix string) error {
		if recs == none || count == 0 {
			return nil
		}
		for k := uint32(0); k < count; k++ {
			if err := add(ui, v, recs+k*0x20, KindName, fmt.Sprintf("%s/n%04d", prefix, k), -1); err != nil {
				return err
			}
		}
		return nil
	}

	for ui, u := range units {
		switch {
		case u.Name == "body":
			// Game data at file offset 0x130: +0x0C command count, +0x10
			// command records, +0x14 global text header.
			v := unitView{u.Data, uint32(u.Off)}
			const game = 0x130
			if err := table(ui, v, v.u32(game+0x14), "g"); err != nil {
				return nil, err
			}
			if err := names(ui, v, v.u32(game+0x0C), v.u32(game+0x10), "g"); err != nil {
				return nil, err
			}
			// Flag records: +0x04 count, +0x08 records of 0x20 bytes. Item
			// and stat names show in game. Bytes 0x0C-0x1B are zero in both
			// games, so the name field is 0x18 bytes like a command name.
			for k, r := uint32(0), v.u32(game+0x08); k < v.u32(game+0x04) && r != none; k++ {
				if err := add(ui, v, r+k*0x20, KindFlag, fmt.Sprintf("g/f%04d", k), -1); err != nil {
					return nil, err
				}
			}
			// System menu and key guide: 0x20-byte records from the pointer
			// at +0xC4 up to the address at +0x00, name at record +3.
			end := v.u32(game)
			for k, r := 0, v.u32(game+0xC4); r != none && r+0x20 <= end && v.in(r, 0x20); k, r = k+1, r+0x20 {
				if err := add(ui, v, r+3, KindMenu, fmt.Sprintf("g/m%04d", k), -1); err != nil {
					return nil, err
				}
			}
		case strings.HasPrefix(u.Name, "chunks/"):
			v := unitView{u.Data, uint32(u.Off)}
			ci := strings.TrimSuffix(strings.TrimPrefix(u.Name, "chunks/"), ".bin")[:3]
			// Block name at header +0x20: the save data title.
			if err := add(ui, v, v.base+0x20, KindBlock, "c"+ci+"/b", -1); err != nil {
				return nil, err
			}
			count := uint32(binary.LittleEndian.Uint16(u.Data[0x0C:]))
			list := v.u32(v.base + 0x14)
			for s := uint32(0); s < count; s++ {
				sp := v.u32(list + 4*s)
				if sp == none || !v.in(sp, 0x20) {
					continue
				}
				prefix := fmt.Sprintf("c%s/s%05d", ci, v.u32(sp+4))
				if err := table(ui, v, v.u32(sp+0x18), prefix); err != nil {
					return nil, err
				}
				if err := names(ui, v, v.u32(sp+0x0C), v.u32(sp+0x10), prefix); err != nil {
					return nil, err
				}
			}
		}
	}
	return slots, nil
}

// Entry is one record of a translation file: one unique source string.
type Entry struct {
	JA    string   `json:"ja"`
	EN    string   `json:"en"`
	Kind  string   `json:"kind"`
	Max   int      `json:"max"`   // smallest byte budget among its slots
	Count int      `json:"count"` // number of slots
	Where []string `json:"where"` // up to five slot labels
}

// Entries groups slots by kind and source text, in first-seen order.
func Entries(slots []Slot) []Entry {
	idx := map[[2]string]int{}
	var out []Entry
	for _, s := range slots {
		if strings.TrimSpace(strings.ReplaceAll(s.JA, "　", "")) == "" {
			continue
		}
		key := [2]string{s.Kind, s.JA}
		i, ok := idx[key]
		if !ok {
			i = len(out)
			idx[key] = i
			out = append(out, Entry{JA: s.JA, Kind: s.Kind, Max: s.Max})
		}
		e := &out[i]
		e.Count++
		if s.Max < e.Max {
			e.Max = s.Max
		}
		if len(e.Where) < 5 {
			e.Where = append(e.Where, s.Where)
		}
	}
	return out
}

// Problem is a translation that was not written.
type Problem struct {
	JA, EN, Where, Reason string
}

// Space hands out and takes back free bytes inside units. Alloc returns
// the offset of free space in unit for data (the encoded string and its
// terminator), or false; it can refuse data its space cannot hold, for
// example non-ASCII bytes in texture space. Free returns n bytes at off.
type Space interface {
	Alloc(unit int, data []byte) (int, bool)
	Free(unit, off, n int)
}

// Result counts what Apply did.
type Result struct {
	Written int // slots written in place
	Moved   int // text strings written to new space, pointers changed
}

// Apply writes translations into the unit data. Slots must come from the
// original data: a slot's budget is the length of the original string, and
// extraction after an import measures the shorter text. Every slot whose
// source has a non-empty translation is rewritten; the rest of the slot is
// filled with zero bytes. A text string that does not fit goes to space from
// sp (if sp is not nil), its pointers change to the new address, and its old
// slot goes back to sp. Moves repeat until no more fit, so later moves can
// use the slots that earlier moves left. Translations that do not fit or do
// not encode are returned as problems.
func Apply(units []maindat.Plain, slots []Slot, entries []Entry, c *sjis.Codec, sp Space) (Result, []Problem) {
	en := map[[2]string]string{}
	for _, e := range entries {
		if e.EN != "" {
			en[[2]string{e.Kind, e.JA}] = e.EN
		}
	}
	type move struct {
		s Slot
		t string
		b []byte
	}
	var res Result
	var probs []Problem
	var moves []move
	for _, s := range slots {
		t, ok := en[[2]string{s.Kind, s.JA}]
		if !ok {
			continue
		}
		b, err := c.Encode(strings.ReplaceAll(t, "\n", "\r\n"))
		if err != nil {
			probs = append(probs, Problem{s.JA, t, s.Where, err.Error()})
			continue
		}
		old := units[s.Unit].Data[s.Off : s.Off+s.Max+1]
		if len(b) <= s.Max {
			copy(old, b)
			clear(old[len(b):])
			res.Written++
			continue
		}
		if s.Kind == KindText && len(s.Ptrs) > 0 && sp != nil {
			moves = append(moves, move{s, t, b})
			continue
		}
		probs = append(probs, Problem{s.JA, t, s.Where, fmt.Sprintf("%d bytes, room for %d", len(b), s.Max)})
	}
	// Largest first: big strings need long runs; small ones fill the gaps.
	// A Space with an EnableGrowth method gets a second round after the
	// free space runs out, so units grow only for what does not fit.
	sort.SliceStable(moves, func(i, j int) bool { return len(moves[i].b) > len(moves[j].b) })
	g, canGrow := sp.(interface{ EnableGrowth() })
	for progress := true; len(moves) > 0; {
		if !progress {
			if !canGrow {
				break
			}
			g.EnableGrowth()
			canGrow = false
		}
		progress = false
		var left []move
		for _, m := range moves {
			off, ok := sp.Alloc(m.s.Unit, append(bytes.Clone(m.b), 0))
			if !ok {
				left = append(left, m)
				continue
			}
			data := units[m.s.Unit].Data
			copy(data[off:], m.b)
			data[off+len(m.b)] = 0
			clear(data[m.s.Off : m.s.Off+m.s.Max+1])
			addr := uint32(units[m.s.Unit].Off + off)
			for _, p := range m.s.Ptrs {
				binary.LittleEndian.PutUint32(data[p:], addr)
			}
			sp.Free(m.s.Unit, m.s.Off, m.s.Max+1)
			res.Moved++
			progress = true
		}
		moves = left
	}
	for _, m := range moves {
		probs = append(probs, Problem{m.s.JA, m.t, m.s.Where, fmt.Sprintf("%d bytes, room for %d", len(m.b), m.s.Max)})
	}
	sort.Slice(probs, func(i, j int) bool { return probs[i].Where < probs[j].Where })
	return res, probs
}
