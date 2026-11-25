"""Print the records of GameInfo.dat (AppliBox catalog inside AdvMenu.bnd).

Usage: python3 -I scripts/gameinfo_dump.py AdvMenu.bnd
"""
import re
import struct
import sys

with open(sys.argv[1], "rb") as f:
    bnd = f.read()
cnt = struct.unpack_from("<I", bnd, 0x10)[0]
for i in range(cnt):
    _, off, size, no = struct.unpack_from("<IIII", bnd, 0x20 + i * 16)
    if bnd[no:bnd.index(b"\0", no)] == b"GameInfo.dat":
        d = bnd[off:off + size]
        break
else:
    sys.exit("GameInfo.dat not found")
text = re.compile(rb"(?:[\x20-\x7e]|[\x81-\x9f\xe0-\xef][\x40-\xfc]){3,}")
for r in range(struct.unpack_from("<I", d, 0)[0]):
    rec = d[0x10 + r * 0x200:0x10 + (r + 1) * 0x200]
    print(r, [m.group().decode("cp932", "replace") for m in text.finditer(rec)])
