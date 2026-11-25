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
| `kfa patch subtitles`, `kfa patch chunkbase` | Refused: `BOOT.BIN` is not known ([0002](0002-boot-addresses.md)) |

## Text

2,553 entries have the same kind and Japanese as an Additional I entry: 1,084 `text`, 903 `flag`, 500 `name`, 57 `block`, 9 `menu`. A translation file can take their English from Additional I (`scripts/text_merge.py`, [kfa1/0004](../kfa1/0004-text.md)).

The rest, 1,503 `text`, `name`, `menu` and `block` entries, are new: about 34,500 characters of Japanese.

The 5 FMG messages that differ name the game: `ＫＦＡ` becomes `ＫＦＡⅡ` in 505013–505016 and 505120.

## Places

The block names show the areas ([kfa1/0004](../kfa1/0004-text.md#block-header)). Shared with Additional I: the town (`トラウジトの町`), the casino games, the shop, the smithy, `苔生した遺構`, `落日の宮廷`, `地下水路`, `玉座の間`, `精霊の神殿`, `古の祭壇`. New: `塞閉せる小径`, `憑呪の歩廊`, `狂騒の魔窟`, `遮光の荒地`, `追懐の城`, `鎖錠の険路`, `賢者の深奥`, `流転の霊廟`, and two casino games, `神経衰弱` (Memory) and `スロット` (Slots).

## Images

91 of the 475 unique images have the same bytes as an Additional I image. None of the seven images edited for Additional I ([kfa1/0012](../kfa1/0012-images.md)) is among them. [0003](0003-images.md) finds their counterparts.

## Title menu

The title menu has a third item, `DATA CONVERT`. Additional I offers to save clear data for Additional II at its end (global text 3203, [kfa1/0004](../kfa1/0004-text.md)); the link between the two is not yet checked.

## Later documents

| Document | Subject |
|---|---|
| [0002](0002-boot-addresses.md) | `BOOT.BIN` addresses for the patches |
| [0003](0003-images.md) | Image text |
| [0004](0004-translation.md) | Translation and builds |
