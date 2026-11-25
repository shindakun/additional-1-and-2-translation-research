package iso9660

import (
	"bytes"
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

func (m *memImage) ReadAt(p []byte, off int64) (int, error)  { return copy(p, m.b[off:]), nil }
func (m *memImage) WriteAt(p []byte, off int64) (int, error) { return copy(m.b[off:], p), nil }

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
