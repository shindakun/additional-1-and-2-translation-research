// Command kfa extracts, inspects and patches King's Field Additional I and II
// disc data. One binary, several subcommands; see docs/0002-tool-planning.md.
package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/fmg"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/font"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/gim"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/images"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/iso9660"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/sjis"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/subpatch"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/text"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "kfa:", err)
		os.Exit(1)
	}
}

var errUsage = errors.New("bad arguments")

func run() error {
	flag.Usage = usage
	flag.Parse()

	// Ctrl-C cancels the whole tree, so long-running work is interruptible.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_ = ctx

	args := flag.Args()
	if len(args) == 0 {
		usage()
		return fmt.Errorf("no command given")
	}
	cmd := args[0]
	if len(args) > 1 {
		cmd += " " + args[1]
	}
	var err error
	switch cmd {
	case "iso ls":
		err = isoList(args[2:])
	case "iso extract":
		err = isoExtract(args[2:])
	case "iso put":
		err = isoPut(args[2:])
	case "bnd ls":
		err = bndList(args[2:])
	case "bnd unpack":
		err = bndUnpack(args[2:])
	case "dat verify":
		err = datVerify(args[2:])
	case "dat unpack":
		err = datUnpack(args[2:])
	case "dat pack":
		err = datPack(args[2:])
	case "font vwf":
		err = fontVWF(args[2:])
	case "text export":
		err = textExport(args[2:])
	case "text import":
		err = textImport(args[2:])
	case "fmg export":
		err = fmgExport(args[2:])
	case "fmg import":
		err = fmgImport(args[2:])
	case "img export":
		err = imgExport(args[2:])
	case "img import":
		err = imgImport(args[2:])
	case "gim topng":
		err = gimToPNG(args[2:])
	case "gim scan":
		err = gimScan(args[2:])
	case "patch counter":
		err = patchCounter(args[2:])
	default:
		if args[0] == "version" {
			fmt.Println("kfa", version)
			return nil
		}
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
	if errors.Is(err, errUsage) {
		usage()
	}
	return err
}

func usage() {
	fmt.Fprint(os.Stderr, `kfa: King's Field Additional disc tools

Usage:
  kfa <command> [args]

Commands:
  iso ls IMAGE                     List files with sector and size
  iso extract IMAGE DIR            Write every file under DIR
  iso put IMAGE PATH FILE          Replace PATH in IMAGE in place (must fit)
  bnd ls FILE.bnd                  List entries, nested BNDs included
  bnd unpack FILE.bnd DIR          Write top-level entries to DIR
  dat verify BOOT.BIN main.dat     Check keys, MD5 and every unit
  dat unpack BOOT.BIN main.dat DIR Write every unit decrypted to DIR
  dat pack BOOT.BIN main.dat DIR OUT
                                   Encrypt units from DIR into a copy of
                                   main.dat (same sizes) and re-sign it
  font vwf IN.bnd OUT.bnd [GAP] [SPACE]
                                   Proportional ASCII metrics in every font
                                   inside IN (default gap 1, space 4)
  text export GAMEDIR OUT.json     Write every translatable string of main.dat
  text import GAMEDIR IN.json OUT.dat [BASE.dat]
                                   Write translations into a copy of main.dat,
                                   or of BASE.dat (for example the output of
                                   img import). A text string that does not
                                   fit moves to space freed by re-encoding a
                                   JPEG of its chunk. Reports the rest.
                                   (GAMEDIR is an extracted PSP_GAME folder)
  fmg export GAMEDIR OUT.json      Write the FMG messages in AdvSys.bnd
  fmg import GAMEDIR IN.json OUT.bnd
                                   Write translations into the FMG messages
                                   and build a new AdvSys.bnd
  img export GAMEDIR DIR           Write every unique image of every main.dat
                                   chunk and the header GIMs (JPEG, BMP, GIM
                                   and a PNG of each GIM) and DIR/index.tsv
  img import GAMEDIR DIR OUT.dat [BASE.dat]
                                   Write the images in DIR (names from img
                                   export; a GIM can also be NAME.png, which
                                   is encoded into the original GIM format)
                                   into main.dat, or into BASE.dat,
                                   in place; each must fit the original size.
                                   BASE.dat must hold the original images, so
                                   run img import before text import
  gim topng IN.gim OUT.png         Convert a GIM image to PNG
  gim scan FILE DIR                Write every GIM found inside FILE as
                                   DIR/<offset>.png
  patch counter BOOT.BIN OUT       Draw a frame counter over movies (test build)
  version                          Print the version and exit
`)
}

func openImage(path string) (*iso9660.Image, *os.File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	img, err := iso9660.Open(fh)
	if err != nil {
		_ = fh.Close() // read-only handle
		return nil, nil, err
	}
	return img, fh, nil
}

func isoList(a []string) error {
	if len(a) != 1 {
		return errUsage
	}
	img, fh, err := openImage(a[0])
	if err != nil {
		return err
	}
	defer func() { _ = fh.Close() }() // read-only handle
	for _, f := range img.Files {
		fmt.Printf("%8d %10d %s\n", f.Extent, f.Size, f.Path)
	}
	return nil
}

func isoExtract(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	img, fh, err := openImage(a[0])
	if err != nil {
		return err
	}
	defer func() { _ = fh.Close() }() // read-only handle
	for _, f := range img.Files {
		b, err := img.Read(f)
		if err != nil {
			return err
		}
		out := filepath.Join(a[1], filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d files to %s\n", len(img.Files), a[1])
	return nil
}

func isoPut(a []string) error {
	if len(a) != 3 {
		return errUsage
	}
	data, err := os.ReadFile(a[2])
	if err != nil {
		return err
	}
	fh, err := os.OpenFile(a[0], os.O_RDWR, 0)
	if err != nil {
		return err
	}
	img, err := iso9660.Open(fh)
	if err != nil {
		_ = fh.Close() // error path; the open error is reported
		return err
	}
	f, ok := img.Find(a[1])
	if !ok {
		_ = fh.Close() // error path; the lookup error is reported
		return fmt.Errorf("%s not in image", a[1])
	}
	if err := iso9660.Replace(fh, f, data); err != nil {
		_ = fh.Close() // error path; the write error is reported
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	fmt.Printf("replaced %s (%d -> %d bytes)\n", f.Path, f.Size, len(data))
	return nil
}

func readBND(path string) (*bnd.File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bnd.Read(b)
}

func bndList(a []string) error {
	if len(a) != 1 {
		return errUsage
	}
	f, err := readBND(a[0])
	if err != nil {
		return err
	}
	var walk func(f *bnd.File, indent string)
	walk = func(f *bnd.File, indent string) {
		fmt.Printf("%sversion %#x, %d entries\n", indent, f.Version, len(f.Entries))
		for _, e := range f.Entries {
			fmt.Printf("%s  %5d %8d %s\n", indent, e.ID, len(e.Data), e.Name)
			if sub, err := bnd.Read(e.Data); err == nil {
				walk(sub, indent+"    ")
			}
		}
	}
	walk(f, "")
	return nil
}

func bndUnpack(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	f, err := readBND(a[0])
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a[1], 0o755); err != nil {
		return err
	}
	for _, e := range f.Entries {
		name := e.Name
		if name == "" {
			name = fmt.Sprintf("%05d.bin", e.ID)
		}
		if err := os.WriteFile(filepath.Join(a[1], name), e.Data, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d entries to %s\n", len(f.Entries), a[1])
	return nil
}

func datVerify(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	boot, err := os.ReadFile(a[0])
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(a[1])
	if err != nil {
		return err
	}
	ks, err := maindat.KeysFromBoot(boot)
	if err != nil {
		return err
	}
	orig := bytes.Clone(raw)
	f, err := maindat.Open(raw, ks)
	if err != nil {
		return err
	}
	h, units, err := f.Units()
	if err != nil {
		return err
	}
	for _, u := range units {
		f.Encrypt(u.Off, f.Decrypt(u.Off, u.Len))
	}
	f.Sign()
	if !bytes.Equal(f.Raw, orig) {
		return errors.New("re-encryption does not reproduce the file")
	}
	fmt.Printf("disc %s, version %#x, key index %d, MD5 ok, %d units re-encrypt exactly\n",
		h.DiscID, h.Version, f.KeyIndex, len(units))
	return nil
}

func fontVWF(a []string) error {
	if len(a) < 2 || len(a) > 4 {
		return errUsage
	}
	gap, space := int16(1), int16(4)
	if len(a) > 2 {
		v, err := strconv.Atoi(a[2])
		if err != nil {
			return errUsage
		}
		gap = int16(v)
	}
	if len(a) > 3 {
		v, err := strconv.Atoi(a[3])
		if err != nil {
			return errUsage
		}
		space = int16(v)
	}
	f, err := readBND(a[0])
	if err != nil {
		return err
	}
	n, err := proportional(f, gap, space)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%s holds no font", a[0])
	}
	if err := os.WriteFile(a[1], f.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s: %d fonts changed\n", a[1], n)
	return nil
}

// proportional rewrites the CCM of every font BND inside f, recursing into
// nested BNDs, and returns the number of fonts changed.
func proportional(f *bnd.File, gap, space int16) (int, error) {
	get := func(name string) []byte {
		for _, e := range f.Entries {
			if e.Name == name {
				return e.Data
			}
		}
		return nil
	}
	n := 0
	if ccmRaw, t0, t1, plt := get("Ccm.ccm"), get("Tex00.gim"), get("Tex01.gim"), get("Plt00.plt"); ccmRaw != nil && t0 != nil && t1 != nil && plt != nil {
		c, err := font.ReadCCM(ccmRaw)
		if err != nil {
			return 0, err
		}
		var tex [2]*font.Texture
		for i, g := range [][]byte{t0, t1} {
			if tex[i], err = font.ReadTexture(g); err != nil {
				return 0, err
			}
		}
		c.Proportional(tex, plt, gap, space)
		for i := range f.Entries {
			if f.Entries[i].Name == "Ccm.ccm" {
				f.Entries[i].Data = c.Bytes()
			}
		}
		n++
	}
	for i := range f.Entries {
		sub, err := bnd.Read(f.Entries[i].Data)
		if err != nil {
			continue
		}
		m, err := proportional(sub, gap, space)
		if err != nil {
			return 0, err
		}
		if m > 0 {
			f.Entries[i].Data = sub.Bytes()
			n += m
		}
	}
	return n, nil
}

func patchCounter(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	boot, err := os.ReadFile(a[0])
	if err != nil {
		return err
	}
	out, err := subpatch.Counter(boot, 16, 16, false)
	if err != nil {
		return err
	}
	if err := os.WriteFile(a[1], out, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", a[1])
	return nil
}

func openDat(bootPath, datPath string) (*maindat.File, error) {
	boot, err := os.ReadFile(bootPath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(datPath)
	if err != nil {
		return nil, err
	}
	ks, err := maindat.KeysFromBoot(boot)
	if err != nil {
		return nil, err
	}
	return maindat.Open(raw, ks)
}

func datUnpack(a []string) error {
	if len(a) != 3 {
		return errUsage
	}
	f, err := openDat(a[0], a[1])
	if err != nil {
		return err
	}
	_, units, err := f.Unpack()
	if err != nil {
		return err
	}
	for _, u := range units {
		out := filepath.Join(a[2], filepath.FromSlash(u.Name))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, u.Data, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %d units to %s\n", len(units), a[2])
	return nil
}

func datPack(a []string) error {
	if len(a) != 4 {
		return errUsage
	}
	f, err := openDat(a[0], a[1])
	if err != nil {
		return err
	}
	_, units, err := f.Unpack()
	if err != nil {
		return err
	}
	for i := range units {
		b, err := os.ReadFile(filepath.Join(a[2], filepath.FromSlash(units[i].Name)))
		if err != nil {
			return err
		}
		units[i].Data = b
	}
	if err := f.Pack(units); err != nil {
		return err
	}
	if err := os.WriteFile(a[3], f.Raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d units)\n", a[3], len(units))
	return nil
}

// gameText opens main.dat, its units, the codec and the text slots of an
// extracted PSP_GAME folder.
func gameText(dir string) (*maindat.File, []maindat.Plain, *sjis.Codec, []text.Slot, error) {
	f, err := openDat(filepath.Join(dir, "SYSDIR", "BOOT.BIN"), filepath.Join(dir, "USRDIR", "GameData", "01_UmdTitle", "main.dat"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	_, units, err := f.Unpack()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	_, c, err := sysBND(dir)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	slots, err := text.Extract(units, c)
	return f, units, c, slots, err
}

func textExport(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	_, _, _, slots, err := gameText(a[0])
	if err != nil {
		return err
	}
	entries := text.Entries(slots)
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(a[1], append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %d entries from %d slots to %s\n", len(entries), len(slots), a[1])
	return nil
}

func textImport(a []string) error {
	if len(a) != 3 && len(a) != 4 {
		return errUsage
	}
	f, units, c, slots, err := gameText(a[0])
	if err != nil {
		return err
	}
	if len(a) == 4 {
		// Slots come from the original; the translations go into BASE.
		f, err = openDat(filepath.Join(a[0], "SYSDIR", "BOOT.BIN"), a[3])
		if err != nil {
			return err
		}
		var base []maindat.Plain
		if _, base, err = f.Unpack(); err != nil {
			return err
		}
		if len(base) != len(units) {
			return fmt.Errorf("%s: %d units, original has %d", a[3], len(base), len(units))
		}
		units = base
	}
	b, err := os.ReadFile(a[1])
	if err != nil {
		return err
	}
	var entries []text.Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	sp := newSpace(units)
	r, probs := text.Apply(units, slots, entries, c, sp.alloc)
	for _, p := range probs {
		fmt.Fprintf(os.Stderr, "not written: %s: %s (%q)\n", p.Where, p.Reason, p.EN)
	}
	if sp.err != nil {
		return sp.err
	}
	if err := f.Pack(units); err != nil {
		return err
	}
	if err := os.WriteFile(a[2], f.Raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d slots in place, %d strings moved (%d JPEGs re-encoded), %d not written\n",
		a[2], r.Written, r.Moved, sp.shrunk, len(probs))
	return nil
}

// space hands out free bytes inside chunk units. It frees them by
// re-encoding the chunk's largest JPEGs at a lower quality (docs/0004).
type space struct {
	units  []maindat.Plain
	free   map[int][][2]int // unit to free [offset, length] runs
	ascii  map[int]bool     // units whose free space takes ASCII only
	done   map[[2]int]int   // unit and image index to qualities tried
	shrunk int
	err    error
}

// shrinkQualities are tried in order. The disc JPEGs are about quality 85;
// quality 80 frees about 4% of the JPEG bytes in chunk 0.
var shrinkQualities = []int{80, 75}

func newSpace(units []maindat.Plain) *space {
	s := &space{units: units, free: map[int][][2]int{}, ascii: map[int]bool{}, done: map[[2]int]int{}}
	for i := range units {
		if units[i].Name == "body" {
			if off, n := gimFreeTail(units[i].Data); n > 0 {
				s.free[i] = append(s.free[i], [2]int{off, n})
				// The atlas is RGBA 5551: bit 7 of each high byte is alpha.
				// ASCII bytes keep every pixel transparent.
				s.ascii[i] = true
			}
		}
	}
	return s
}

// gimFreeTail finds the first swizzled GIM in data and returns the empty
// tile bands at the end of its pixel data, less one band as a margin. In the
// header unit this is the bottom of the HUD atlas, which no sprite uses
// (docs/0004).
func gimFreeTail(data []byte) (int, int) {
	off := bytes.Index(data, []byte("MIG.00.1PSP\x00"))
	if off < 0 {
		return 0, 0
	}
	g, err := gim.Decode(data[off:])
	if err != nil || g.Order != 1 {
		return 0, 0
	}
	stride := 0
	switch g.Format {
	case gim.RGBA8888:
		stride = g.Width * 4
	case gim.Index8:
		stride = g.Width
	case gim.Index4:
		stride = g.Width / 2
	default:
		stride = g.Width * 2
	}
	stride = (stride + 15) &^ 15
	band := stride * 8
	start := off + g.ImageOffset
	bands := (g.Height + 7) / 8
	first := bands
	for first > 0 {
		if bytes.Count(data[start+(first-1)*band:start+first*band], []byte{0}) != band {
			break
		}
		first--
	}
	first++ // margin
	if first >= bands {
		return 0, 0
	}
	return start + first*band, (bands - first) * band
}

func (s *space) alloc(unit int, data []byte) (int, bool) {
	u := &s.units[unit]
	n := len(data)
	if s.err != nil {
		return 0, false
	}
	if s.ascii[unit] {
		for _, c := range data {
			if c >= 0x80 {
				return 0, false
			}
		}
	}
	for {
		for i, r := range s.free[unit] {
			if r[1] >= n {
				s.free[unit][i] = [2]int{r[0] + n, r[1] - n}
				return r[0], true
			}
		}
		if !strings.HasPrefix(u.Name, "chunks/") {
			return 0, false
		}
		list, err := images.List(u.Data, uint32(u.Off))
		if err != nil {
			s.err = err
			return 0, false
		}
		// The largest JPEG with the fewest tries frees the most.
		var pick *images.Entry
		tries := len(shrinkQualities)
		for i := range list {
			e := &list[i]
			k := s.done[[2]int{unit, e.Index}]
			if e.Type == images.TypeJPEG && k < len(shrinkQualities) && (k < tries || k == tries && e.Size > pick.Size) {
				pick, tries = e, k
			}
		}
		if pick == nil {
			return 0, false
		}
		s.done[[2]int{unit, pick.Index}]++
		freed, err := images.ShrinkJPEG(u.Data, *pick, shrinkQualities[tries])
		if err != nil {
			s.err = fmt.Errorf("%s: %w", u.Name, err)
			return 0, false
		}
		if freed > 0 {
			s.shrunk++
			s.free[unit] = append(s.free[unit], [2]int{pick.Off + pick.Size - freed, freed})
		}
	}
}

func writePNG(path string, g *gim.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, g.Image); err != nil {
		_ = f.Close() // the encode error is reported
		return err
	}
	return f.Close()
}

func gimToPNG(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	b, err := os.ReadFile(a[0])
	if err != nil {
		return err
	}
	g, err := gim.Decode(b)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %dx%d, format %d, order %d\n", a[0], g.Width, g.Height, g.Format, g.Order)
	return writePNG(a[1], g)
}

func gimScan(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	b, err := os.ReadFile(a[0])
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a[1], 0o755); err != nil {
		return err
	}
	mark := []byte("MIG.00.1PSP\x00")
	n, bad := 0, 0
	for off := bytes.Index(b, mark); off >= 0; {
		g, err := gim.Decode(b[off:])
		if err != nil {
			bad++
		} else if err := writePNG(filepath.Join(a[1], fmt.Sprintf("%08x_%dx%d.png", off, g.Width, g.Height)), g); err != nil {
			return err
		} else {
			n++
		}
		next := bytes.Index(b[off+1:], mark)
		if next < 0 {
			break
		}
		off += 1 + next
	}
	fmt.Printf("%s: %d images written, %d not decoded\n", a[0], n, bad)
	return nil
}

// imageRef is one image table entry with its export file name. Equal images
// share the name of their first use.
type imageRef struct {
	unit  *maindat.Plain
	entry images.Entry
	name  string
}

// gameImages lists the images of dat, or of the game's main.dat if dat is
// empty. Export names come from image contents, so dat must hold the original
// images (a text import leaves them unchanged).
func gameImages(gamedir, dat string) (*maindat.File, []maindat.Plain, []imageRef, error) {
	if dat == "" {
		dat = filepath.Join(gamedir, "USRDIR", "GameData", "01_UmdTitle", "main.dat")
	}
	f, err := openDat(filepath.Join(gamedir, "SYSDIR", "BOOT.BIN"), dat)
	if err != nil {
		return nil, nil, nil, err
	}
	_, units, err := f.Unpack()
	if err != nil {
		return nil, nil, nil, err
	}
	var refs []imageRef
	names := map[[20]byte]string{}
	for i := range units {
		u := &units[i]
		if u.Name == "body" {
			// The header unit has GIMs outside any image table (the HUD).
			mark := []byte("MIG.00.1PSP\x00")
			for off := bytes.Index(u.Data, mark); off >= 0; {
				if off+24 <= len(u.Data) {
					size := 16 + int(binary.LittleEndian.Uint32(u.Data[off+20:]))
					if off+size <= len(u.Data) {
						e := images.Entry{Index: -1, Type: images.TypeGIM, Off: off, Size: size, TableOff: -1}
						refs = append(refs, imageRef{u, e, fmt.Sprintf("g_%06x.gim", off)})
					}
				}
				next := bytes.Index(u.Data[off+1:], mark)
				if next < 0 {
					break
				}
				off += 1 + next
			}
			continue
		}
		if !strings.HasPrefix(u.Name, "chunks/") {
			continue
		}
		list, err := images.List(u.Data, uint32(u.Off))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", u.Name, err)
		}
		chunk := strings.TrimPrefix(u.Name, "chunks/")[:3]
		for _, e := range list {
			sum := sha1.Sum(u.Data[e.Off : e.Off+e.Size])
			name, ok := names[sum]
			if !ok {
				name = fmt.Sprintf("c%s_%02d.%s", chunk, e.Index, images.Ext(e.Type))
				names[sum] = name
			}
			refs = append(refs, imageRef{u, e, name})
		}
	}
	return f, units, refs, nil
}

func imgExport(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	_, _, refs, err := gameImages(a[0], "")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a[1], 0o755); err != nil {
		return err
	}
	var index strings.Builder
	index.WriteString("chunk\tentry\ttype\twidth\theight\tsize\tsha1\tfile\n")
	written := map[string]bool{}
	for _, r := range refs {
		data := r.unit.Data[r.entry.Off : r.entry.Off+r.entry.Size]
		if !written[r.name] {
			if err := os.WriteFile(filepath.Join(a[1], r.name), data, 0o644); err != nil {
				return err
			}
			if r.entry.Type == images.TypeGIM {
				if g, err := gim.Decode(data); err == nil {
					if err := writePNG(filepath.Join(a[1], strings.TrimSuffix(r.name, ".gim")+".png"), g); err != nil {
						return err
					}
				}
			}
			written[r.name] = true
		}
		sum := sha1.Sum(data)
		chunk := "hdr"
		if strings.HasPrefix(r.unit.Name, "chunks/") {
			chunk = strings.TrimPrefix(r.unit.Name, "chunks/")[:3]
		}
		fmt.Fprintf(&index, "%s\t%d\t%s\t%d\t%d\t%d\t%s\t%s\n", chunk, r.entry.Index,
			images.Ext(r.entry.Type), r.entry.Width, r.entry.Height, r.entry.Size, hex.EncodeToString(sum[:]), r.name)
	}
	if err := os.WriteFile(filepath.Join(a[1], "index.tsv"), []byte(index.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("%d images, %d unique, written to %s\n", len(refs), len(written), a[1])
	return nil
}

func imgImport(a []string) error {
	if len(a) != 3 && len(a) != 4 {
		return errUsage
	}
	base := ""
	if len(a) == 4 {
		base = a[3]
	}
	f, units, refs, err := gameImages(a[0], base)
	if err != nil {
		return err
	}
	n, files := 0, map[string]bool{}
	for _, r := range refs {
		data, err := replacement(a[1], r)
		if err != nil {
			return fmt.Errorf("%s: %w", r.name, err)
		}
		if data == nil {
			continue
		}
		if r.entry.TableOff < 0 {
			if len(data) != r.entry.Size {
				return fmt.Errorf("%s: %d bytes, want %d", r.name, len(data), r.entry.Size)
			}
			copy(r.unit.Data[r.entry.Off:], data)
		} else if err := images.Replace(r.unit.Data, r.entry, data); err != nil {
			return fmt.Errorf("%s in %s: %w", r.name, r.unit.Name, err)
		}
		files[r.name] = true
		n++
	}
	if err := f.Pack(units); err != nil {
		return err
	}
	if err := os.WriteFile(a[2], f.Raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d files replaced %d images\n", a[2], len(files), n)
	return nil
}

// replacement reads the new data for r from dir: the file with the export
// name, or for a GIM a PNG with the same base name, encoded into the
// original GIM's format. It returns nil if dir has neither.
func replacement(dir string, r imageRef) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(dir, r.name))
	if err == nil || !errors.Is(err, os.ErrNotExist) || r.entry.Type != images.TypeGIM {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return data, err
	}
	pf, err := os.Open(filepath.Join(dir, strings.TrimSuffix(r.name, ".gim")+".png"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer pf.Close() //nolint:errcheck // read only
	img, err := png.Decode(pf)
	if err != nil {
		return nil, err
	}
	orig := r.unit.Data[r.entry.Off : r.entry.Off+r.entry.Size]
	return gim.Encode(orig, img)
}

// sysBND reads AdvSys.bnd and builds the Shift-JIS codec from its tables.
func sysBND(dir string) (*bnd.File, *sjis.Codec, error) {
	sys, err := readBND(filepath.Join(dir, "USRDIR", "Bnd", "AdvSys.bnd"))
	if err != nil {
		return nil, nil, err
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
	return sys, c, err
}

// fmgEntry is one message in a translation file.
type fmgEntry struct {
	File string `json:"file"`
	ID   int    `json:"id"`
	JA   string `json:"ja"`
	EN   string `json:"en"`
}

func fmgExport(a []string) error {
	if len(a) != 2 {
		return errUsage
	}
	sys, c, err := sysBND(a[0])
	if err != nil {
		return err
	}
	var out []fmgEntry
	for _, e := range sys.Entries {
		if !strings.HasSuffix(e.Name, ".fmg") {
			continue
		}
		f, err := fmg.Read(e.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}
		for i, raw := range f.Strings {
			if raw == nil {
				continue
			}
			ja, err := c.Decode(raw)
			if err != nil {
				return fmt.Errorf("%s string %d: %w", e.Name, i, err)
			}
			out = append(out, fmgEntry{e.Name, f.ID(i), ja, ""})
		}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(a[1], append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d messages\n", a[1], len(out))
	return nil
}

func fmgImport(a []string) error {
	if len(a) != 3 {
		return errUsage
	}
	sys, c, err := sysBND(a[0])
	if err != nil {
		return err
	}
	b, err := os.ReadFile(a[1])
	if err != nil {
		return err
	}
	var entries []fmgEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	type key struct {
		file string
		id   int
	}
	tr := map[key]string{}
	for _, e := range entries {
		if e.EN != "" {
			tr[key{e.File, e.ID}] = e.EN
		}
	}
	n := 0
	for k := range sys.Entries {
		e := &sys.Entries[k]
		if !strings.HasSuffix(e.Name, ".fmg") {
			continue
		}
		f, err := fmg.Read(e.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}
		for i := range f.Strings {
			en, ok := tr[key{e.Name, f.ID(i)}]
			if !ok || f.Strings[i] == nil {
				continue
			}
			raw, err := c.Encode(en)
			if err != nil {
				return fmt.Errorf("%s %d: %w", e.Name, f.ID(i), err)
			}
			f.Strings[i] = raw
			n++
		}
		e.Data = f.Bytes()
	}
	if err := os.WriteFile(a[2], sys.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d messages translated\n", a[2], n)
	return nil
}
