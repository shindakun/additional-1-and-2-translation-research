# 0002: Tool planning

## Goal

Extract text and assets from King's Field Additional I and II. Support an English translation. Build an ISO that runs on PPSSPP and on PSP hardware with custom firmware.

## Layout

The tools are in Go. One binary, `kfa`, has a subcommand for each format. Format code has no dependency on the CLI.

```text
cmd/kfa/            CLI
internal/iso9660/   UMD image read and write
internal/bnd/       BND containers, both variants
internal/fmg/       FMG message tables
internal/sjis/      Shift-JIS codec from the game's JIS tables
internal/gim/       GIM decode and encode (PNG through image/png)
internal/maindat/   main.dat (ADVP) container
internal/adv/       Scene chunks
internal/patch/     Executable patches
scripts/            Research scripts
docs/               Numbered design documents
```

Each format package has `Read(io.ReaderAt) (T, error)` and `Write(io.Writer, T) error`. A format is complete when read then write gives the original bytes.

## main.dat

`kfa dat pack` encrypts each unit with AES-128-CBC and writes the salted MD5. The verifier `FUN_000544a4` checks the MD5. Its use on the disc load path is not known, so the pack always writes a correct MD5. Go's `crypto/aes`, `crypto/cipher` and `crypto/md5` cover all of it. Keys come from `BOOT.BIN` at run time and are not stored in the repository.

## Shift-JIS

`AdvSys.bnd` holds `jis2ucs.bin` and `ucs2jis.bin`. `internal/sjis` loads them from the extracted disc. The codec then matches the game's mapping, and the module needs no encoding dependency.

## Commands

| Command | Action | State |
|---|---|---|
| `kfa iso ls <iso>` | List files with sector and size | Built |
| `kfa iso extract <iso> <dir>` | Write all files to a directory tree | Built |
| `kfa iso put <iso> <path> <file>` | Replace a file in place; the new data must fit its sectors | Built |
| `kfa iso build <dir> <iso>` | Build a UMD image from a directory tree | Not built |
| `kfa bnd ls <bnd>` | List entries, nested BNDs included | Built |
| `kfa bnd unpack <bnd> <dir>` | Write entries | Built |
| `kfa bnd pack <dir> <bnd>` | Build a BND from a directory and a manifest | Not built; `internal/bnd` writes BNDs |
| `kfa dat verify <boot> <main.dat>` | Check keys and MD5; re-encrypt every unit | Built |
| `kfa dat unpack <boot> <main.dat> <dir>` | Write every unit decrypted | Built |
| `kfa dat pack <boot> <main.dat> <dir> <out>` | Encrypt units from a directory and re-sign | Built for units of unchanged size; a new layout is not built |
| `kfa font vwf <in.bnd> <out.bnd> [gap] [space]` | Proportional ASCII metrics in every font inside a BND | Built |
| `kfa patch counter <boot> <out>` | Frame counter over movies ([0010](0010-subtitle-patch.md)) | Built |
| `kfa text export <gamedir> <json>` | Translatable `main.dat` strings ([0004](0004-text.md)) | Built |
| `kfa text import <gamedir> <json> <out>` | Translations into `main.dat`, in place | Built; strings must fit their original length |
| `kfa fmg export`, `kfa fmg import` | FMG text out and in ([0013](0013-fmg.md)) | Built; byte-exact rebuild, seen in game |
| `kfa gim topng`, `kfa gim scan` | GIM to PNG; every GIM inside a file ([0012](0012-images.md)) | Built |
| `kfa img export`, `kfa img import` | Chunk images out and in, in place ([0012](0012-images.md)) | Built; JPEG and BMP edits seen in game |
| `kfa gim frompng` | PNG to GIM | Not built |
| `kfa build <project>` | Build a finished ISO from translation files | Not built |

## Translation files

One JSON file per source text file. A record has string ID, Japanese source, translation and note. The importer compares markup tokens (`#c[...]`, `#c`, `#m[n]`, `#b`, line breaks). A missing token stops the build. An empty translation keeps the source.

## Build pipeline

