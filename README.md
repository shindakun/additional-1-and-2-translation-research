# King's Field Additional I and II translation research

Format research and tools for the PSP games King's Field Additional I (ULJS-00070) and King's Field Additional II (ULJS-00076). The goal is an English translation that runs on PPSSPP and on PSP hardware with custom firmware.

The repository holds no game data and no keys. The scripts read both from your own disc images.

## Status

| Area | State |
|---|---|
| Disc file system, BND, FMG | Parsed |
| `main.dat` container and encryption | Decoded and verified for both games |
| Text | Both games: all `main.dat` and FMG text a player sees has a first-draft English; chunks grow and relocate for longer English; seen in game |
| Images | Both games: the town map, status screen, HUD, shop panels and roulette images edited; seen in game |
| Fonts | Format known; ASCII glyphs padded against filtering slivers (`kfa font pad`) |
| Movies | All have burned-in Japanese narration. Additional I: English subtitle tables for all four movies. Additional II: tables for all seven. The openings are seen in game |
| Go tools (`kfa`) | ISO, BND, `main.dat` unpack and pack, text, FMG and image export and import, fonts, movie patch |

## Requirements

- The two archives in the repository root:
  - `King's Field - Additional I (Japan, Asia).zip`
  - `King's Field - Additional II (Asia).zip`
- Optional reference material in the root: `Adventure Player (Japan).zip` and `ADVPStudio101.exe` ([0007](docs/kfa1/0007-adventure-player.md)).
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

Output goes to `research/`, which git ignores.

To build the translated image:

```sh
make kfa
scripts/build_english.sh
```

The result is `research/builds/kfaI-english.iso` (`scripts/build_english.sh kfaII` for Additional II). The translations are in `translations/`, without Japanese; the build takes the Japanese from the disc ([0004](docs/kfa1/0004-text.md#translation-files), [0009](docs/kfa1/0009-test-builds.md)).

### Go tools

```sh
make kfa
./bin/kfa                      # lists commands
KFA_DATA=$PWD/research go test ./...
make check                 # format, vet, lint, test, markdown lint, tidy
```

[0002](docs/kfa1/0002-tool-planning.md) lists every command and its state. [0009](docs/kfa1/0009-test-builds.md) describes the test builds for PPSSPP.

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
| `scripts/make_patch.sh` | Build an xdelta3 patch and readme in `dist/` |
| `scripts/release.sh` | Tag and publish a patch as a GitHub pre-release (dry run without `--publish`) |
| `scripts/build_english.sh` | Build `research/builds/kfaI-english.iso` with every current translation |
| `scripts/map_boot.py` | Map `BOOT.BIN` addresses from one game to the other |
| `scripts/translation.py` | Convert between English-only translation files and full ones |
| `scripts/text_merge.py` | Carry translations into a new text export |
| `scripts/text_apply_tsv.py` | Set translations from an index-to-English TSV |
| `scripts/tail_breaks.py` | Give English lines the same trailing line breaks as their source |
| `scripts/text_center.py` | Center English like its space-padded source |
| `scripts/gen_items.py` | Fill found, used, priced and description entries of a translation file from the item tables |
| `scripts/item_desc.py` | Build English item and spell descriptions |
| `scripts/ppsspp-headless-stick.patch` | Analog stick input for PPSSPPHeadless `--press` |
| `scripts/relabel_jpeg.py` | Replace label text in a JPEG (byte budget) or PNG (Pillow) |
| `scripts/prx_unreferenced.py` | List functions no relocation or branch refers to |
| `scripts/headless_shots.sh` | Screenshots of an image at emulated times, no window |
| `scripts/ghidra/analyze.sh` | Decompile the executables with Ghidra |
| `scripts/boot_info.py`, `gameinfo_dump.py`, `maindat_survey.py` | Early survey tools |

## Documents

`docs/kfa1` covers Additional I and the shared formats. `docs/kfa2` covers Additional II and links to `docs/kfa1` where the games share a format.

### Additional I

| Document | Subject |
|---|---|
| [0001](docs/kfa1/0001-initial-findings.md) | Discs, file layout, BND and FMG formats |
| [0002](docs/kfa1/0002-tool-planning.md) | Tool plan |
| [0003](docs/kfa1/0003-maindat.md) | `main.dat` container and keys |
| [0004](docs/kfa1/0004-text.md) | Text location, IDs, names and markup |
| [0005](docs/kfa1/0005-fonts.md) | Fonts |
| [0006](docs/kfa1/0006-movies.md) | Movies |
| [0007](docs/kfa1/0007-adventure-player.md) | Adventure Player disc and authoring tool |
| [0008](docs/kfa1/0008-open-questions.md) | Open questions |
| [0009](docs/kfa1/0009-test-builds.md) | Test builds |
| [0010](docs/kfa1/0010-subtitle-patch.md) | Subtitle patch |
| [0011](docs/kfa1/0011-movie-script.md) | Movie script and draft English |
| [0012](docs/kfa1/0012-images.md) | Images and image text |
| [0013](docs/kfa1/0013-fmg.md) | FMG messages |
| [0014](docs/kfa1/0014-debug-screens.md) | Debug mode and debug screens |

### Additional II

| Document | Subject |
|---|---|
| [0001](docs/kfa2/0001-initial-findings.md) | Disc, differences from Additional I, first tool results |
| [0002](docs/kfa2/0002-boot-addresses.md) | `BOOT.BIN` addresses for the patches |
| [0003](docs/kfa2/0003-images.md) | Image text |
| [0004](docs/kfa2/0004-translation.md) | Translation and builds |
| [0005](docs/kfa2/0005-movies.md) | Movies and subtitles |
