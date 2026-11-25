# 0007: Adventure Player reference material

Two reference sources come from the same engine family. Neither ships in this repository.

| Source | Contents |
|---|---|
| `Adventure Player (Japan).zip` | Adventure Player, ULJS-00011, disc version 1.05. `BOOT.BIN` build `11:22:36,May 27 2005`. |
| `ADVPStudio101.exe` | Adventure Player Studio 1.01 for Windows, the authoring tool (`PspAdvTool`, July 2005). |

## Adventure Player disc

`PSP_GAME/SYSDIR/BOOT.BIN` keeps its symbol table. Ghidra names 4,873 of 4,886 functions. The names use the SN/EDG mangling scheme and encode argument types, for example `CAP_ActionFunc::PutText(ADVP_ACTION_PUTTEXT_ST)`. The DWARF sections cover only the C library.

`GameData/01_UmdTitle` holds three `.adv` files: `UmdGame01.adv` (ハコノナカ), `UmdGame02.adv` (Echo Night #1), `UmdGame03.adv` (怪奇圏). They use the `ADVP` container of [0003](0003-maindat.md) with the same keys. Their body has no chunk table at `0x14C`.

Key table addresses in this `BOOT.BIN`: encrypted table `0x123944`, table key `0x123A10`.

## Name transfer

`scripts/match_functions.py` carries names to the King's Field executables. It masks immediates and jump targets, matches unique instruction fingerprints, then follows call sites of matched pairs.

| Target | Functions | Exact | After call propagation |
|---|---|---|---|
| Additional I | 4,865 | 1,793 | 2,661 |
| Additional II | 4,867 | 1,788 | 2,660 |

In Additional I, 2,341 matched pairs have equal size, 238 differ by less than 25 %, and 82 differ more. Treat names as hints.

Changed functions need fuzzy matching. Additional I results:

| Address | Adventure Player function | Similarity |
|---|---|---|
| `0x56450` | `CAP_GameFileMng::Decryption` | 0.90 |
| `0x5805C` | `CAP_GameFileMng::RemapTextDataPtr` | 0.97 |
| `0x58448` | `CAP_GameFileMng::RemapCmdDataPtr` | 0.98 |
| `0x58594` | `CAP_GameFileMng::RemapEventDataPtr` | 0.96 |
| `0x587C8` | `CAP_GameFileMng::RemapBlockListDataPtr` | 0.98 |
| `0x58D8C` | `CAP_GameFileMng::RemapEditEreaDataPtr` | 0.94 |
| `0x591B4` | `CAP_GameFileMng::ReloadCall` | call propagation |

## File structure names

`CAP_GameFileMng` has one remap function per structure. The argument types name the structures:

| Function | Structures |
|---|---|
| `RemapTextDataPtr` | `ADVP_TEXTDATA_HEADER_ST` |
| `RemapFlagDataPtr` | `ADVP_FLAGDATA_ST` |
| `RemapCmdDataPtr` | `ADVP_CMDDATA_ST` |
| `RemapEventDataPtr` | `ADVP_EVENTDATA_HEADER_ST`, `ADVP_EVENTOFFSETTABLE_ST`, `ADVP_EVENTDATA_ST`, `ADVP_ACTIONDATA_ST` |
| `RemapImageDataPtr` | `ADVP_IMAGEDATA_HEADER_ST` |
| `RemapCmdTreeDataPtr` | `ADVP_CMDTABHEADER_ST`, `ADVP_CMDFOLDERTREEHEADER_ST`, `ADVP_CMDFOLDERTREENODE_ST`, `ADVP_CMDFONDER_ST` |
| `RemapBlockListDataPtr` | `ADVP_BLOCKLIST_HEADER_ST`, `ADVP_BLOCKLOADINFOTABLE_ST` |
| `RemapCursorDataPtr` | `C_ICON_ST` |
| `RemapSoundDataPtr` | `ADVP_OPTIONBGM_HEADER_ST`, `ADVP_OPTIONBGM_INFO_ST` |
| `RemapEditEreaDataPtr` | `ADVP_EDITEREA_HEADER_ST`, `_GROUP_ST`, `_IMAGEGROUP_HEADER_ST`, `_RECTLISTINFO_ST`, `_TEXTGROUP_HEADER_ST`, `_TEXTIDLISTINFO_ST` |

The chunk table of [0003](0003-maindat.md) is the block list. A chunk is a block. Chunk objects are events, and command lists are action data.

## Actions

`CAP_ActionFunc::CallAction` dispatches on the first byte of an action. King's Field keeps the same numbers. Its handlers match the Adventure Player handlers at the same opcode: `0x1E` 0.79, `0x3C`–`0x42` 0.55–0.66, `0x4B` 0.51, `0x5D` 0.71, `0x8E`–`0x8F` 0.57–0.60.

| Opcode | Adventure Player handler | Struct |
|---|---|---|
| `0x00`–`0x02` | `LoadGame`, `SaveGame`, `QuitGame` | `ADVP_ACTIONDATA_ST` |
| `0x0A` | `ChangeScene` | `ADVP_ACTION_GOTOSCENE_ST` |
| `0x0B` | `CallScene` | `ADVP_ACTION_GOTOSCENE_ST` |
| `0x0C` | `ReturnScene` | `ADVP_ACTION_RETURNSCENE_ST` |
| `0x14`–`0x17`, `0x1E` | `SetFlag` | `ADVP_ACTION_SETFLAG_ST` |
| `0x1F`–`0x22`, `0x28` | `PutText` | `ADVP_ACTION_PUTTEXT_ST` |
| `0x29` | `PutTextToWnd` | `ADVP_ACTION_PUTTEXT_ST` |
| `0x2A` | `ClearText` | `ADVP_ACTION_CLEARTEXT_ST` |
| `0x32`, `0x33` | `AddCmd`, `DelCmd` | `ADVP_ACTION_ADDCOMMAND_ST` |
| `0x3C`–`0x43` | `AddFlag`, `SubFlag`, `MulFlag`, `DivFlag`, `ModFlag`, `AndFlag`, `OrAddFlag`, `XorAddFlag` | `ADVP_ACTION_CALCADD_ST` |
| `0x44` | `NotAddFlag` | `ADVP_ACTION_CALCBOOLNOT_ST` |
| `0x45`–`0x4A` | `IsEqualFlag`, `IsNotEqualFlag`, `IsLessThanFlag`, `IsGreaterThanFlag`, `IsLteFlag`, `IsGteFlag` | `ADVP_ACTION_CALCADD_ST` |
| `0x4B` | `SetRandFlag` | `ADVP_ACTION_RANDAM_ST` |
| `0x5A` | `ActionJamp` | `ADVP_ACTION_JUMP_ST` |
| `0x5B` | `ActionBranch` | `ADVP_ACTION_BRANCH_ST` |
| `0x5C` | `Nop` | `ADVP_ACTIONDATA_ST` |
| `0x5D` | `Wait` | `ADVP_ACTION_WAIT_ST` |
| `0x64` | `CallDialog` | `ADVP_ACTION_DIALOG_ST` |
| `0x6E`–`0x70` | `PlaySE`, `StopSE`, `ChangeVolSe` | `ADVP_ACTION_SE*_ST` |
| `0x71`–`0x74` | `PlayBGM`, `StopBGM`, `ChangeVolBGM`, `BackReadBGM` | `ADVP_ACTION_BGM*_ST` |
| `0x82` | `InitGame` | `ADVP_ACTIONDATA_ST` |
| `0x8C`–`0x9A` | Screen effects: fade, flash, quake, scale, gray scale, sepia, roll, random | `ADVP_ACTION_IE*_ST` |

Opcodes that King's Field uses and Adventure Player does not handle: `0x23`, `0x24`, `0x65`, `0x66`, `0x83`–`0x85`, `0xA0`, `0xA1`, `0xA5`–`0xA8`.

## Adventure Player Studio

An InstallShield archive holds an MSI and `ADVEN.CAB`. The CAB holds `AdventurePlayerStudio.EXE` (MFC 7.0, Xerces-C 2.5, ICU 3.0) and a 50 MB data file. Projects are XML. The tool writes `.adv` files.

Its validation messages state engine limits for the 2005 version. The King's Field build can differ.

| Item | Limit |
|---|---|
| Text | 1,024 bytes, 256 line breaks |
| Text window line, main window (not full screen) | 36 half-width characters display safely |
| Full-screen line | 56 half-width characters |
| Dialog line | 42 half-width characters |
| Dialog question | 640 bytes |
| Dialog answer | 64 bytes |
| Sub-window text | 16 bytes |
| Command name, item name, folder name | 6 full-width or 12 half-width characters |
| Tab name | 5 full-width or 10 half-width characters |
| Title | 15 full-width or 30 half-width characters |
| Author | 10 full-width or 20 half-width characters |
| Rows or columns in a list | 1–20 |
