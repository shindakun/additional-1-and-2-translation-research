# 0001: Initial findings

## Discs

| | Additional I | Additional II |
|---|---|---|
| Archive | `King's Field - Additional I (Japan, Asia).zip` | `King's Field - Additional II (Asia).zip` |
| Archive SHA-1 | `ccebf9d21cadc49a7fee2680e91de446d3fc73fe` | `b4619893f41df95c6886fc9e38d031f5b9c2b4a8` |
| Archive SHA-256 | `25ad3b561411a031e6351524cad4ae52a7407bcdc3cb29a950bfe29f9043434f` | `c570620ec1d76e89a681ac94fc36a814df5be68616cf4b5778dd8b73e08a9f9b` |
| ISO size | 809,369,600 | 822,738,944 |
| ISO SHA-1 | `0955d3e00ee2f89862e546492750f7fdf9503fae` | `1c4c950bbb0d3b89aef5cb498568e158b1106a9b` |
| ISO SHA-256 | `f4e785cc22a199e1dccabce2f083bf3105ca9a54e4abcccfbced86ce02d4c055` | `ad794f2e3aed5c8b30d342576e748ec2e228d3dc13c8069c89acd5415ffccb68` |
| `BOOT.BIN` SHA-256 | `13c9d059322e23f23e9b49c687d5f27e6f3c138147cfbf7c5278a22bd05ef783` | `a308614afdb16374834bb642198bb8fa6793952c1c81a23ce7ac9c0d029a877f` |
| `EBOOT.BIN` SHA-256 | `b88a11734981b68ad628c4cf6e27e2348258d522a9126cac8f160a562a2c3653` | `2ca76232ea8da9305e95950704a18cb6f9c93ff5d4a7c460fe6332478b453d50` |
| `main.dat` SHA-256 | `c23144ffe853c56b246cab218cae2305ead35f0f2cfaee501c7fe8164e3f1cfe` | `a9c8d3774ffd2cda07f662e0aa8bf2ebe9b03f2b803b6b27c13ce4216f9cf7d2` |
| Disc ID | ULJS-00070 | ULJS-00076 |
| Disc version | 1.02 | 1.01 |
| `PSP_SYSTEM_VER` | 2.60 | 2.71 |
| Engine build | `Adventure Player  22:10:33,May 29 2006` | `Adventure Player  14:39:25,Jul 12 2006` |
| Files | 432 | 435 |

Each archive holds one UMD ISO9660 image.

Both games use the FromSoftware "Adventure Player" engine. The engine also ran the AppliBox platform (save path `ULJS00050_AppliBox`). Both discs keep AppliBox data.

## Reproduce

| Script | Output |
|---|---|
| `scripts/extract_iso.sh` | ISOs in `research/iso`, file trees in `research/extract` |
| `scripts/bnd_list.py` | BND entries, nested |
| `scripts/fmg_dump.py` | FMG strings |
| `scripts/gameinfo_dump.py` | `GameInfo.dat` records |
| `scripts/boot_info.py` | `BOOT.BIN` sections, AES Te0 offset |
| `scripts/maindat_survey.py` | `main.dat` entropy, segments, magic counts |

Run Python scripts with `python3 -I`. The README lists all scripts.

## Disc layout

Sizes are totals in bytes.

| Path | Additional I | Additional II | Contents |
|---|---|---|---|
| `PSP_GAME/SYSDIR/EBOOT.BIN` | 1,615,408 | 1,616,000 | Encrypted executable (`~PSP`) |
| `PSP_GAME/SYSDIR/BOOT.BIN` | 1,615,060 | 1,615,652 | Plain ELF, same program |
| `PSP_GAME/SYSDIR/UPDATE/` | 18,761,344 | 20,117,024 | Firmware updater |
| `USRDIR/GameData/01_UmdTitle/main.dat` | 237,586,432 | 208,187,392 | Game data pack |
| `USRDIR/Bnd/*.bnd` (2) | 2,712,507 | 2,313,147 | Menu and system resources |
| `USRDIR/Font/*.bnd` (22) | 15,944,512 | same | Bitmap fonts |
| `USRDIR/Layout/Cursor/*.icn` (80) | 1,420,860 | same | Cursor icons |
| `USRDIR/Layout/SkinData/9000.SKN` | 271,088 | 271,088, other content | UI skin |
| `USRDIR/movie/*.pmf` | 4 files, 156,719,104 | 7 files, 199,768,064 | PSMF video |
| `USRDIR/sound/BGM/*.at3` (279) | 121,888,468 | same | ATRAC3plus music |
| `USRDIR/boot/prx250`, `prx260` | 15 + 15 modules | same | Sony library PRX |
| `USRDIR/Dmy/DUMMY*.BIN` (2) | 220,200,960 | same | All zero, filler |

"Same" means byte-identical.

## Formats

