# 0003: main.dat

`main.dat` is an `ADVP` container. Both games use the same format and the same keys. `scripts/maindat.py` verifies and extracts it.

```sh
python3 -I scripts/maindat.py BOOT.BIN main.dat [OUTDIR]
```

## Layout

| Range | Contents | Encrypted |
|---|---|---|
| `0x0000`–`0x000F` | MD5 | No |
| `0x0010`–`0x012F` | Header, 0x120 bytes | Yes, own chain |
| `0x0130`–body end | Header body: chunk table, sound and resource pointers, scene names | Yes, own chain |
| Chunk table entries | Scene chunks | Yes, one chain per chunk |
| Sound base onward | Unnamed BNDs (`0x633`) of HVG sound banks | No |
| Extra range 1 | BMP thumbnail | Yes |
| Extra range 2 | `SKIN_DAT4` skin | Yes |
| Extra range 3 | Music table | Yes |
| Music table entries | ATRAC3plus tracks, IDs from 50000 | Yes, one chain per track |

Sections follow each other in this order. No byte falls outside them.

| | Additional I | Additional II |
|---|---|---|
| Body end | `0x6B800` | `0x6BD10` |
| Chunk table | `0x6B110`, 111 entries | `0x6B6F0`, 98 entries |
| Chunks | `0x6B800` to `0xAD3C570` | `0x6C000` to `0x960F700` |
| Sound base | `0xAD3C800`, 106 BNDs | `0x960F800`, 93 BNDs |
| Thumbnail | `0xD5FF800`, 0x8800 | `0xB811000`, 0x8800 |
| Skin | `0xD608000`, 0x42800 | `0xB819800`, 0x42800 |
| Music table | `0xD64A800`, 0x210 | `0xB85C000`, 0x250 |
| Tracks | 32 | 36 |
| Data end | `0xE294560` | `0xC68AE90` |

## Encryption

AES-128-CBC. Every encrypted unit restarts the chain with the same IV. Data is padded to 16 bytes.

Units:

- The header, `0x10` to `0x130`.
- The body, from `[0x5C]` to `[0x58]`.
- Each chunk, from its table offset for its table size.
- The thumbnail, skin and music table, from their header ranges.
- Each music track, from its music table entry.

### Key derivation

`BOOT.BIN` holds a 0x80-byte encrypted key table and its AES-128 key. Virtual addresses (file offset − 0x80):

| | Additional I | Additional II |
|---|---|---|
| Encrypted table | `0x12C85C` | `0x12CA5C` |
| Table key | `0x12C918` | `0x12CB18` |

Both games contain the same table bytes and the same table key.

`FUN_0004fdd0` (Additional I) decrypts the table with AES-128-ECB at startup. The result is 16 records of 8 bytes. Bytes 0–5 of each record are columns 0–5; bytes 6–7 are zero. Each column, read down the 16 records, is a 16-byte value:

| Column | Use |
|---|---|
| 0 | MD5 salt, key index 0 |
| 1 | MD5 salt, key index 1 |
| 2 | AES key, key index 0 |
| 3 | AES key, key index 1 |
| 4 | IV, key index 0 |
| 5 | IV, key index 1 |

`main.dat` uses key index 0 on both discs. Bit 2 of the u16 at file offset `0x82` selects index 1. The u16 is 0 on both discs.

### MD5

MD5 over the full file, with bytes `0x0`–`0xF` replaced by the salt. The result is stored in bytes `0x0`–`0xF`.

`FUN_000544a4` checks it. That function has no direct callers. Its use on the disc load path is not known.

## Header

Offsets are file offsets. The header starts at `0x10`.

| Offset | Value (Additional I) | Meaning |
|---|---|---|
| `0x10` | `ADVP` | Magic |
| `0x14` | `0x05030920` | Version. The loader rejects values below this. |
| `0x18` | `0xE294800` | File size |
| `0x20` | `0x0080`, `0x0002` | Not known; same on both discs |
| `0x24` | `0x4F` | Not known; same on both discs |
| `0x34` | `ULJS00070` | Disc ID |
| `0x58` | `0x6B800` | Body end |
| `0x5C` | `0x130` | Body start |
| `0x60` | 4 × (offset, size) | Extra ranges 0–3. Range 0 is unused (`0xFFFFFFFF`, 0). |
| `0x82` | u16 | Flags; bit 2 selects key index 1 |
| `0x90` | `KING'S FIELD ADDITIONAL` | Title |
| `0xD0` | `FromSoftWare` | Publisher |
| `0x120` | 16 bytes | Must equal 16 bytes at `BOOT.BIN` `0x12CAFC` (zero) |

## Body

| Offset | Meaning |
|---|---|
| `0x14C` | Pointer to the chunk table header |
| `0x1DC` | Pointer to a structure with count `0x41` (not known) |
| `0x1E0` | First sound BND size, rounded to 0x800 |
| `0x1E4` | Sound base |
| `0x1F4` | Pointer to a Shift-JIS name list |

All pointers are file offsets.

### Chunk table

Table header: count (u32), table offset (u32). Entry, 16 bytes:

| Offset | Field |
|---|---|
| 0x0 | Size |
| 0x4 | Chunk ID |
| 0x8 | File offset |
| 0xC | 0 |

The first u32 of each decrypted chunk equals its size.

## Music table

Decrypted: count (u32) and 12 bytes of padding, then 16-byte entries:

| Offset | Field |
|---|---|
| 0x0 | Track ID |
| 0x4 | Size |
| 0x8 | File offset |
| 0xC | 0 |

Tracks are not sector-aligned. Each decrypts to a RIFF ATRAC3plus file.

## Chunks

A chunk is a scene. It starts with its size, its ID and counts. A Shift-JIS scene name is at offset `0x20`. Examples: `街・戦闘`, `オープニング`, `223`, `B12F　神殿　陣`. Additional I chunks hold 1,259 GIM images.

The rest of the chunk format is not known.

## Code

Additional I addresses (Ghidra, base 0).

| Function | Role |
|---|---|
| `FUN_0004fdd0` | Decrypts the key table, fills key, IV and salt arrays |
| `FUN_000533f8` | AES-CBC decrypt with key index n |
| `FUN_0005387c` | AES-CBC decrypt |
| `FUN_000544a4` | Verifies an `ADVP` file: header, magic, key index, MD5 |
| `FUN_00056450` | Decrypts a loaded unit by slot |
| `FUN_000635e4` | Load state machine; checks magic and version |
| `FUN_0001dd1c`, `FUN_0001f520` | AES block encrypt and decrypt |
