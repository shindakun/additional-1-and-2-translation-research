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

| Command | Action |
|---|---|
| `kfa iso extract <iso> <dir>` | Write all files to a directory tree |
| `kfa iso build <dir> <iso>` | Build a UMD image from a directory tree |
| `kfa bnd unpack <bnd> <dir>` | Write entries and `bnd.json` (IDs, names, order, version, alignment) |
| `kfa bnd pack <dir> <bnd>` | Build a BND from a directory and its manifest |
| `kfa fmg export <fmg> <json>` | Write ID, source and empty translation for each string |
| `kfa fmg import <fmg> <json> <out>` | Write an FMG with the translations |
| `kfa gim topng`, `kfa gim frompng` | Convert textures |
| `kfa dat unpack`, `kfa dat pack` | Decrypt and split `main.dat`; encrypt, rebuild and re-sign it |
| `kfa build <project>` | Build a finished ISO from translation files |

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

Load `BOOT.BIN` as a relocatable ELF at base 0.

## Order of work

| Step | Work | Needs |
|---|---|---|
| 1 | Module setup, `iso extract`, round-trip test harness | |
| 2 | `bnd` unpack and pack, both variants | |
| 3 | `sjis`, `fmg` export and import | 2 |
| 4 | `iso build`; boot an unchanged build on PPSSPP | 1 |
| 5 | `internal/maindat` from [0003](0003-maindat.md): keys from `BOOT.BIN`, AES-CBC, MD5 | |
| 6 | `dat unpack`, `dat pack` | 5 |
| 7 | Scene chunk format and text export | 6 |
| 8 | Font format (CCM, PLT, GIM), Latin glyphs | 3 |
| 9 | Executable patches for Latin text layout | 7, 8 |
| 10 | `kfa build` end to end | all |

Step 4 proves the build path before content changes. Step 7 is the main unknown. Steps 5 to 7 run in parallel with steps 1 to 4.

## Testing

- Each format package has a round-trip test on original files.
- Tests read game files from the path in `KFA_DATA`. Tests skip when it is not set. Game data is not committed.
- Small synthetic fixtures cover edge cases: empty BND, FMG with unused IDs, markup tokens.
- A built ISO passes when it boots in PPSSPP and shows the edited text.

## Not in the first release

- Audio (`.at3`, `.hvg`) and movie (`.pmf`) changes. Check movies for burned-in text first.
- Distribution patches (xdelta, BPS). The build gives a full ISO.
