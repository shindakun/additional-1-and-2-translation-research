// Package iso9660 reads the ISO 9660 subset used by UMD images and replaces
// files in place. See docs/0002-tool-planning.md.
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
