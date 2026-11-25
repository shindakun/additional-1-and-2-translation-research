"""Print every string in an FMG message table.

Usage: python3 -I scripts/fmg_dump.py FILE.fmg [OFFSET SIZE]
OFFSET and SIZE (hex) select an FMG inside a larger file, such as a BND.
"""
import struct
import sys

with open(sys.argv[1], "rb") as f:
    d = f.read()
if len(sys.argv) == 4:
    off, size = int(sys.argv[2], 16), int(sys.argv[3], 16)
    d = d[off:off + size]
groups, count, table = struct.unpack_from("<III", d, 0x0C)
offsets = struct.unpack_from(f"<{count}I", d, table)
for g in range(groups):
    index, first, last = struct.unpack_from("<iii", d, 0x1C + g * 12)
    for k in range(last - first + 1):
        o = offsets[index + k]
        if o:
            text = d[o:d.index(b"\0", o)].decode("cp932", "replace")
            print(f"{first + k}\t{text!r}")
