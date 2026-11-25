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
4. Draw the subtitle text: black at an offset of one pixel, then white on top.
5. Return.

Engine functions it calls:

| Address | Function | Arguments |
|---|---|---|
| `0x14C1C` | `FsFontSetCurrent` | `a0` = font, the word at `0x1262F4` |
| `0x14EA0` | `FsFontSetBaseColor` | `a0` = color, `0xFFFFFFFF` is white |
| `0x15168` | `FsFontSetAlignment` | `a0` = alignment |
| `0x15288` | `FsFontDrawText` | `$f12` = x, `$f13` = y (float), `a0` = Shift-JIS string |
| `0x13388` | `Gfx_FastSprite` | Sprite or plain quad; see below |
| `0xFE6F0` | `sprintf` | Frame counter test only |

`CAdvMenuPrinter::DrawText` (`0x2FB4C`) shows the font call sequence: `FsFontSetAlignment(align)`, then `FsFontDrawText(x, y, text)`.

`CAcffMovie::Draw` calls `Gfx_FastSprite` with these registers:

| Register | Video frame | No picture |
|---|---|---|
| `a0`, `a1` | 480, 272 | 480, 272 |
| `a2` | 3 | 3 |
| `a3` (color) | `0xFFFFFFFF` | `0xFF000000` |
| `$f12`–`$f15` (x, y, w, h) | 0, 0, 480, 272 | 0, 0, 480, 272 |
| `$f16`–`$f19` (u0, v0, u1, v1) | 0, 0, 512, 512 | 0, 0, 512, 512 |

Before the plain black quad it calls `sceGuColorMaterial(7)`, `sceGuColor(0xFF000000)` and `sceGuDisable` for states 10, 5, 9 and 4 (lighting, cull, texture, blend). A box behind a subtitle uses the same sequence with blending on (`sceGuEnable(4)`, `sceGuBlendFunc`) and a color with alpha below `0xFF`.

| Address | Function |
|---|---|
| `0xC4444` | `sceGuColorMaterial` |
| `0xC5504` | `sceGuColor` |
| `0xC4AF4` | `sceGuEnable` |
| `0xC4B4C` | `sceGuDisable` |
| `0xC5DE0` | `sceGuBlendFunc` |

## Style

Subtitles use white text with a black shadow, like film subtitles. The patch draws each line twice with `FsFontDrawText`: black at (x + 1, y + 1), then white at (x, y). A four-way black outline (offsets of one pixel left, right, up and down) is an option for bright backgrounds.

Subtitles do not cover the burned-in Japanese text. Each subtitle has its own position, in screen space that the Japanese block does not use ([0011](0011-movie-script.md) gives the bounds). A translucent box (see the `Gfx_FastSprite` notes above) is a last resort for a block with no clear space.

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
| Position: x, y in pixels | u16, u16 |
| Style: shadow, outline or box | u16 |

Text is Shift-JIS with ASCII. Frame numbers count calls of the patched draw function.

## Font calls in order

`FsFontDrawText` only queues glyphs. The patch uses the sequence of `CAdvMenuPrinter::Begin` and `End`:

1. `FsFontSetCurrent(font)` (`0x14C1C`)
2. `FsFontBegin()` (`0x168F4`)
3. `FsFontSetScale(1.0, 1.0)` (`0x150E8`, `$f12`, `$f13`)
4. `FsFontSetAlignment`, `FsFontSetBaseColor`, `FsFontDrawText` for each pass
5. `FsFontEnd()` (`0x16958`) draws the queued glyphs
6. `sceGuDisable` for states 10, 7 and 1

`FsFontBegin` writes through the draw-state pointer at `0x1415FC`. `FsFontInit` sets it when the first font loads (`CFont::Init`, `0xC044`). The patch skips the text while the pointer is zero.

## Counter results

Test: `kfa patch counter`, run headless for 300 emulated seconds.

- The first play of the opening at boot shows no counter: no font is loaded yet, so the guard skips the text. The game does not crash.
- After the title screen the opening replays (attract loop). The counter shows at (16, 16) in white with a black shadow.
- The counter rises by 600 in 20 seconds: one step per video frame at 29.97 fps. Subtitle times in seconds convert to frames by × 29.97.
- The counter does not reset between plays. It counted about 4,230 frames during the first play (141 s).

Remaining problems:

1. The first play at boot has no font. The subtitles need the font loaded before that play, or their own glyph drawing.
2. The counter must reset when a movie starts.

## Timing

The narration appears at known times in the video. [0011](0011-movie-script.md) lists the text and its approximate times. The counter rate is not known: the game loop and the 29.97 fps video can differ. A first test build draws the counter on screen so the rate can be read against the video.

## Test builds

| Build | Shows | State |
|---|---|---|
| Frame counter | Hook runs; counter rate | Done; see results |
| One subtitle | Font, shadow and position | Not built |
| Full table, movie000 | Timing against the narration | Not built |

## Tools

`internal/prx` reads and edits the relocation table. `internal/mips` encodes the instructions; its test compares the encodings with instructions in `BOOT.BIN`. `internal/subpatch` builds the patch and checks that the hooked word is the expected `jal` with an `R_MIPS_26` relocation. Its test checks that no relocation inside the code space stays active and that nothing outside the code space and the hook changes except the 61 relocation types.

```sh
./kfa patch counter research/extract/kfaI/PSP_GAME/SYSDIR/BOOT.BIN research/builds/kfaI-counter-BOOT.BIN
cp research/iso/kfaI.iso research/builds/kfaI-counter.iso
./kfa iso put research/builds/kfaI-counter.iso PSP_GAME/SYSDIR/EBOOT.BIN research/builds/kfaI-counter-BOOT.BIN
```

## Other discs

Additional II needs the same analysis: its `BOOT.BIN` differs, so every address above changes.
