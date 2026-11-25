// Package iso9660 reads the ISO 9660 subset used by UMD images and replaces
// files in place. See docs/kfa1/0002-tool-planning.md.
package iso9660

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// SectorSize is the logical block size of a UMD image.
const SectorSize = 2048

// File is one regular file in the image.
type File struct {
	Path   string // slash-separated, no leading slash, no ";1"
	Extent uint32 // first sector
	Size   uint32
	// Record is the byte offset of the directory record, used to update the
	// size when a file is replaced.
	Record int64
}

// Sectors returns the number of sectors the file occupies.
func (f File) Sectors() int64 { return (int64(f.Size) + SectorSize - 1) / SectorSize }

// Image is an opened UMD image.
type Image struct {
	r     io.ReaderAt
	Files []File
}

// Open reads the primary volume descriptor and walks every directory.
func Open(r io.ReaderAt) (*Image, error) {
	pvd := make([]byte, SectorSize)
	if _, err := r.ReadAt(pvd, 16*SectorSize); err != nil {
		return nil, err
	}
	if pvd[0] != 1 || string(pvd[1:6]) != "CD001" {
		return nil, errors.New("iso9660: no primary volume descriptor")
	}
	img := &Image{r: r}
	root := pvd[156 : 156+34]
	if err := img.walk("", binary.LittleEndian.Uint32(root[2:]), binary.LittleEndian.Uint32(root[10:])); err != nil {
		return nil, err
	}
	sort.Slice(img.Files, func(i, j int) bool { return img.Files[i].Extent < img.Files[j].Extent })
	return img, nil
}

func (img *Image) walk(dir string, extent, size uint32) error {
	buf := make([]byte, size)
	base := int64(extent) * SectorSize
	if _, err := img.r.ReadAt(buf, base); err != nil {
		return err
	}
	for off := 0; off < len(buf); {
		n := int(buf[off])
		if n == 0 { // records do not cross sectors; skip to the next one
			off = (off/SectorSize + 1) * SectorSize
			continue
		}
		rec := buf[off : off+n]
		nameLen := int(rec[32])
		name := string(rec[33 : 33+nameLen])
		ext, sz, flags := binary.LittleEndian.Uint32(rec[2:]), binary.LittleEndian.Uint32(rec[10:]), rec[25]
		if name != "\x00" && name != "\x01" {
			name = strings.TrimSuffix(name, ";1")
			p := path.Join(dir, name)
			if flags&2 != 0 {
				if err := img.walk(p, ext, sz); err != nil {
					return err
				}
			} else {
				img.Files = append(img.Files, File{Path: p, Extent: ext, Size: sz, Record: base + int64(off)})
			}
		}
		off += n
	}
	return nil
}

// Find returns the file with the given path, compared case-insensitively.
func (img *Image) Find(p string) (File, bool) {
	for _, f := range img.Files {
		if strings.EqualFold(f.Path, p) {
			return f, true
		}
	}
	return File{}, false
}

// Read returns a file's contents.
func (img *Image) Read(f File) ([]byte, error) {
	b := make([]byte, f.Size)
	_, err := img.r.ReadAt(b, int64(f.Extent)*SectorSize)
	return b, err
}

// Replace writes data over a file in w, which must be the same image. The new
// data must fit in the sectors the file already occupies. The directory
// record size is updated and the rest of the last sector is zeroed.
func Replace(w io.WriterAt, f File, data []byte) error {
	if (int64(len(data))+SectorSize-1)/SectorSize > f.Sectors() {
		return fmt.Errorf("iso9660: %s: %d bytes do not fit in %d sectors", f.Path, len(data), f.Sectors())
	}
	end := (int64(len(data)) + SectorSize - 1) / SectorSize * SectorSize
	padded := make([]byte, end)
	copy(padded, data)
	if _, err := w.WriteAt(padded, int64(f.Extent)*SectorSize); err != nil {
		return err
	}
	var both [8]byte
	binary.LittleEndian.PutUint32(both[:4], uint32(len(data)))
	binary.BigEndian.PutUint32(both[4:], uint32(len(data)))
	_, err := w.WriteAt(both[:], f.Record+10)
	return err
}

// ReadWriterAt reads and writes an image in place.
type ReadWriterAt interface {
	io.ReaderAt
	io.WriterAt
}

func putBoth(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b[:4], v)
	binary.BigEndian.PutUint32(b[4:], v)
}

// volumeSectors returns the volume space size from the primary volume
// descriptor.
func volumeSectors(r io.ReaderAt) (int64, error) {
	var b [4]byte
	if _, err := r.ReadAt(b[:], 16*SectorSize+80); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint32(b[:])), nil
}

// Grow writes data over file f in rw like Replace, but data can be larger
// than the sectors f holds. Every file that lies in the sectors f needs
// moves to the end of the image: its data is copied there and its directory
// record points to the new place. The volume space size in the primary
// volume descriptor grows to match. Files are found by path, so moving them
// does not change what a program reads.
func Grow(rw ReadWriterAt, img *Image, f File, data []byte) error {
	need := (int64(len(data)) + SectorSize - 1) / SectorSize
	if need <= f.Sectors() {
		return Replace(rw, f, data)
	}
	end, err := volumeSectors(rw)
	if err != nil {
		return err
	}
	for _, g := range img.Files {
		if g.Extent+uint32(g.Sectors()) > uint32(end) {
			end = int64(g.Extent) + g.Sectors()
		}
	}
	lo, hi := int64(f.Extent)+f.Sectors(), int64(f.Extent)+need
	// Moved files go after the grown file, even when it reaches past the
	// old end of the volume.
	end = max(end, hi)
	for i := range img.Files {
		g := &img.Files[i]
		if g.Path == f.Path || int64(g.Extent) >= hi || int64(g.Extent)+g.Sectors() <= lo {
			continue
		}
		b, err := img.Read(*g)
		if err != nil {
			return err
		}
		padded := make([]byte, g.Sectors()*SectorSize)
		copy(padded, b)
		if _, err := rw.WriteAt(padded, end*SectorSize); err != nil {
			return err
		}
		var rec [8]byte
		putBoth(rec[:], uint32(end))
		if _, err := rw.WriteAt(rec[:], g.Record+2); err != nil {
			return err
		}
		g.Extent = uint32(end)
		end += g.Sectors()
	}
	var vol [8]byte
	putBoth(vol[:], uint32(end))
	if _, err := rw.WriteAt(vol[:], 16*SectorSize+80); err != nil {
		return err
	}
	// Extend the image to the new volume size; keep the last byte.
	last := make([]byte, 1)
	_, _ = rw.ReadAt(last, end*SectorSize-1) // past the end reads as zero
	if _, err := rw.WriteAt(last, end*SectorSize-1); err != nil {
		return err
	}
	g := f
	g.Size = uint32(need * SectorSize)
	return Replace(rw, g, data)
}
