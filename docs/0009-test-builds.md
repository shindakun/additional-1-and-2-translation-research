# 0009: Test builds

Test builds are images for PPSSPP. They go to `research/builds`, which git ignores. Each build changes one thing so that a difference on screen has one cause.

## PPSSPP use

- Test runs use a copy of the app with its own bundle ID (`org.ppsspp.kfa-test`) and a memory stick folder under `research/ppsspp`, set in that bundle's preferences. The user's PPSSPP settings stay unchanged.
- PPSSPP 1.20.4 rejects extra command-line arguments with an abort: only one file path is allowed.
- In PPSSPP 1.20.4 the WebSocket debugger pauses the emulated CPU for every memory read and screenshot. Three test runs crashed after such requests. Do not use the debugger with 1.20.4.
- The headless build saves screenshots at emulated times and needs no window. Use it for all screenshots.

## Headless PPSSPP

Build from the PPSSPP source (`research/tools/ppsspp-src`, with submodules). `HEADLESS_CROSS` builds the headless target without the SDL front end:

```sh
cmake -G Ninja -DCMAKE_BUILD_TYPE=Release -DHEADLESS=ON -DHEADLESS_CROSS=ON ..
ninja PPSSPPHeadless
```

`scripts/headless_shots.sh IMAGE OUTDIR SECONDS...` runs an image with the software renderer and saves a PNG at each emulated second. It passes `--memstick` with a folder under `research/ppsspp`; without it, headless uses `~/Library/memstick`. 145 emulated seconds take about 20 seconds. A crash prints the MIPS call stack, with addresses relative to the module (`AdventurePlayer.text+...`).

Build `kfa` first:

```sh
go build -o kfa ./cmd/kfa
```

## VWF test

Question: does the engine use the CCM metrics ([0005](0005-fonts.md))?

```sh
scripts/build_vwf_test.sh kfaI
```

Output: `research/builds/kfaI-vwf-test.iso`. The script replaces 22 files in place: the 21 `Font/BitFont_GameNN.bnd` files and `Bnd/AdvMenu.bnd`, whose nested font has new metrics too. Only the CCM glyph entries change.

| Glyph | Original (left, width, advance) | Test build | Pen step |
|---|---|---|---|
| Space | 0, 8, 8 | 0, 8, 4 | 4 |
| `i` | 0, 8, 8 | −3, 8, 5 | 2 |
| `l` | 0, 8, 8 | −2, 8, 5 | 3 |
| Capitals, `m` | 0, 8, 8 | 0, 8, 8 | 8 |

Capitals keep their step because their ink fills the 8-pixel cell.

What to check:

1. The title menu: chunk 1 holds `NEW GAME` and `LOAD GAME` as text. If the menu draws them with these fonts, the gap between the words is narrower.
2. Any ASCII text with lowercase `i`, `l` or spaces: the letters sit closer together.
3. The game boots and Japanese text looks unchanged.

A narrower gap confirms the engine applies CCM metrics, so English can be proportional without code patches.

Result: the build boots and regular text looks unchanged. The game shows little ASCII text, so the spacing change is not yet seen on screen. A subtitle test build ([0010](0010-subtitle-patch.md)) draws ASCII text and gives the next check.
