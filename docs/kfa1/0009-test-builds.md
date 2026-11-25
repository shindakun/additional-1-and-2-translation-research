# 0009: Test builds

Test builds are images for PPSSPP. They go to `research/builds`, which git ignores. Each build changes one thing so that a difference on screen has one cause.

## PPSSPP use

- Test runs use a copy of the app with its own bundle ID (`org.ppsspp.kfa-test`) and a memory stick folder under `research/ppsspp`, set in that bundle's preferences. The user's PPSSPP settings stay unchanged.
- PPSSPP 1.20.4 rejects extra command-line arguments with an abort: only one file path is allowed.
- In PPSSPP 1.20.4 the WebSocket debugger pauses the emulated CPU for every memory read and screenshot. Three test runs crashed after such requests. Do not use the debugger with 1.20.4.
- The headless build saves screenshots at emulated times and needs no window. Use it for all screenshots.

## Headless PPSSPP

Build from the PPSSPP source (`research/tools/ppsspp-src`, with submodules, commit `ee4594f`). Apply `scripts/ppsspp-headless-stick.patch` (`git apply`) for analog stick input. `HEADLESS_CROSS` builds the headless target without the SDL front end:

```sh
cmake -G Ninja -DCMAKE_BUILD_TYPE=Release -DHEADLESS=ON -DHEADLESS_CROSS=ON ..
ninja PPSSPPHeadless
```

`scripts/headless_shots.sh IMAGE OUTDIR SECONDS...` runs an image with the software renderer and saves a PNG at each emulated second. It passes `--memstick` with a folder under `research/ppsspp`; without it, headless uses `~/Library/memstick`. 145 emulated seconds take about 20 seconds. A crash prints the MIPS call stack, with addresses relative to the module (`AdventurePlayer.text+...`).

Build `kfa` first:

```sh
make kfa
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

## Text test

`scripts/build_english.sh [GAME]` builds `research/builds/GAME-english.iso` (default `kfaI`) from `translations/GAME` ([0004](0004-text.md#translation-files)). It builds `bin/kfa`, exports the Japanese from the extracted disc and fills in the English, redraws the image labels from the specs (Pillow and the fonts in the specs), writes the images, then the text, into `main.dat`, writes the FMG messages into `AdvSys.bnd`, applies the subtitle and chunk base patches to `EBOOT.BIN` (with no subtitle tables, only the chunk base patch), and pads the fonts. It verifies `main.dat` and leaves no other files. The same input gives the same image byte for byte. This is the image to play.

`scripts/make_patch.sh GAME VERSION` builds the image and writes `dist/GAME-english-VERSION/`: an xdelta3 patch from the original ISO, a readme with the commit and SHA-256 sums. It applies the patch to a copy of the original and checks the result byte for byte. The encoder uses a 1 GB source window (`-B`); with the default window, the files that move to the end of the image go into the patch whole. The patch for Additional I is about 184 MB: chunks with new text re-encrypt, and AES output has no matches.

Test builds for one change go to `research/builds` under a name that says what they test, and are deleted after the test. Screenshots that show a result stay in `research/ppsspp/examples`.

The headless memory stick (`research/ppsspp/headless-ms`) must hold no save for these input scripts: with a save, the title selects Load.

In town, Start opens the system menu; Down, Down, Circle opens the options screen.

Headless input: `--press=SECONDS:MASK[:DURATION],...` holds buttons at emulated times (default 0.15 s). Masks: Select `0x1`, Start `0x8`, Up `0x10`, Right `0x20`, Down `0x40`, Left `0x80`, Triangle `0x1000`, Circle `0x2000`, Cross `0x4000`, Square `0x8000`. With the stick patch, `0x10000000`, `0x20000000`, `0x40000000` and `0x80000000` push the left stick up, right, down and left.

From boot, Start at 5 and 7 s skips the title movie. Circle at 10, 12, 14 and 16 s starts a new game. The opening movie plays from about 19 s; Circle at 28 s skips it. The town map shows at about 31 s.

The town map cursor moves with the left stick only, about 430 pixels per second. It starts at about (255, 135). To enter the tavern: stick left for 0.16 s at 34 s, stick up for 0.06 s at 35 s, Circle at 36 s. Circle at 39 s and after advances the text. To enter the shop: stick right for 0.42 s at 34 s, stick up for 0.05 s at 35 s, Circle at 36, 39, 42, 45 and 48 s (Buy, Items, Herb). Cross in town opens the status overlay.
