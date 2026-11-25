# King's Field Additional I and II translation research

Format research and tools for the PSP games King's Field Additional I (ULJS-00070) and King's Field Additional II (ULJS-00076). The goal is an English translation that runs on PPSSPP and on PSP hardware with custom firmware.

The repository holds no game data and no keys. The scripts read both from your own disc images.

## Status

| Area | State |
|---|---|
| Disc file system, BND, FMG | Parsed |
| `main.dat` container and encryption | Decoded and verified for both games |
| Text | Locations, IDs, names and markup known |
| Fonts | Format known; ASCII glyphs present |
| Movies | All have burned-in Japanese narration |
| Go tools (`kfa`) | ISO read and in-place patch, BND, `main.dat` verify, proportional fonts |

## Requirements

- The two archives in the repository root:
  - `King's Field - Additional I (Japan, Asia).zip`
  - `King's Field - Additional II (Asia).zip`
- Optional reference material in the root: `Adventure Player (Japan).zip` and `ADVPStudio101.exe` ([0007](docs/0007-adventure-player.md)).
- Go (version in `go.mod`), Python 3, `7z`, `openssl`, `ffmpeg`.
- For executable analysis: Ghidra 12.2, JDK 25 or later, and the [ghidra-allegrex](https://github.com/kotcrab/ghidra-allegrex) extension v21.6.

## Use

```sh
scripts/extract_iso.sh
python3 -I scripts/maindat.py \
  research/extract/kfaI/PSP_GAME/SYSDIR/BOOT.BIN \
  research/extract/kfaI/PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat \
  research/maindat/kfaI
python3 -I scripts/scene_text.py research/maindat/kfaI research/maindat/kfaI/text.tsv
```

Output goes to `research/`, which git ignores. Use `kfaII` for Additional II.

### Go tools

```sh
go build -o kfa ./cmd/kfa
./kfa                      # lists commands
KFA_DATA=$PWD/research go test ./...
make check                 # format, vet, lint, test, markdown lint, tidy
```

[0002](docs/0002-tool-planning.md) lists every command and its state. [0009](docs/0009-test-builds.md) describes the test builds for PPSSPP.

| Script | Action |
|---|---|
| `scripts/extract_iso.sh` | Unzip both discs and extract their file trees |
| `scripts/maindat.py` | Verify and decrypt `main.dat` |
| `scripts/scene_text.py` | List text through engine structures |
| `scripts/text_refs.py` | Resolve text IDs in actions; count names |
| `scripts/text_survey.py` | Count text by kind (scan) |
| `scripts/chunk_commands.py` | Walk events and actions |
| `scripts/match_functions.py` | Carry function names from Adventure Player |
| `scripts/bnd_list.py` | List BND entries; `--out` unpacks one BND |
| `scripts/fmg_dump.py` | Print FMG strings |
| `scripts/font_render.py` | Render font glyph layers to PNG |
| `scripts/font_pages.py` | Check glyph pages against textures |
| `scripts/movie_sheets.sh` | Write movie contact sheets |
| `scripts/pmf_streams.py` | Count packets per stream in a PMF movie |
| `scripts/build_vwf_test.sh` | Build the proportional-font test image |
| `scripts/prx_unreferenced.py` | List functions no relocation or branch refers to |
| `scripts/headless_shots.sh` | Screenshots of an image at emulated times, no window |
| `scripts/ghidra/analyze.sh` | Decompile the executables with Ghidra |
| `scripts/boot_info.py`, `gameinfo_dump.py`, `maindat_survey.py` | Early survey tools |

## Documents

| Document | Subject |
|---|---|
| [0001](docs/0001-initial-findings.md) | Discs, file layout, BND and FMG formats |
| [0002](docs/0002-tool-planning.md) | Tool plan |
| [0003](docs/0003-maindat.md) | `main.dat` container and keys |
| [0004](docs/0004-text.md) | Text location, IDs, names and markup |
| [0005](docs/0005-fonts.md) | Fonts |
| [0006](docs/0006-movies.md) | Movies |
| [0007](docs/0007-adventure-player.md) | Adventure Player disc and authoring tool |
| [0008](docs/0008-open-questions.md) | Open questions |
| [0009](docs/0009-test-builds.md) | Test builds |
| [0010](docs/0010-subtitle-patch.md) | Subtitle patch |
| [0011](docs/0011-movie-script.md) | Movie script and draft English |