1. Extract the original ISO to a work directory.
2. Unpack all BNDs, nested and in `main.dat`.
3. Import translations into FMG files and scene chunks.
4. Replace font and texture assets.
5. Pack containers from the inside out.
6. Install the patched executable as `PSP_GAME/SYSDIR/EBOOT.BIN`.
7. Build the ISO.

The build does not change the original ISO. All output goes to the work directory.

## ISO build

The Go standard library has no ISO9660 package. `internal/iso9660` implements the UMD subset: primary volume descriptor, path tables, directory records, 2048-byte sectors. No Joliet, no Rock Ridge.

The builder keeps the original file order. `UMD_DATA.BIN` and `PSP_GAME/SYSDIR` come first. `main.dat` keeps its sector alignment. The `Dmy` filler files shrink when `main.dat` grows. Image size stays a multiple of 2048.

## Executable

`EBOOT.BIN` is encrypted. `BOOT.BIN` is a plain copy. PPSSPP and custom firmware run a plain ELF at `EBOOT.BIN`. Patches go into `BOOT.BIN`; the result becomes `EBOOT.BIN`.

A patch is a named list of address, original bytes and new bytes. The patcher stops if the original bytes do not match.

## Analysis tools

| Tool | Requirement |
|---|---|
| Ghidra | 12.2, JDK 25 or later, native decompiler for the host |
| ghidra-allegrex | v21.6 (Ghidra 12.2 build), in `Ghidra/Extensions` |
| REA | `rea-agents`, Ghidra provider |

Load `BOOT.BIN` as a relocatable ELF at base 0. Adventure Player's `BOOT.BIN` keeps its symbols; `scripts/match_functions.py` carries the names to the King's Field builds ([0007](0007-adventure-player.md)).

## Order of work

| Step | Work | Needs | State |
|---|---|---|---|
| 1 | Module setup, `iso extract`, round-trip test harness | | Built |
| 2 | `bnd` unpack and pack, both variants | | Library built; CLI unpack built |
| 3 | `sjis`, `fmg` export and import | 2 | Built |
| 4 | `iso build`; boot an unchanged build on PPSSPP | 1 | In-place `iso put` built; full build not built |
| 5 | `internal/maindat` from [0003](0003-maindat.md): keys from `BOOT.BIN`, AES-CBC, MD5 | | Built |
| 6 | `dat unpack`, `dat pack` | 5 | Built; same-size units only |
| 7 | Text export and in-place import ([0004](0004-text.md)) | 6 | Built and seen in game; strings longer than the source need a new layout |
| 8 | Test CCM glyph metrics for variable-width English in PPSSPP ([0005](0005-fonts.md)) | 3 | Test build ready ([0009](0009-test-builds.md)) |
| 9 | Executable patches: Latin text layout, movie subtitles ([0006](0006-movies.md)) | 7, 8 | Not built |
| 10 | `kfa build` end to end | all | Not built |

Steps 7 and 8 hold most of the open questions ([0008](0008-open-questions.md)).

## Testing

- Each format package has a round-trip test on original files.
- Tests read game files from the path in `KFA_DATA`: the `research` directory written by `scripts/extract_iso.sh` and `scripts/maindat.py`. Tests skip when it is not set. Game data is not committed.

```sh
KFA_DATA=$PWD/research go test ./...
```

| Package | Test on disc data |
|---|---|
| `internal/bnd` | 276 BNDs on all three discs, nested included, re-encode byte for byte |
| `internal/maindat` | Every unit of both `main.dat` files decrypts, re-encrypts and re-signs to the original; unpack then pack gives the original; a changed byte changes the MD5 |
| `internal/iso9660` | Every file of both images matches the 7-Zip extraction; in-place replacement reads back |
| `internal/font` | CCM round trip; proportional metrics for all 95 ASCII glyphs |

- Small synthetic fixtures cover edge cases: empty BND, FMG with unused IDs, markup tokens.
- A built ISO passes when it boots in PPSSPP and shows the edited text.

## Not in the first release

- Audio (`.at3`, `.hvg`) changes.
- Movie changes. All 11 movies have burned-in Japanese narration ([0006](0006-movies.md)).
- Distribution patches (xdelta, BPS). The build gives a full ISO.
