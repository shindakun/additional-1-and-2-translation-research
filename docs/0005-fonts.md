# 0005: Fonts

`USRDIR/Font` holds 21 game fonts (`BitFont_Game00`–`20`) and `Debug.bnd`. `AdvMenu.bnd` holds a font BND also named `BitFont_Game12.bnd`. It differs from the file in `USRDIR/Font` in 1,542 bytes, all in `Tex01`. `scripts/font_render.py` renders the glyph layers to PNG.

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
| `0x10` | u16 a |
| `0x12` | u16 b |
| `0x14` | u16 c |
| `0x16` | u16, 0 |

In all game fonts, the 95 ASCII glyphs have a = 0, b = 8, c = 8. In `Debug.bnd` the ASCII glyphs have varied values, for example (2, 1, 4) and (0, 5, 6). These values match left offset, glyph width and advance. The engine's use of them is not verified.

## Textures

GIM, 512 × 512, 8-bit index, PSP swizzle (16-byte × 8-row tiles). Each pixel index holds four glyph layers. Palettes 0–3 of `Plt00.plt` each map the index to one layer's alpha (0, 85, 170 or 255). Palettes 4–15 map every index to alpha 0.

The glyph table does not show how a glyph selects its texture and layer.

## Latin coverage

Layer 0 of `Tex00` holds:

- Printable ASCII `0x20`–`0x7E` as 8 × 16 half-width glyphs. Code `0x5C` draws `¥`.
- Shift-JIS symbols such as `§`, `°`, `±`, `×`, `÷`.
- Greek capitals.

English text can use the existing ASCII glyphs at a fixed 8-pixel advance: 60 characters across the 480-pixel screen.
