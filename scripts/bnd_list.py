"""List BND container entries, recursing into nested BNDs.

Usage: python3 -I scripts/bnd_list.py FILE.bnd [...]
       python3 -I scripts/bnd_list.py --out DIR FILE.bnd
--out writes the top-level entries of one BND to DIR under their names.
"""
import os
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


if sys.argv[1] == "--out":
    out, path = sys.argv[2], sys.argv[3]
    data = open(path, "rb").read()
    os.makedirs(out, exist_ok=True)
    for i in range(struct.unpack_from("<I", data, 0x10)[0]):
        idn, off, sz, no = struct.unpack_from("<IIII", data, 0x20 + i * 16)
        name = data[no:data.index(b"\0", no)].decode("ascii") if no else f"{idn:05d}.bin"
        open(os.path.join(out, name), "wb").write(data[off:off + sz])
    print("wrote", out)
else:
    for p in sys.argv[1:]:
        with open(p, "rb") as f:
            walk(f.read(), p)
