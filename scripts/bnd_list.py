"""List BND container entries, recursing into nested BNDs.

Usage: python3 -I scripts/bnd_list.py FILE.bnd [...]
"""
import struct
import sys


def walk(d, label, depth=0):
    _, ver, flag, size, cnt = struct.unpack_from("<4sIIII", d, 0)
    pad = "  " * depth
    print(f"{pad}== {label} ver={ver:#x} flag={flag:#x} size={size:#x} len={len(d):#x} count={cnt}")
    for i in range(cnt):
        idn, off, sz, no = struct.unpack_from("<IIII", d, 0x20 + i * 16)
        name = d[no:d.index(b"\0", no)].decode("ascii", "replace") if no else f"#{idn}"
        b = d[off:off + sz]
        print(f"{pad}  {idn:5d} off={off:#08x} size={sz:#08x} {name:24s} {b[:8].hex()}")
        if b[:4] == b"BND\0":
            walk(b, name, depth + 2)


for p in sys.argv[1:]:
    with open(p, "rb") as f:
        walk(f.read(), p)
