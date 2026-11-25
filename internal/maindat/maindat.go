// Package maindat reads and writes the ADVP container (main.dat, .adv).
// See docs/kfa1/0003-maindat.md.
package maindat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// keyTables gives the virtual addresses of the encrypted key table and its
// AES key in each known BOOT.BIN, by the file's SHA-1.
var keyTables = map[string][2]int{
	"e62c3559fc94e9c917081bdf5bbf8dd5713ef1ae": {0x12C85C, 0x12C918}, // Additional I
	"e832e575aef3cb4cf1e68ec3104c970a0191ab11": {0x12CA5C, 0x12CB18}, // Additional II
	"c37b28402bc0405c626be1b01329fd305838f20d": {0x123944, 0x123A10}, // Adventure Player
}

// elfDataDelta converts a virtual address to a BOOT.BIN file offset.
const elfDataDelta = 0x80

// Keys is one key set from the decrypted key table.
type Keys struct {
	Salt, Key, IV [16]byte
}

// KeysFromBoot derives both key sets from a known BOOT.BIN.
func KeysFromBoot(boot []byte) ([2]Keys, error) {
	var ks [2]Keys
	sum := sha1.Sum(boot)
	addr, ok := keyTables[hex.EncodeToString(sum[:])]
	if !ok {
		return ks, errors.New("maindat: unknown BOOT.BIN")
	}
	table := boot[addr[0]+elfDataDelta : addr[0]+elfDataDelta+0x80]
	block, err := aes.NewCipher(boot[addr[1]+elfDataDelta : addr[1]+elfDataDelta+16])
	if err != nil {
		return ks, err
	}
	rec := make([]byte, len(table))
	for i := 0; i < len(table); i += 16 {
		block.Decrypt(rec[i:], table[i:])
	}
	// 16 records of 8 bytes; byte c of every record forms column c.
	col := func(c int) (v [16]byte) {
		for i := range v {
			v[i] = rec[i*8+c]
		}
		return v
	}
	for n := range ks {
		ks[n] = Keys{Salt: col(n), Key: col(2 + n), IV: col(4 + n)}
	}
	return ks, nil
}

// File is an ADVP container held in memory as its original bytes.
type File struct {
	Raw      []byte
	Keys     Keys
	KeyIndex int
	block    cipher.Block
}

var advpMagic = []byte("ADVP")

// Open selects the key set that decrypts the header and checks the MD5.
func Open(raw []byte, ks [2]Keys) (*File, error) {
	if len(raw) < 0x130 {
		return nil, errors.New("maindat: file too short")
	}
	for n, k := range ks {
		f := &File{Raw: raw, Keys: k, KeyIndex: n}
		var err error
		if f.block, err = aes.NewCipher(k.Key[:]); err != nil {
			return nil, err
		}
		if bytes.Equal(f.Decrypt(0x10, 4), advpMagic) {
			if f.Digest() != [16]byte(raw[:16]) {
				return f, errors.New("maindat: MD5 mismatch")
			}
			return f, nil
		}
	}
	return nil, errors.New("maindat: no key decrypts the header")
}

func blocks(n int) int { return (n + 15) &^ 15 }

// Decrypt returns n plaintext bytes of the unit that starts at off. Every
// unit is its own CBC chain with the fixed IV.
func (f *File) Decrypt(off, n int) []byte {
	out := make([]byte, blocks(n))
	cipher.NewCBCDecrypter(f.block, f.Keys.IV[:]).CryptBlocks(out, f.Raw[off:off+blocks(n)])
	return out[:n]
}

// Encrypt writes plain as the unit that starts at off. A length that is not
// a multiple of 16 keeps the decrypted tail of the last block, so an
// unchanged unit re-encrypts to its original bytes.
func (f *File) Encrypt(off int, plain []byte) {
	full := f.Decrypt(off, blocks(len(plain)))
	copy(full, plain)
	cipher.NewCBCEncrypter(f.block, f.Keys.IV[:]).CryptBlocks(f.Raw[off:off+len(full)], full)
}

// Digest is the MD5 over the file with bytes 0-15 replaced by the salt.
func (f *File) Digest() [16]byte {
	h := md5.New()
	h.Write(f.Keys.Salt[:])
	h.Write(f.Raw[16:])
	var d [16]byte
	copy(d[:], h.Sum(nil))
	return d
}

// Sign stores the digest in bytes 0-15.
func (f *File) Sign() {
	d := f.Digest()
	copy(f.Raw, d[:])
}

// Unit is one encrypted range.
type Unit struct {
	Name     string
	Off, Len int
}

// Header holds the parsed fields the layout needs. Offsets are file offsets.
type Header struct {
	Version, Size   uint32
	DiscID          string
	BodyStart       int
	BodyEnd         int
	Extra           [4][2]uint32 // offset, size; 0xFFFFFFFF offset = unused
	ChunkTableCount int
	ChunkTable      int
}

