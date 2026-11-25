"""Resolve text IDs in actions and count command and flag names.

Usage: python3 -I scripts/text_refs.py research/maindat/<game>
Input is the OUTDIR written by scripts/maindat.py. See docs/0004-text.md.
"""
import collections
import struct
import sys

root = sys.argv[1]
u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]
u16 = lambda b, o: struct.unpack_from("<H", b, o)[0]
NONE = 0xFFFFFFFF
GAME = 0x130
body = b"\0" * 0x130 + open(f"{root}/body.bin", "rb").read()
th = u32(body, GAME + 0x1C)
rows = [struct.unpack_from("<4I", body, u32(body, th + 4) + i * 16) for i in range(u32(body, th))]


def table(buf, hdr, base):
    if hdr == NONE:
        return []
    n, t = u32(buf, hdr - base), u32(buf, hdr - base + 4)
    return [] if t == NONE else [u32(buf, t - base + 4 * k) for k in range(n)]


def names(buf, ptr, n, base):
    p = ptr - base
    return [buf[p + i * 0x20:p + i * 0x20 + 0x18].split(b"\0")[0].decode("cp932", "replace") for i in range(n)]


glob = table(body, u32(body, GAME + 0x14), 0)
stats = collections.Counter()
scene_names = set()


def resolve(v, local):
    if v >= 3000:
        k = v - 3000
        return "scene" if k < len(local) and local[k] != NONE else "bad"
    return "global" if v < len(glob) and glob[v] != NONE else "bad"


for i, (_, ident, off, _) in enumerate(rows):
    d = open(f"{root}/chunks/{i:03d}_{ident:05d}.bin", "rb").read()
    n, lst = u16(d, 0x0C), u32(d, 0x14) - off
    for k in range(n):
        sp = u32(d, lst + 4 * k)
        if sp == NONE:
            continue
        sc = sp - off
        local = table(d, u32(d, sc + 0x18), off)
        if u32(d, sc + 0x10) != NONE and u32(d, sc + 0x0C):
            cmds = names(d, u32(d, sc + 0x10), u32(d, sc + 0x0C), off)
            stats["scene command records"] += len(cmds)
            scene_names.update(c for c in cmds if c)
        eh = u32(d, sc + 0x1C)
        if eh == NONE:
            continue
        for e in range(u32(d, eh - off)):
            ep = u32(d, u32(d, eh - off + 4) - off + 4 * e)
            if ep == NONE:
                continue
            ev = ep - off
            p = u32(d, ev + 0x3C) - off
            for _ in range(u16(d, ev + 0x3A)):
                op, length = d[p], d[p + 1]
                if op == 0x29:
                    stats[f"0x29 {resolve(u16(d, p + 4), local)}"] += 1
                elif op == 0x64:
                    for a in range(0xC, 0x18, 2):
                        v = u16(d, p + a)
                        if v:
                            stats[f"0x64 {resolve(v, local)}"] += 1
                p += length

print(f"flags {u32(body, GAME + 4)}, global commands {u32(body, GAME + 0xC)}: "
      + ", ".join(names(body, u32(body, GAME + 0x10), u32(body, GAME + 0xC), 0)))
print(f"unique scene command names {len(scene_names)}")
for k in sorted(stats):
    print(f"{k}: {stats[k]}")
