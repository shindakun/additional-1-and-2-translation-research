package bnd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTripSynthetic(t *testing.T) {
	for _, f := range []*File{
		{Version: VersionNamed, Flag: 0xD3, Entries: []Entry{
			{ID: 0, Name: "a.gim", Data: []byte{1, 2, 3}},
			{ID: 1, Name: "b.fmg", Data: bytes.Repeat([]byte{9}, 0x900)},
		}},
		{Version: VersionUnnamed, Entries: []Entry{{ID: 5000, Data: []byte("HVG ")}}},
		{Version: VersionNamed, Flag: 0xD3},
	} {
		b := f.Bytes()
		g, err := Read(b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(g.Bytes(), b) {
			t.Fatalf("round trip differs for version %#x", f.Version)
		}
	}
}

// TestRoundTripDiscs re-encodes every BND under KFA_DATA, nested ones
// included, and compares bytes. KFA_DATA is the research directory written
// by scripts/extract_iso.sh and scripts/maindat.py.
func TestRoundTripDiscs(t *testing.T) {
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	n := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".bnd") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n += check(t, p, b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no BND files found under KFA_DATA")
	}
	t.Logf("%d BND files round-trip", n)
}

func check(t *testing.T, name string, b []byte) int {
	t.Helper()
	f, err := Read(b)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return 0
	}
	if !bytes.Equal(f.Bytes(), b) {
		t.Errorf("%s: re-encoded bytes differ", name)
		return 0
	}
	n := 1
	for _, e := range f.Entries {
		if bytes.HasPrefix(e.Data, magic) {
			n += check(t, name+"/"+e.Name, e.Data)
		}
	}
	return n
}
