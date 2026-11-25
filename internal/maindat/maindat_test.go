package maindat

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// games lists the discs to test under KFA_DATA (the research directory).
var games = []struct{ dir, file string }{
	{"kfaI", "GameData/01_UmdTitle/main.dat"},
	{"kfaII", "GameData/01_UmdTitle/main.dat"},
}

func load(t *testing.T, dir, file string) (*File, []byte) {
	t.Helper()
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	base := filepath.Join(root, "extract", dir, "PSP_GAME")
	boot, err := os.ReadFile(filepath.Join(base, "SYSDIR", "BOOT.BIN"))
	if err != nil {
		t.Skip(err)
	}
	raw, err := os.ReadFile(filepath.Join(base, "USRDIR", file))
	if err != nil {
		t.Fatal(err)
	}
	ks, err := KeysFromBoot(boot)
	if err != nil {
		t.Fatal(err)
	}
	orig := bytes.Clone(raw)
	f, err := Open(raw, ks)
	if err != nil {
		t.Fatal(err)
	}
	return f, orig
}

// TestReencrypt decrypts and re-encrypts every unit, re-signs, and compares
// the whole file with the original.
func TestReencrypt(t *testing.T) {
	for _, g := range games {
		t.Run(g.dir, func(t *testing.T) {
			f, orig := load(t, g.dir, g.file)
			h, units, err := f.Units()
			if err != nil {
				t.Fatal(err)
			}
			for _, u := range units {
				if u.Name != "header" && u.Name != "body" && len(u.Name) > 7 && u.Name[:7] == "chunks/" {
					p := f.Decrypt(u.Off, 4)
					if binary.LittleEndian.Uint32(p) != uint32(u.Len) {
						t.Fatalf("%s: first word is not the size", u.Name)
					}
				}
				f.Encrypt(u.Off, f.Decrypt(u.Off, u.Len))
			}
			f.Sign()
			if !bytes.Equal(f.Raw, orig) {
				t.Fatal("re-encrypted file differs")
			}
			t.Logf("%s: %d units, %d chunks, disc %s", g.dir, len(units), h.ChunkTableCount, h.DiscID)
		})
	}
}

// TestEditDetected changes one byte of a chunk and checks the digest follows.
func TestEditDetected(t *testing.T) {
	f, _ := load(t, games[0].dir, games[0].file)
	_, units, err := f.Units()
	if err != nil {
		t.Fatal(err)
	}
	u := units[2]
	p := f.Decrypt(u.Off, u.Len)
	p[0x20] ^= 0xFF
	f.Encrypt(u.Off, p)
	if f.Digest() == [16]byte(f.Raw[:16]) {
		t.Fatal("digest did not change")
	}
	f.Sign()
	if f.Digest() != [16]byte(f.Raw[:16]) {
		t.Fatal("sign did not store the digest")
	}
	if !bytes.Equal(f.Decrypt(u.Off, u.Len), p) {
		t.Fatal("edited unit does not decrypt to the edit")
	}
}

// TestUnpackPack unpacks every unit, packs them back unchanged and compares.
func TestUnpackPack(t *testing.T) {
	f, orig := load(t, games[0].dir, games[0].file)
	_, units, err := f.Unpack()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Pack(units); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Raw, orig) {
		t.Fatal("pack of unchanged units differs")
	}
	units[0].Data = units[0].Data[:10]
	if err := f.Pack(units); err == nil {
		t.Fatal("resized unit accepted")
	}
}
