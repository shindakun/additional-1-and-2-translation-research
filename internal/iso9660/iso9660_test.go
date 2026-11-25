package iso9660

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestDiscs walks each image under KFA_DATA/iso and compares every file with
// the tree that 7z extracted to KFA_DATA/extract.
func TestDiscs(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	for _, g := range []string{"kfaI", "kfaII"} {
		t.Run(g, func(t *testing.T) {
			fh, err := os.Open(filepath.Join(root, "iso", g+".iso"))
			if err != nil {
				t.Skip(err)
			}
			defer func() { _ = fh.Close() }() // read-only handle
			img, err := Open(fh)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range img.Files {
				got, err := img.Read(f)
				if err != nil {
					t.Fatal(err)
				}
				want, err := os.ReadFile(filepath.Join(root, "extract", g, filepath.FromSlash(f.Path)))
				if err != nil {
					t.Fatalf("%s: %v", f.Path, err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s differs from extracted copy", f.Path)
				}
			}
			t.Logf("%s: %d files match", g, len(img.Files))
		})
	}
}

type memImage struct{ b []byte }

func (m *memImage) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(m.b)) {
		return 0, io.EOF
	}
	return copy(p, m.b[off:]), nil
}
func (m *memImage) WriteAt(p []byte, off int64) (int, error) {
	if end := int(off) + len(p); end > len(m.b) {
		m.b = append(m.b, make([]byte, end-len(m.b))...)
	}
	return copy(m.b[off:], p), nil
}

// TestReplace patches a file in a copy of an image and reads it back.
func TestReplace(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	b, err := os.ReadFile(filepath.Join(root, "iso", "kfaI.iso"))
	if err != nil {
		t.Skip(err)
	}
	m := &memImage{b}
	img, err := Open(m)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := img.Find("PSP_GAME/PARAM.SFO")
	if !ok {
		t.Fatal("PARAM.SFO not found")
	}
	data := bytes.Repeat([]byte{0xAB}, int(f.Size)-8)
	if err := Replace(m, f, data); err != nil {
		t.Fatal(err)
	}
	img2, err := Open(m)
	if err != nil {
		t.Fatal(err)
	}
	f2, _ := img2.Find("PSP_GAME/PARAM.SFO")
	got, _ := img2.Read(f2)
	if !bytes.Equal(got, data) {
		t.Fatal("replaced file does not read back")
	}
	if err := Replace(m, f, make([]byte, int(f.Sectors())*SectorSize+1)); err == nil {
		t.Fatal("oversized replacement accepted")
	}
}

// TestGrow makes main.dat larger than its sectors, by a little and by more
// than the rest of the image: the files after it move to the end and every
// file still reads back.
func TestGrow(t *testing.T) {
	for _, extra := range []int{300000, 200 << 20} {
		t.Run(fmt.Sprint(extra), func(t *testing.T) { testGrow(t, extra) })
	}
}

func testGrow(t *testing.T, extra int) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	b, err := os.ReadFile(filepath.Join(root, "iso", "kfaI.iso"))
	if err != nil {
		t.Skip(err)
	}
	m := &memImage{b}
	img, err := Open(m)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string][]byte{}
	extent := map[string]uint32{}
	for _, f := range img.Files {
		extent[f.Path] = f.Extent
		if before[f.Path], err = img.Read(f); err != nil {
			t.Fatal(err)
		}
	}
	const p = "PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat"
	f, _ := img.Find(p)
	data := append(bytes.Clone(before[p]), bytes.Repeat([]byte{0x5A}, extra)...)
	if err := Grow(m, img, f, data); err != nil {
		t.Fatal(err)
	}
	img2, err := Open(m)
	if err != nil {
		t.Fatal(err)
	}
	moved := 0
	for _, f2 := range img2.Files {
		got, err := img2.Read(f2)
		if err != nil {
			t.Fatal(err)
		}
		want := before[f2.Path]
		if f2.Path == p {
			want = data
		}
		if !bytes.Equal(got, want) {
			i := 0
			for i < len(got) && i < len(want) && got[i] == want[i] {
				i++
			}
			t.Fatalf("%s does not read back: len %d/%d, first diff %d", f2.Path, len(got), len(want), i)
		}
		if extent[f2.Path] != f2.Extent && f2.Path != p {
			moved++
		}
	}
	vol, _ := volumeSectors(m)
	if moved == 0 || vol*SectorSize != int64(len(m.b)) {
		t.Fatalf("moved %d files; volume %d sectors, image %d bytes", moved, vol, len(m.b))
	}
	// No two files share a sector.
	for i := 1; i < len(img2.Files); i++ {
		a, c := img2.Files[i-1], img2.Files[i]
		if int64(a.Extent)+a.Sectors() > int64(c.Extent) && a.Size > 0 {
			t.Fatalf("%s overlaps %s", a.Path, c.Path)
		}
	}
}
