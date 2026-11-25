# 0001: Initial findings, Additional II

King's Field Additional II, ULJS-00076. The formats are the same as in Additional I; [docs/kfa1](../kfa1/0001-initial-findings.md) describes them. This document lists what differs and what the Additional I tools do with this disc.

## Disc

| Item | Value |
|---|---|
| Image size | 822,738,944 bytes |
| Files | 435 |
| `BOOT.BIN` | 1,615,652 bytes (Additional I: 1,615,408) |
| `main.dat` | 208,187,392 bytes, 139 units, 98 chunks |
| Movies | 7: `movie000`, `movie201`–`movie203`, `movie210`–`movie212` |
| Fonts | 21 `BitFont_GameNN.bnd` and `Debug.bnd`, as in Additional I |

The layout of `main.dat` is in [kfa1/0003](../kfa1/0003-maindat.md). The keys are the same. `kfa dat verify` passes: MD5 correct, every unit re-encrypts to the original bytes.

## Tool results

| Tool | Result |
|---|---|
| `kfa text export` | 4,446 entries from 67,369 slots: 2,488 `text`, 1,293 `flag`, 560 `name`, 96 `block`, 9 `menu` |
| `kfa fmg export` | 316 messages; 311 are the same as in Additional I |
| `kfa img export` | 1,499 images, 475 unique |
| `kfa font pad` | Pads `BitFont_Game12.bnd` without errors |
| `kfa patch subtitles`, `kfa patch chunkbase` | Apply; see [BOOT.BIN](#bootbin) |

## Text

2,553 entries have the same kind and Japanese as an Additional I entry: 1,084 `text`, 903 `flag`, 500 `name`, 57 `block`, 9 `menu`. A translation file can take their English from Additional I (`scripts/text_merge.py`, [kfa1/0004](../kfa1/0004-text.md)).

The rest, 1,503 `text`, `name`, `menu` and `block` entries, are new: about 34,500 characters of Japanese.

The 5 FMG messages that differ name the game: `ＫＦＡ` becomes `ＫＦＡⅡ` in 505013–505016 and 505120.

## Places

The block names show the areas ([kfa1/0004](../kfa1/0004-text.md#block-header)). Shared with Additional I: the town (`トラウジトの町`), the casino games, the shop, the smithy, `苔生した遺構`, `落日の宮廷`, `地下水路`, `玉座の間`, `精霊の神殿`, `古の祭壇`. New: `塞閉せる小径`, `憑呪の歩廊`, `狂騒の魔窟`, `遮光の荒地`, `追懐の城`, `鎖錠の険路`, `賢者の深奥`, `流転の霊廟`, and two casino games, `神経衰弱` (Memory) and `スロット` (Slots).

## Images

91 of the 475 unique images have the same bytes as an Additional I image. None of the seven images edited for Additional I ([kfa1/0012](../kfa1/0012-images.md)) is among them: the town maps, the status screen, the roulette images, the shop panels and the HUD atlas need their own edits. The HUD atlas is not yet located.

## BOOT.BIN

The patches identify `BOOT.BIN` by a code range; Additional II has its own entry in `subpatch`. `scripts/map_boot.py FROM.BIN TO.BIN ADDR...` maps each Additional I address:

- Code: the same 16 words around the address, with `jal` targets and 16-bit immediates masked, must match once.
- Data in segment 0 (`FontPtr`): the `lui` pairs that build the address; 37 sites agree.
- Data in segment 1: instructions hold offsets from the segment base, `0x13D4A0` in Additional I and `0x13D6A0` in Additional II. All matching instructions hold the same offsets (3 for `DrawState`, 64 for `FileSystem`).
- The `sceKernelGetSystemTimeWide` stub: import stubs have the same code, so the address comes from the `jal` at a matching call site.

| Address | Additional I | Additional II |
|---|---|---|
| Movie draw call, target | `0x3EB30`, `0xB35D0` | `0x3EBE8`, `0xB3668` |
| Movie exit call, target | `0x3EB58`, `0xB33FC` | `0x3EC10`, `0xB3490` |
| Movie file call, target | `0x3E998`, `0xAFCC` | `0x3EA50`, `0xB084` |
| Boot movie call, target | `0x688C4`, `0x689A4` | `0x68954`, `0x68A34` |
| Chunk table load, base store | `0x59A70`, `0x59BC8` | `0x59B44`, `0x59C9C` |
| Code spaces | `0xF232C` (6,552), `0xC877C` (1,712) | `0xF252C` (6,552), `0xC897C` (1,712) |
| Font: set current, color, align, draw, scale | `0x14C1C`, `0x14EA0`, `0x15168`, `0x15288`, `0x150E8` | `0x14CD4`, `0x14F58`, `0x15220`, `0x15340`, `0x151A0` |
| Font: begin, end, set, delete | `0x168F4`, `0x16958`, `0x107FC`, `0x10910` | `0x169AC`, `0x16A10`, `0x108B4`, `0x109C8` |
| `sceGuDisable`, `sprintf`, `AllocRead`, `CBinder::Delete` | `0xC4B4C`, `0xFE6F0`, `0xACA8`, `0x257C0` | `0xC4D4C`, `0xFE8F0`, `0xAD60`, `0x25878` |
| Font pointer, draw state, file system | `0x1262F4`, `0x1415FC`, `0x1529E8` | `0x1264F4`, `0x1417FC`, `0x152BE8` |
| System time stub | `0x11C444` | `0x11C64C` |

No relocation reaches either code space. Both patches accept the file: each checks the expected instruction and relocation at its hooks.

Test: `kfa patch subtitles --boot-font` with one line on `movie000`, then `kfa patch chunkbase`. Headless PPSSPP shows the line over the boot movie, then the title and the town map. No fatal error.

The title menu has a third item, `DATA CONVERT`. Additional I offers to save clear data for Additional II at its end (global text 3203, [kfa1/0004](../kfa1/0004-text.md)); the link between the two is not yet checked.

## Next steps

1. Merge the Additional I English into an Additional II translation file; translate the new entries.
2. Locate the image text: town map, status screen, casino boards, HUD.
3. Transcribe the seven movies ([kfa1/0011](../kfa1/0011-movie-script.md)).
4. A build script for Additional II and a headless test.
