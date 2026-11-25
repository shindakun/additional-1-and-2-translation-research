package text

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/sjis"
)

func setup(t *testing.T) ([]maindat.Plain, *sjis.Codec) {
	t.Helper()
	root := os.Getenv("KFA_DATA")
	if root == "" {
		t.Skip("KFA_DATA not set")
	}
	base := filepath.Join(root, "extract", "kfaI", "PSP_GAME")
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
		if e.Name == "jis2ucs.bin" {
			j = e.Data
		}
		if e.Name == "ucs2jis.bin" {
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
	return units, c
}

func TestExtractApply(t *testing.T) {
	units, c := setup(t)
	slots, err := Extract(units, c)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, s := range slots {
		kinds[s.Kind]++
	}
	entries := Entries(slots)
	t.Logf("slots %v, unique entries %d", kinds, len(entries))
	if kinds[KindText] < 2000 || kinds[KindName] < 1000 {
		t.Fatalf("too few slots: %v", kinds)
	}

	orig := make([][]byte, len(units))
	for i := range units {
		orig[i] = bytes.Clone(units[i].Data)
	}
	if r, probs := Apply(units, slots, entries, c, nil); r.Written != 0 || len(probs) != 0 {
		t.Fatalf("empty translations wrote %d slots, %d problems", r.Written, len(probs))
	}
	for i := range units {
		if !bytes.Equal(units[i].Data, orig[i]) {
			t.Fatalf("unit %s changed with no translations", units[i].Name)
		}
	}

	var target *Entry
	for i := range entries {
		if entries[i].JA == "特に変わったところはない" {
			target = &entries[i]
		}
	}
	if target == nil {
		t.Fatal("known global text not found")
	}
	long := Entry{JA: target.JA, Kind: target.Kind, EN: strings.Repeat("x", target.Max+1)}
	if r, probs := Apply(units, slots, []Entry{long}, c, nil); r.Written != 0 || len(probs) != target.Count {
		t.Fatalf("overlong translation: wrote %d, %d problems", r.Written, len(probs))
	}
	target.EN = "Nothing unusual."
	r, probs := Apply(units, slots, []Entry{*target}, c, nil)
	if r.Written != target.Count || len(probs) != 0 {
		t.Fatalf("wrote %d of %d, problems %v", r.Written, target.Count, probs)
	}
	again, err := Extract(units, c)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range again {
		if s.JA == "Nothing unusual." {
			found = true
		}
	}
	if !found {
		t.Fatal("translation not read back")
	}
}

// TestMove checks that a text string longer than its slot goes to space from
// the allocator and that extraction finds it there through its pointers.
func TestMove(t *testing.T) {
	units, c := setup(t)
	slots, err := Extract(units, c)
	if err != nil {
		t.Fatal(err)
	}
	var s *Slot
	for i := range slots {
		if slots[i].Kind == KindText && strings.HasPrefix(slots[i].Where, "c000/") && len(slots[i].Ptrs) > 0 {
			s = &slots[i]
			break
		}
	}
	if s == nil {
		t.Fatal("no chunk 0 text slot")
	}
	long := strings.Repeat("y", s.Max+20)
	// Free space: the last 64 bytes of the chunk's first image.
	data := units[s.Unit].Data
	free := len(data) - 64
	for i := free; i < len(data); i++ {
		data[i] = 0
	}
	sp := &testSpace{unit: s.Unit, off: free, n: 64}
	r, probs := Apply(units, slots, []Entry{{JA: s.JA, Kind: KindText, EN: long}}, c, sp)
	if r.Moved == 0 || sp.calls == 0 || sp.freed != s.Max+1 {
		t.Fatalf("not moved: %+v %v", r, probs)
	}
	again, err := Extract(units, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range again {
		if a.Unit == s.Unit && a.Off == free {
			if a.JA != long {
				t.Fatalf("read back %q", a.JA)
			}
			return
		}
	}
	t.Fatal("moved string not found by extraction")
}

type testSpace struct {
	unit, off, n int
	calls, freed int
}

func (t *testSpace) Alloc(unit int, b []byte) (int, bool) {
	t.calls++
	if unit != t.unit || len(b) > t.n {
		return 0, false
	}
	t.n = 0
	return t.off, true
}

func (t *testSpace) Free(_, _, n int) { t.freed += n }
