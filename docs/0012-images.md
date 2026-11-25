# 0012: Images

Some screen text is part of an image. Translation needs those images edited.

## Image table

The block header `+0x18` points to an image table of 16-byte entries ([0004](0004-text.md)):

| Offset | Field |
|---|---|
| `0x0` | Type: 0 = JPEG, 2 = GIM |
| `0x4` | Size in bytes |
| `0x8` | File offset in `main.dat` |
| `0xC` | Width (u16), height (u16) |

Chunk 0 of Additional I: 33 JPEGs (480 × 272 scene backgrounds) and 3 GIMs (256 × 512 and 256 × 256 texture atlases).

## GIM

`internal/gim` decodes GIM files: RGBA 5650, 5551, 4444 and 8888, and 4- and 8-bit indexed images with a palette, in normal or swizzled order. Rows are padded to 16 bytes; swizzled data uses 16-byte × 8-row tiles. Zero bytes can follow the last block.

| Command | Action |
|---|---|
| `kfa gim topng IN.gim OUT.png` | Convert one GIM |
| `kfa gim scan FILE DIR` | Write every GIM found inside a file, named by offset |

All GIMs in `AdvMenu.bnd` and `AdvSys.bnd` decode.

## Images with Japanese text

| Image | Location | Text |
|---|---|---|
| Shop panels | Chunk 0 GIM at `0x53BE0` | `修理費用`, `売却額`, `購入額`, `所持数`, `在庫数` |
| Title logo and menu | `AdvMenu.bnd` `Title.gim` | `NEW GAME`, `LOAD GAME` (English) |
| Town map labels | Not yet located | `トラウジトの町`, `街道`, `酒場`, `商店`, `宿屋`, `鍛冶屋`, `迷宮入口` |
| Status panel labels | Not yet located | `体力`, `魔力`, `素早さ`, `経験値` |
| Menu buttons | Not yet located | `アイテム`, `魔法`, `キャンプ`, `消耗品`, `貴重品` |

The flag records hold some of the same words (`体力`, `キャンプ汎用`), but flag names show only on debug screens ([0004](0004-text.md)).

## Writing images back

Not built. A GIM keeps its format and palette; a new image must use the same palette or replace it. A JPEG in `main.dat` must fit its original size until chunks can be relaid ([0004](0004-text.md)).
