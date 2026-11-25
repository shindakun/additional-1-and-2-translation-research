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

A second hook replaces the first `jal` in `CSimpleMovieScene::Exit` (`0x3EB58`, target `0xB33FC`). It calls the original function and sets the counter to 0, so every movie starts at frame 0. Headless runs confirm the reset.

The first play at boot has no font unless the patch loads one ([Font at boot](#font-at-boot)).

## Font at boot

Two functions create a movie scene (`FUN_0003E89C` with ID `0x4E3D`) for the opening:

| Function | Order |
|---|---|
| `FUN_00068820` (boot) | Creates the scene and plays `movie000` at once (`FUN_000689A4`, number 0). No font is loaded. |
| `FUN_0009FA3C` (title) | Creates the scene, loads `Bnd/AdvMenu.bnd` in a loop (`FUN_00025730`), takes `BitFont_Game12.bnd` from it (`FUN_00050670`), calls `CFontMng::SetFont` (`0x107FC`), then loads `Title.gim`, `FromLogo.gim` and `Thumb.gim`. |

Options for the first play:

| Option | Work |
|---|---|
| A. Load the font in the hook | On the first frame with no font, repeat the title steps: load `AdvMenu.bnd`, take `BitFont_Game12.bnd`, call `CFontMng::SetFont`. Needs a binder object and memory for it. |
| B. Own glyph drawing | Read prepared subtitle images from a file on the disc (`sceIo` imports exist), draw them with `sceGuTexImage` and `Gfx_FastSprite`. Independent of the boot order. |

First option A test (`subpatch.Counter` with `loadFont`): the draw hook loads `Font/BitFont_Game12.bnd` with `CFileSystem::AllocRead` (`0xACA8`, object `0x1529E8`) on the first movie frame and calls `CFontMng::SetFont`. The counter shows at boot, but after the movie the game shows its fatal-error screen (ゲーム実行中に致命的なエラーが発生しました, FMG 505013). The error screen opens when the error object at `+0x46C` is set; the `main.dat` chunk loader sets it, for example, when it cannot allocate its 0x500000-byte buffer.

Working option A (`kfa patch subtitles --boot-font`): a fourth hook replaces the `jal` at `0x688C4` in the boot function (`FUN_00068820`, target `0x689A4`, which starts `movie000`). It loads the font the same way before the movie starts, then calls the original target. `SetFont` hands the buffer to the slot 0 binder with the own flag, so the exit hook frees it: after `movie000`, it calls `CFontMng::DelFont` and `CBinder::Delete` on the binder at `CFontMng + 0x401C0`, once. The title then loads its own font as before. Headless runs from boot: `movie000` shows subtitles, the title follows, New Game plays `movie001` with subtitles, and the town map loads at 320 s. No fatal error.

## Timing

The counter counts video frames at 29.97 per second, from 0 at the start of each movie. In a headless run the counter and the emulated time agree: frame = 30 × (emulated seconds − movie start). [0011](0011-movie-script.md) lists the narration times, measured with the counter build.

## Subtitle build

`kfa patch subtitles BOOT.BIN OUT MOVIE=IN.tsv...` builds the patch from one table for each movie. `MOVIE` is a movie file name (`movie001.pmf`) or `all`. Each TSV line is one screen line:

| Column | Content |
|---|---|
| `START`, `END` | Seconds from the first frame of the movie |
| `ROW` | 0 is the bottom row (y 244), rows count up; -1 is the top row (y 6), negative rows count down. Rows are 17 pixels apart |
| `ALIGN` | `l`, `c` or `r` |
| `X` | The pixel column that the alignment refers to |
| `TEXT` | ASCII, no `%` |

The patch has three entries, four with `--boot-font`:

| Entry | Hook | Action |
|---|---|---|
| Boot (`--boot-font`) | `jal` at `0x688C4` in `FUN_00068820` | Loads the game font, then starts `movie000` |
| Draw | `jal` at `0x3EB30` in `CSimpleMovieScene::Draw` | Calls the movie draw, reads the clock, draws the active lines |
| Exit | `jal` at `0x3EB58` in `CSimpleMovieScene::Exit` | Calls the original function, clears the clock, removes the boot font once |
| File | `jal` at `0x3E998` in `CSimpleMovieScene::SetMovieFile` (target `0xAFCC`) | Stores the movie key of the path in `a2`, then jumps to the original target |

The movie key is the three characters before the last `.` of the path: `movie001.pmf` gives `001`. The clock: on the first draw of a movie the patch stores the low word of `sceKernelGetSystemTimeWide` (import stub `0x11C444`); every draw computes the elapsed microseconds. A draw counter falls behind the video when the emulator skips frames; system time follows the emulated clock and the movie audio.

The draw entry walks a table of 9-word records: start and end in microseconds, x, y, shadow x, shadow y, text offset, alignment, movie key. A record draws if its range holds the counter and its key is 0 or the current key. It sets the alignment and draws the black shadow one pixel down and right, then the white text. The code uses 352 words; the font path, the table and the text follow it in the same code space. Equal texts share one copy. The code space (6,552 bytes) holds about 70 records with their texts.

To read the current key on screen, point a line's text offset at the key word: the bytes are the three characters in reverse order, then a zero.

`FsFontSetAlignment` (`0x15168`) values for the horizontal position, from a headless test with four lines at x 240:

| Value | Position |
|---|---|
| `0x00` | Text starts at x |
| `0x10` | Text ends at x |
| `0x20` | Text is centered on x |
| `0x30` | Same as `0x00` |

The ASCII glyphs of the game font are about 8 pixels wide: a 60-character line is 480 pixels, the full screen. A line is about 15 pixels tall with descenders: at 14-pixel rows the lines overlap and look doubled, and a row at y 256 loses its descenders at the screen edge.

## Subtitle results

`research/subs/kfaI-movie001.tsv` holds 33 lines for the 11 narration blocks of `movie001`. `movie000` is the same video 5.0 s later (the logo part): frame matching at 2 frames per second gives the best fit at 5.0 s. `research/subs/kfaI-movie000.tsv` is the `movie001` table with 5 s added. Each block sits clear of its Japanese text: at the bottom when the Japanese is high, at the top when the Japanese is low. Blocks 7 and 8 share the screen: block 7 at the left of the middle band, block 8 in three short lines at the upper right. Each block starts 0.3 s before the Japanese is fully visible; a 10 fps measurement of the video shows the Japanese fade-in starts 0.5 to 1 s after those times.

The New Game opening plays `movie001` (the key on screen reads `001`), after the title, with the font loaded. `movie000` is the same narration after the FromSoftware logo; it plays at boot. Which movie the attract loop plays is not yet checked. Headless runs show every block in English at the planned position, and the game continues to the town map after the movie.

Limits:

- `movie010` and `movie011` have no tables yet.
- Which movie the attract loop plays is not yet checked.

## Test builds

| Build | Shows | State |
|---|---|---|
| Frame counter | Hook runs; counter rate | Done; see results |
| Alignment test | `FsFontSetAlignment` values | Done |
| Full table, movie001 | Timing and position against the narration | Done; see results |
| Tables for each movie | Correct lines for each movie | Built; `movie001` table done |

## Tools

`internal/prx` reads and edits the relocation table. `internal/mips` encodes the instructions; its test compares the encodings with instructions in `BOOT.BIN`. `internal/subpatch` builds the patch and checks that the hooked word is the expected `jal` with an `R_MIPS_26` relocation. Its test checks that no relocation inside the code space stays active and that nothing outside the code space and the hook changes except the 61 relocation types.

```sh
./kfa patch subtitles --boot-font research/extract/kfaI/PSP_GAME/SYSDIR/BOOT.BIN OUT-BOOT.BIN \
  movie000.pmf=research/subs/kfaI-movie000.tsv movie001.pmf=research/subs/kfaI-movie001.tsv
./kfa iso put IMAGE.iso PSP_GAME/SYSDIR/EBOOT.BIN OUT-BOOT.BIN
```

`kfa patch counter` builds the counter test the same way.

## Other discs

Additional II needs the same analysis: its `BOOT.BIN` differs, so every address above changes.