| Format | Magic | Status | Notes |
|---|---|---|---|
| BND, named | `BND\0`, version `0xFFFF`, flag `0xD3` | Parsed | Container |
| BND, unnamed | `BND\0`, version `0x633` | Parsed | Container in `main.dat`, IDs only |
| FMG | `00 00 01 00` | Parsed | Message table, Shift-JIS |
| GIM | `MIG.00.1PSP` | Known | PSP texture |
| HVG | `48 56 47 20 20` (`HVG` and two spaces) | Header seen | Holds a Sony sound bank (`IECSsreV`, `IECSdaeH`, `IECSigaV`, `IECSlpmS`) |
| LMN | `YGF_LMN` | Not studied | Layout |
| MFX | `YGF_MFX` | Not studied | Effects |
| CCM | `00 00 01 00` + size | Not studied | Font glyph map |
| PLT | | Not studied | Font palette, 0x4000 bytes |
| SKN | `SKIN_DAT` | Not studied | UI skin |
| ICN | holds a GIM | Not studied | Cursor icon, source `icon_NN.tga` |
| AT3 | `RIFF`/`WAVE`, format `0xFFFE` | Known | ATRAC3plus |
| PMF | `PSMF0014` | Known | PSP movie |
| `jis2ucs.bin`, `ucs2jis.bin` | | Known | JIS and UCS-2 tables, 0x20000 bytes each |

### BND, named

Little-endian. Header is 0x20 bytes. Entry data aligns to 0x800.

| Offset | Size | Field |
|---|---|---|
| 0x00 | 4 | `BND\0` |
| 0x04 | 4 | `0x0000FFFF` |
| 0x08 | 4 | `0xD3` |
| 0x0C | 4 | File size |
| 0x10 | 4 | Entry count |
| 0x14 | 12 | `00000000 00080100 00000000` in all files |
| 0x20 | 16 × n | Entries |

Entry: ID, data offset, data size, name offset (u32 each). Names are NUL-terminated ASCII after the entry table. BNDs nest.

### BND, unnamed

Same layout. Version is `0x633`, flag is 0, name offset is 0. The ID gives the file name. Example: ID 4 in a sound BND is `sss00004.hvg`.

### FMG

Little-endian.

| Offset | Size | Field |
|---|---|---|
| 0x00 | 4 | `00 00 01 00` |
| 0x04 | 4 | File size |
| 0x0C | 4 | Group count |
| 0x10 | 4 | String count |
| 0x14 | 4 | String offset table position |
| 0x1C | 12 × groups | First string index, first ID, last ID |

A string offset points to NUL-terminated Shift-JIS text. Offset 0 means no string.

Markup:

| Token | Meaning |
|---|---|
| `#c[RRGGBB]` | Set color |
| `#c` | Reset color |
| `#m[n]` | Icon `n` |
| `#b` | Bold on or off |
| `\n` | Line break |

Example, `AdvCommon.fmg` ID 300000:

```text
#c[FFFF00]#m[38] #b警告#b
#cメモリースティック#m[82]が挿入されていません。
```

The executable also refers to `$(AppCommonRoot)\msg\menu.fmg`.

## Containers

### `Bnd/AdvMenu.bnd`

| ID | Name | Type |
|---|---|---|
| 0 | `0129.SKN` | Skin |
| 1 | `BitFont_Game12.bnd` | Font BND |
| 2 | `Title.gim` | Title screen |
| 3 | `FromLogo.gim` | Logo |
| 4 | `Thumb.gim` | Thumbnail |
| 5 | `AdvMenuSe.bnd` | Menu sound effects (HVG) |
| 6 | `GameInfo.dat` | AppliBox catalog |

### `Bnd/AdvSys.bnd`

Additional I has 19 entries. Additional II has 14. Contents: JIS tables, `MfxData.mfx`, `Sfx00.gim`, `sk_sys.gim`, `Layout.lmn`, `AdvCommon.fmg`, save icons, `ICON0.png`, `PIC1.png`, `applibox.png`, three Japanese developer `readme.txt` files. Additional I has a second save icon set (`bk_saveicon_*`).

### `GameInfo.dat`

Three AppliBox records of 0x200 bytes: `UmdGame01.adv` (ハコノナカ), `UmdGame02.adv` (Echo Night #1), `UmdGame03.adv` (怪奇圏). BMP thumbnails follow. Both discs have the same file. It does not describe King's Field. It shows that the engine's game unit is an `.adv` file.

### Font BNDs

`BitFont_GameNN.bnd` holds `Ccm.ccm`, `Plt00.plt`, `Plt01.plt`, `Tex00.gim`, `Tex01.gim`. `Debug.bnd` holds one CCM and one GIM. The CCM maps characters to glyph cells.

## `main.dat`

`main.dat` is an encrypted `ADVP` container. [0003](0003-maindat.md) describes the format and keys.

## Executable

`BOOT.BIN` is a plain, relocatable PSP ELF (PRX). It has section headers and no symbol table. For `.text`, `.rodata` and `.data`, address = file offset − 0x80.

| Section | Address | Size |
|---|---|---|
| `.text` | `0x000000` | `0x11BF2C` |
| `.rodata` | `0x11CD90` | `0x8F50` |
| `.data` | `0x125CE0` | `0x17737` |
| `.bss` | `0x13F380` | `0x17ED0` |

The string `$(GameDataRoot)main.dat` is present. `GameDataRoot` is `$(AppDataRoot)GameData/01_UmdTitle/`.

`BOOT.BIN` differs between the discs, import stubs included.

Ghidra 12.2 with the `ghidra-allegrex` extension loads `BOOT.BIN` with the "PSP Executable (ELF)" loader and language `Allegrex:LE:32:default`.

[0003](0003-maindat.md) lists the `main.dat` functions.
