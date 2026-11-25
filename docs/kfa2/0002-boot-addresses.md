# 0002: BOOT.BIN addresses, Additional II

The subtitle patch and the chunk base patch ([kfa1/0010](../kfa1/0010-subtitle-patch.md)) need addresses in `BOOT.BIN`. Additional II has other addresses than Additional I.

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
