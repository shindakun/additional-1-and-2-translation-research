# 0010: Subtitle patch

Goal: English subtitles over the movies without new video. Addresses are for Additional I `BOOT.BIN` (Ghidra, base 0). [0006](0006-movies.md) lists the movies.

## Why a code patch

A movie plays as its own scene. The event and text system does not run during the scene, so no text action can draw over a movie. The scene's draw function is the patch point.

## Patch site

`CSimpleMovieScene::Draw`, `0x3EB28`, 28 bytes:

| Address | Instruction |
|---|---|
| `0x3EB28` | `addiu sp, sp, -0x20` |
| `0x3EB2C` | `sw ra, 0x10(sp)` |
| `0x3EB30` | `jal 0xB35D0` (`CAcffMovie::Draw`) |
| `0x3EB34` | `addiu a0, a0, 0x130` |
| `0x3EB38` | `lw ra, 0x10(sp)` |
| `0x3EB3C` | `jr ra` |
| `0x3EB40` | `addiu sp, sp, 0x20` |

The patch changes only the `jal` target at `0x3EB30` to the new code. The loader relocates that `jal` (`R_MIPS_26`), and the relocation works for any target in `.text`.

## New code

The new code runs once per movie frame draw:

1. Call `CAcffMovie::Draw` (`0xB35D0`) with the original `a0`.
2. Add 1 to a frame counter.
3. Find the subtitle whose frame range holds the counter.
4. Draw a box, then the subtitle text.
5. Return.

Engine functions it calls:

| Address | Function | Arguments |
|---|---|---|
| `0x14C1C` | `FsFontSetCurrent` | `a0` = font, the word at `0x1262F4` |
| `0x14EA0` | `FsFontSetBaseColor` | `a0` = color, `0xFFFFFFFF` is white |
| `0x15168` | `FsFontSetAlignment` | `a0` = alignment |
| `0x15288` | `FsFontDrawText` | `$f12` = x, `$f13` = y (float), `a0` = Shift-JIS string |
| `0x13388` | Screen quad | `CAcffMovie::Draw` uses it to fill 480 × 272 with a color |
| `0xFE6F0` | `sprintf` | Frame counter test only |

`CAdvMenuPrinter::DrawText` (`0x2FB4C`) shows the font call sequence: `FsFontSetAlignment(align)`, then `FsFontDrawText(x, y, text)`.

## Code space

`scripts/prx_unreferenced.py` lists functions that no relocation and no branch refers to. Additional I has 493 such functions, 81,660 bytes. The largest is `0xF232C`, 6,552 bytes, matched to zlib `inflate`.

The loader still applies the relocations of the old code: 61 entries inside `0xF232C`–`0xF3CC4` (23 `R_MIPS_26`, 19 `R_MIPS_HI16`, 19 `R_MIPS_LO16`). The patch sets their type to `R_MIPS_NONE` (0) so they do not change the new code.

The new code is position-independent. A `bal` gives its own address; data, strings and engine function addresses are computed from it. The PRX loads at a base address that is not known at build time, so absolute addresses in new code would need new relocation entries.

## Subtitle table

Stored in the code space after the code:

| Field | Size |
|---|---|
| First frame | u16 |
| Last frame | u16 |
| Text offset in the table | u16 |
| Position (top or bottom) | u16 |

Text is Shift-JIS with ASCII. Frame numbers count calls of the patched draw function.

## Timing

The narration appears at known times in the video ([0006](0006-movies.md)). The counter rate is not known: the game loop and the 29.97 fps video can differ. A first test build draws the counter on screen so the rate can be read against the video.

## Test builds

| Build | Shows | State |
|---|---|---|
| Frame counter | Hook runs; counter rate | Not built |
| One subtitle | Font, box and position | Not built |
| Full table, movie000 | Timing against the narration | Not built |

## Other discs

Additional II needs the same analysis: its `BOOT.BIN` differs, so every address above changes.
