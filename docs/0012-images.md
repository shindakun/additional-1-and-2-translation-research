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

The global header body also holds one GIM at `0x2AB30` (file offset): the HUD atlas, 512 × 256, RGBA 5551, swizzled. Its export name is `g_02aa00.gim` (offset in the header unit).

## GIM

`internal/gim` decodes GIM files: RGBA 5650, 5551, 4444 and 8888, and 4- and 8-bit indexed images with a palette, in normal or swizzled order. Rows are padded to 16 bytes; swizzled data uses 16-byte × 8-row tiles. The root block size gives the end of the file; other data or zero bytes can follow it.

All GIMs in `AdvMenu.bnd`, `AdvSys.bnd`, the header and the chunk image tables decode. The 16-bit formats expand each channel with `value * 255 / max` in integer arithmetic.

`gim.Encode` writes an edited image into a copy of the original GIM. The format, the swizzle and the palette stay, so the size does not change. Indexed formats use the nearest palette color. Row padding keeps the original bytes. Encoding a decoded disc GIM gives the original bytes for 219 of 230 files; the other 11 are indexed and decode to the same image.

## Tools

| Command | Action |
|---|---|
| `kfa img export GAMEDIR DIR` | Write each unique image once, a PNG of each GIM, and `index.tsv` (chunk, entry, type, size, SHA-1, file) |
| `kfa img import GAMEDIR DIR OUT.dat [BASE.dat]` | Write the files in `DIR` over every image with the same export name. For a GIM, `NAME.png` also works: it is encoded into the original GIM format |
| `kfa gim topng IN.gim OUT.png` | Convert one GIM |
| `kfa gim scan FILE DIR` | Write every GIM found inside a file, named by offset |
| `scripts/relabel_jpeg.py SPEC.json IN OUT` | Replace label text in a JPEG (within a byte budget) or a PNG (with alpha) |

An export name is `cCCC_EE.ext`: the chunk and entry of the first use. Import finds the images by content, so `BASE.dat` must hold the original images. The output of `kfa text import` is a valid base.

Import writes the new file in place, fills the rest of the old space with zero bytes and sets the size in the table. A new file must not be larger than the original. `relabel_jpeg.py` uses the highest JPEG quality that fits.

## Images with Japanese text

| Image | Text |
|---|---|
| `c000_19.jpg` (town map) | `トラウジトの町` |
| `c000_20.jpg` (town map with locations) | `トラウジトの町`, `街道`, `酒場`, `商店`, `宿屋`, `鍛冶屋`, `迷宮入口`, `ステータス` |
| `c009_04.jpg` (status screen, 16 uses) | `体力`, `魔力`, `素早さ`, `攻撃力`, `防御力`, equipment slots, damage types, `重量` |
| `c004_04.jpg` (roulette board) | `倍率表を見る` (payout table button) |
| `c004_06.jpg` (roulette payout table) | Bet types: `1個賭け`, `横1列賭け`, `偶数奇数、赤黒、前半後半` and others |
| Chunk 0 GIM `c000_00` (shop panels) | `修理費用`, `売却額`, `購入額`, `所持数`, `在庫数` |
| HUD atlas `g_02aa00` in the header body | `体力`, `魔力`, `素早さ` (status overlay), `必要`, `経験値`, `GOLD` |

`AdvMenu.bnd` and `AdvSys.bnd` have English only (`NEW GAME`, `Now loading`, the logo).

The menu buttons (`アイテム`, `魔法`, `キャンプ`, `消耗品`, `貴重品`) are font text: they are command names in the text export ([0004](0004-text.md)).

## Relabel options

`relabel_jpeg.py` finds the old text in each box by brightness (`threshold`), removes it and draws the new text with a 1-pixel shadow. Label fields: `box`, `text`, `size`, `color`, `threshold`, `align` (`center` or `left`) and `fill`:

| `fill` | Action |
|---|---|
| `inpaint` (default) | Fill the text pixels, grown by 1 pixel, from their neighbours. The background texture stays. |
| `flat` | Fill the text's bounding box with the median background color |
| `box` | Fill the whole box with its median color; for dark text on a light box. `shadow: false` turns off the shadow |
| `clear` | Make the box transparent (PNG on a transparent background) |

## Test result

Specs in `research/img/edit`:

| Spec | Image | Result |
|---|---|---|
| `town.json`, `town19.json` | Town maps `c000_20`, `c000_19` | Eight labels and the title; quality 86 and 85 |
| `status.json` | Status screen `c009_04` | Stats, equipment slots, attack and defence, damage types, load; quality 92 |
| `hud.json` | HUD atlas `g_02aa00` (PNG) | `Vit`, `Mag`, `Agi`, `Next`, `EXP` |
| `shop.json` | Shop panels `c000_00` (PNG) | `Repair cost`, `Sell price`, `Buy price`, `Owned`, `Stock` |
| `board.json`, `payout.json` | Roulette `c004_04`, `c004_06` | `Payout table`; bet types on the white boxes (`fill: box`) |

`scripts/build_english.sh` builds the image ([0009](0009-test-builds.md)). Headless PPSSPP shows the English town map, the status overlay (Cross in town), the shop price panel, the equipment screen and the roulette board button. The payout table image is not yet seen in a run. The HUD atlas also holds moved header strings in its unused rows ([0004](0004-text.md)); the HUD draws without change. Camp, Equipment shows the full status screen (`c009_04`) with the English labels.

## Not built

- Images larger than the original: they need chunk relayout ([0004](0004-text.md)).

`kfa text import` can re-encode JPEGs to free space for text ([0004](0004-text.md)). Run `kfa img import` first: it finds images by their original contents.