// Units lists every encrypted unit: header, body, chunks, the three extra
// ranges and the music tracks. Plain sound BNDs are not units.
func (f *File) Units() (Header, []Unit, error) {
	le := binary.LittleEndian
	var h Header
	hb := append(make([]byte, 0x10), f.Decrypt(0x10, 0x120)...)
	h.Version, h.Size = le.Uint32(hb[0x14:]), le.Uint32(hb[0x18:])
	h.DiscID = string(bytes.TrimRight(hb[0x34:0x44], "\x00"))
	h.BodyEnd, h.BodyStart = int(le.Uint32(hb[0x58:])), int(le.Uint32(hb[0x5C:]))
	for i := range h.Extra {
		h.Extra[i] = [2]uint32{le.Uint32(hb[0x60+i*8:]), le.Uint32(hb[0x64+i*8:])}
	}
	if int(h.Size) != len(f.Raw) || h.BodyEnd > len(f.Raw) || h.BodyStart >= h.BodyEnd {
		return h, nil, fmt.Errorf("maindat: header sizes do not match file")
	}
	units := []Unit{{"header", 0x10, 0x120}, {"body", h.BodyStart, h.BodyEnd - h.BodyStart}}
	body := append(make([]byte, h.BodyStart), f.Decrypt(h.BodyStart, h.BodyEnd-h.BodyStart)...)
	th := int(le.Uint32(body[0x14C:]))
	if th == 0 {
		return h, units, nil // .adv layout without a block list
	}
	h.ChunkTableCount, h.ChunkTable = int(le.Uint32(body[th:])), int(le.Uint32(body[th+4:]))
	for i := 0; i < h.ChunkTableCount; i++ {
		e := body[h.ChunkTable+i*16:]
		size, id, off := le.Uint32(e), le.Uint32(e[4:]), le.Uint32(e[8:])
		units = append(units, Unit{fmt.Sprintf("chunks/%03d_%05d.bin", i, id), int(off), int(size)})
	}
	for i, name := range []string{"", "thumb.bmp", "skin.skn", "bgm.tbl"} {
		off, n := h.Extra[i][0], h.Extra[i][1]
		if off == 0xFFFFFFFF || name == "" {
			continue
		}
		units = append(units, Unit{name, int(off), int(n)})
	}
	if off := h.Extra[3][0]; off != 0xFFFFFFFF {
		tbl := f.Decrypt(int(off), int(h.Extra[3][1]))
		for i := 0; i < int(le.Uint32(tbl)); i++ {
			e := tbl[0x10+i*16:]
			id, n, o := le.Uint32(e), le.Uint32(e[4:]), le.Uint32(e[8:])
			units = append(units, Unit{fmt.Sprintf("bgm/%d.at3", id), int(o), int(n)})
		}
	}
	return h, units, nil
}

// Plain is the decrypted contents of one unit.
type Plain struct {
	Unit
	Data []byte
}

// Unpack decrypts every unit.
func (f *File) Unpack() (Header, []Plain, error) {
	h, units, err := f.Units()
	if err != nil {
		return h, nil, err
	}
	out := make([]Plain, len(units))
	for i, u := range units {
		out[i] = Plain{u, f.Decrypt(u.Off, u.Len)}
	}
	return h, out, nil
}

// chunkAlign is the alignment of chunks and of the file size.
const chunkAlign = 0x800

// Relocate moves chunk unit i to new space at the end of the file with room
// for n bytes. Its data grows to n bytes (the new bytes are zero), its
// offset changes, and the chunk table entry in the body unit gets the new
// offset (word 2) and size (word 0). Word 3, zero on the disc, gets the old
// offset: the pointers in the chunk keep their values, and a patched loader
// uses word 3 as their base (subpatch.ChunkBase, docs/kfa1/0004). The old bytes
// stay where they were; nothing refers to them. It returns the old offset.
func (f *File) Relocate(units []Plain, i, n int) (int, error) {
	var name string
	var idx int
	if _, err := fmt.Sscanf(units[i].Name, "chunks/%03d_", &idx); err != nil {
		return 0, fmt.Errorf("maindat: %s is not a chunk", units[i].Name)
	}
	name = units[i].Name
	if n < len(units[i].Data) {
		return 0, fmt.Errorf("maindat: %s cannot shrink", name)
	}
	if n > MaxChunk {
		return 0, fmt.Errorf("maindat: %s would be %#x bytes; the loader takes %#x", name, n, MaxChunk)
	}
	body := -1
	for k := range units {
		if units[k].Name == "body" {
			body = k
		}
	}
	if body < 0 {
		return 0, errors.New("maindat: no body unit")
	}
	le := binary.LittleEndian
	b := units[body].Data
	at := func(off int) int { return off - units[body].Off }
	th := int(le.Uint32(b[at(0x14C):]))
	table := int(le.Uint32(b[at(th+4):]))
	e := at(table + idx*16)
	old := units[i].Off
	if int(le.Uint32(b[e+8:])) != old {
		return 0, fmt.Errorf("maindat: chunk table entry %d does not match %s", idx, name)
	}
	newOff := (len(f.Raw) + chunkAlign - 1) / chunkAlign * chunkAlign
	end := (newOff + n + chunkAlign - 1) / chunkAlign * chunkAlign
	f.Raw = append(f.Raw, make([]byte, end-len(f.Raw))...)
	le.PutUint32(b[e:], uint32(n))
	le.PutUint32(b[e+8:], uint32(newOff))
	if le.Uint32(b[e+12:]) == 0 {
		le.PutUint32(b[e+12:], uint32(old))
	}
	units[i].Data = append(units[i].Data, make([]byte, n-len(units[i].Data))...)
	units[i].Off, units[i].Len = newOff, n
	return old, nil
}

// MaxChunk is the largest chunk the loader reads into its buffer.
const MaxChunk = 0x500000

// Pack encrypts units back and re-signs the file. A unit keeps its size
// unless Relocate moved it. The header gets the file size.
func (f *File) Pack(units []Plain) error {
	for k := range units {
		if units[k].Name == "header" && len(units[k].Data) >= 0x18-0x10+4 {
			binary.LittleEndian.PutUint32(units[k].Data[0x18-0x10:], uint32(len(f.Raw)))
		}
	}
	for _, p := range units {
		if len(p.Data) != p.Len {
			return fmt.Errorf("maindat: %s is %d bytes, want %d", p.Name, len(p.Data), p.Len)
		}
		f.Encrypt(p.Off, p.Data)
	}
	f.Sign()
	return nil
}
