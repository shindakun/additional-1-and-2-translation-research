package relayout

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/sjis"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/text"
)

func load(t *testing.T, game string) (*maindat.File, []maindat.Plain, []text.Slot) {
	t.Helper()
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	base := filepath.Join(root, "extract", game, "PSP_GAME")
	read := func(p ...string) []byte {
		b, err := os.ReadFile(filepath.Join(append([]string{base}, p...)...))
		if err != nil {
			t.Skip(err)
		}
		return b
	}
	sys, err := bnd.Read(read("USRDIR", "Bnd", "AdvSys.bnd"))
	if err != nil {
		t.Fatal(err)
	}
	var j, u []byte
	for _, e := range sys.Entries {
		switch e.Name {
		case "jis2ucs.bin":
			j = e.Data
		case "ucs2jis.bin":
			u = e.Data
		}
	}
	c, err := sjis.New(j, u)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := maindat.KeysFromBoot(read("SYSDIR", "BOOT.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := maindat.Open(read("USRDIR", "GameData", "01_UmdTitle", "main.dat"), ks)
	if err != nil {
		t.Fatal(err)
	}
	_, units, err := f.Unpack()
	if err != nil {
		t.Fatal(err)
	}
	slots, err := text.Extract(units, c)
	if err != nil {
		t.Fatal(err)
	}
	return f, units, slots
}

// TestMove moves a chunk and reads it back from the packed file: the data
// is at the new offset, the table entry holds the new offset, the size and
// the old offset as base, and the file signs and verifies.
func TestMove(t *testing.T) {
	f, units, _ := load(t, "kfaI")
	u := -1
	for i := range units {
		if strings.HasPrefix(units[i].Name, "chunks/067_") {
			u = i
		}
	}
	old, oldLen := units[u].Off, len(units[u].Data)
	want := append(append([]byte(nil), units[u].Data...), "added"...)
	if err := Move(f, units, u, len(want)); err != nil {
		t.Fatal(err)
	}
	copy(units[u].Data[oldLen:], "added")
	if err := f.Pack(units); err != nil {
		t.Fatal(err)
	}
	ks := [2]maindat.Keys{f.Keys, f.Keys}
	g, err := maindat.Open(f.Raw, ks)
	if err != nil {
		t.Fatal(err)
	}
	_, back, err := g.Unpack()
	if err != nil {
		t.Fatal(err)
	}
	if back[u].Off == old || back[u].Off%0x800 != 0 || string(back[u].Data) != string(want) {
		t.Fatalf("moved chunk does not read back: off %#x len %d", back[u].Off, len(back[u].Data))
	}
	var body []byte
	var bodyOff int
	for _, p := range back {
		if p.Name == "body" {
			body, bodyOff = p.Data, p.Off
		}
	}
	le := binary.LittleEndian
	th := int(le.Uint32(body[0x14C-bodyOff:]))
	table := int(le.Uint32(body[th+4-bodyOff:]))
	idx := 67
	e := table + idx*16 - bodyOff
	if got := le.Uint32(body[e+12:]); got != uint32(old) {
		t.Fatalf("entry word 3 is %#x, want old offset %#x", got, old)
	}
}
