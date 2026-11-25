# 0012: Images

Some screen text is part of an image. Translation needs those images edited.

## Image table

The block header `+0x18` points to an image table of 16-byte entries ([0004](0004-text.md)). The value `0xFFFFFFFF` means no table. The number of entries is the u16 at block header `+0x0A` (GIMs) plus the u16 at `+0x0E` (JPEGs and BMPs).

| Offset | Field |
|---|---|
| `0x0` | Type: 0 = JPEG, 2 = GIM, 3 = BMP |
| `0x4` | Size in bytes |
| `0x8` | File offset in `main.dat` |
| `0xC` | Width (u16), height (u16) |

| Game | JPEG | GIM | BMP | Unique files |
|---|---|---|---|---|
| Additional I | 629 | 1,259 | 16 | 442 |
| Additional II | 571 | 911 | 16 | Not counted |

Many chunks hold copies of the same image. JPEGs are 480 × 272 scene backgrounds. GIMs are mostly textures for monsters, people and dungeon walls. BMPs are 144 × 80.

The global header body also holds one GIM at `0x2AB30` (file offset): the HUD atlas, 512 × 256, RGBA 5551, swizzled.

## GIM

`internal/gim` decodes GIM files: RGBA 5650, 5551, 4444 and 8888, and 4- and 8-bit indexed images with a palette, in normal or swizzled order. Rows are padded to 16 bytes; swizzled data uses 16-byte × 8-row tiles. The root block size gives the end of the file; other data or zero bytes can follow it.

All GIMs in `AdvMenu.bnd`, `AdvSys.bnd` and the chunk image tables decode.

## Tools

| Command | Action |
|---|---|
| `kfa img export GAMEDIR DIR` | Write each unique image once, a PNG of each GIM, and `index.tsv` (chunk, entry, type, size, SHA-1, file) |
| `kfa img import GAMEDIR DIR OUT.dat [BASE.dat]` | Write the files in `DIR` over every image with the same export name |
| `kfa gim topng IN.gim OUT.png` | Convert one GIM |
| `kfa gim scan FILE DIR` | Write every GIM found inside a file, named by offset |
| `scripts/relabel_jpeg.py SPEC.json IN.jpg OUT.jpg` | Replace label text in a JPEG and fit a byte budget |

An export name is `cCCC_EE.ext`: the chunk and entry of the first use. Import finds the images by content, so `BASE.dat` must hold the original images. The output of `kfa text import` is a valid base.

Import writes the new file in place, fills the rest of the old space with zero bytes and sets the size in the table. A new file must not be larger than the original. `relabel_jpeg.py` uses the highest JPEG quality that fits.

## Images with Japanese text

| Image | Text |
|---|---|
| `c000_19.jpg` (town map) | `トラウジトの町` |
| `c000_20.jpg` (town map with locations) | `トラウジトの町`, `街道`, `酒場`, `商店`, `宿屋`, `鍛冶屋`, `迷宮入口`, `ステータス` |
| `c009_04.jpg` (status screen, 16 uses) | `体力`, `魔力`, `素早さ`, `攻撃力`, `防御力`, equipment slots, damage types, `重量` |
| `c011_06.jpg` and the other roulette boards | Bet labels and rules |
| Chunk 0 GIM `c000_00` (shop panels) | `修理費用`, `売却額`, `購入額`, `所持数`, `在庫数` |
| HUD atlas in the header body | `必要`, `経験値`, `GOLD` |

`AdvMenu.bnd` and `AdvSys.bnd` have English only (`NEW GAME`, `Now loading`, the logo).

The menu buttons (`アイテム`, `魔法`, `キャンプ`, `消耗品`, `貴重品`) are font text: they are command names in the text export ([0004](0004-text.md)).

## Test result

`town.json` in `research/img/edit` places English labels in the eight boxes of `c000_20.jpg` and the title box of `c000_19.jpg`. The new JPEGs are quality 87 (50,081 of 51,330 bytes) and quality 85 (47,446 of 48,124 bytes). `scripts/build_text_test.sh kfaI research/text/kfaI-test.json research/img/edit/kfaI` builds the image. Headless PPSSPP shows the English town map at 50 s.

## Not built

- GIM write-back: a new GIM must keep the format, the swizzle and the palette size.
- Writing back the HUD atlas in the header body.
- Images larger than the original: they need chunk relayout ([0004](0004-text.md)).

`kfa text import` can re-encode JPEGs to free space for text ([0004](0004-text.md)). Run `kfa img import` first: it finds images by their original contents.
