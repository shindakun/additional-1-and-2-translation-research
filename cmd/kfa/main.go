// Command kfa extracts, inspects and patches King's Field Additional I and II
// disc data. One binary, several subcommands; see docs/0002-tool-planning.md.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/shindakun/additional-1-and-2-translation-research/internal/bnd"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/font"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/iso9660"
	"github.com/shindakun/additional-1-and-2-translation-research/internal/maindat"
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
	case "font vwf":
		err = fontVWF(args[2:])
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
  font vwf IN.bnd OUT.bnd [GAP] [SPACE]
                                   Proportional ASCII metrics in every font
                                   inside IN (default gap 1, space 4)
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
