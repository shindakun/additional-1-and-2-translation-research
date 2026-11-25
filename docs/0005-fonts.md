# 0005: Fonts

`USRDIR/Font` holds 21 game fonts (`BitFont_Game00`–`20`) and `Debug.bnd`. `AdvMenu.bnd` holds a font BND also named `BitFont_Game12.bnd`. It differs from the file in `USRDIR/Font` in 1,542 bytes, all in `Tex01`. `scripts/bnd_list.py --out` unpacks a font BND. `scripts/font_render.py` renders its glyph layers to PNG.

## Font BND

| Entry | Contents |
|---|---|
| `Ccm.ccm` | Code ranges and glyph table |
| `Plt00.plt`, `Plt01.plt` | 16 palettes of 256 RGBA entries each |
| `Tex00.gim`, `Tex01.gim` | Glyph textures |

## CCM

Little-endian.

| Offset | Field |
|---|---|
| `0x00` | `0x00010000` |
| `0x04` | File size |
| `0x10` | Range count (u16), glyph count (u16) |
| `0x14` | Range table offset |
| `0x18` | Glyph table offset |

Range, 12 bytes: first code, last code, first glyph index. Codes are Shift-JIS values. Game fonts have 103 ranges and 7,485 glyphs. The first range is `0x20`–`0x7F` at glyph 0.

Glyph, 24 bytes:

| Offset | Field |
|---|---|
| `0x00` | u0, v0, u1, v1 (float32, 0–1 on a 512 × 512 texture) |
| `0x10` | Left offset (s16) |
| `0x12` | Drawn width (s16) |
| `0x14` | Advance (s16) |
| `0x16` | Page (s16): texture = page ÷ 4, layer = page mod 4 |

`FUN_00015F00` (Additional I, `FontDrawChar` in Adventure Player) adds the left offset to the pen position, draws the glyph at the drawn width, and adds the advance. `FUN_00016EA0` (`FontGetLineSize`) sums left offset and advance to measure a line. Per-glyph widths in the CCM therefore change text layout without code changes. [0009](0009-test-builds.md) has a test build to confirm this on screen. `kfa font vwf` sets proportional ASCII metrics from the glyph ink (`internal/font`).

In all game fonts, the 95 ASCII glyphs have offset 0, width 8, advance 8, page 0. In `Debug.bnd` the ASCII glyphs have varied values, for example (2, 1, 4) and (0, 5, 6).

Pages in `BitFont_Game00`:

| Page | Glyphs |
|---|---|
| 0 | 1,072 (ASCII and 976 more) |
| 1–6 | 1,024 each |
| 7 | 269 |

`scripts/font_pages.py` tests 400 sampled glyphs: 399 have ink in the predicted texture and layer. The exception is the space glyph, whose cell is blank in its own layer.

## Textures

GIM, 512 × 512, 8-bit index, PSP swizzle (16-byte × 8-row tiles). Each pixel index holds four glyph layers. `PltNN.plt` belongs to `TexNN.gim`. Palettes 0–3 each map the index to one layer's alpha. Palettes 4–15 map every index to alpha 0.

In `BitFont_Game12`, `Plt00` uses indices 0–15: each is one of the 16 on/off combinations of the four layers. `Plt01` uses indices 0–31: layers 0–2 are on or off, layer 3 has four levels (0, 85, 170, 255).

## Latin coverage

Layer 0 of `Tex00` holds:

- Printable ASCII `0x20`–`0x7E` as 8 × 16 half-width glyphs. Code `0x5C` draws `¥`.
- Shift-JIS symbols such as `§`, `°`, `±`, `×`, `÷`.
- Greek capitals.

English text can use the existing ASCII glyphs at a fixed 8-pixel advance: 60 characters across the 480-pixel screen.

## Padding

`FsFontEnd` draws text with linear texture filtering (`sceGuTexFilter(5, 1)` in `FUN_000085A4`). The ASCII cells are 8 × 16 pixels with no blank column between them: the next glyph's ink starts in the next texture column. At 1× the samples fall on texel centers and nothing shows. At a raised render resolution in PPSSPP, samples fall between texels, and each letter shows a sliver of its neighbor (`thalt`, `wi l l l`).

`kfa font pad IN.bnd OUT.bnd` moves the 94 printable ASCII glyphs (`0x21`–`0x7E`) to page 7 (`Tex01`, layer 3), each in a 10 × 18 slot with a blank border, and changes their texture coordinates. Each pixel keeps the states of the other three layers. In `BitFont_Game12`, page 7 has room for about 800 such slots. All 21 `BitFont_GameNN.bnd` fonts and the copy in `AdvMenu.bnd` pad without a size change. `scripts/build_english.sh` pads them all.

The test checks that each moved glyph has the original ink and a blank border, and that layers 0–2 of `Tex01` do not change. Headless PPSSPP at 1× shows the subtitles and the menu text unchanged. The effect at a raised render resolution needs a check in the PPSSPP app.
