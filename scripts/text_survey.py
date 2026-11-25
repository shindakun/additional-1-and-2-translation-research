"""Classify Shift-JIS strings in extracted main.dat chunks and count unique text.

Usage: python3 -I scripts/text_survey.py research/maindat/<game>
Input is the OUTDIR written by scripts/maindat.py.
"""
import collections
import re
import struct
import sys

root = sys.argv[1]
u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]
body = b"\0" * 0x130 + open(f"{root}/body.bin", "rb").read()
th = u32(body, 0x14C)
rows = [struct.unpack_from("<4I", body, u32(body, th + 4) + i * 16) for i in range(u32(body, th))]
pat = re.compile(rb"(?<=\x00)(?:[\x81-\x9f\xe0-\xef][\x40-\x7e\x80-\xfc]|[\x20-\x7e]|\r\n)+(?=\x00)")


def text(b):
    try:
        s = b.decode("cp932")
    except UnicodeDecodeError:
        return None
    return s if any("぀" <= c <= "ヿ" or "一" <= c <= "鿿" for c in s) else None


count = collections.Counter()
uniq = collections.defaultdict(set)
for i, (_, ident, off, _) in enumerate(rows):
    d = open(f"{root}/chunks/{i:03d}_{ident:05d}.bin", "rb").read()
    ptrs = {u32(d, o) - off for o in range(0, len(d) - 3, 4)}
    for m in pat.finditer(d):
        s, p = text(m.group()), m.start()
        if not s:
            continue
        if p in ptrs:
            kind = "pointer"
        elif p % 0x10 == 0 and len(m.group()) < 0x18 and not any(d[p + len(m.group()):p + 0x18]):
            kind = "record"
        elif any("\u3040" <= c <= "\u30ff" for c in s):
            kind = "other"
        else:
            continue
        count[kind] += 1
        uniq[kind].add(s)

n, tab = u32(body, 0x204), u32(body, 0x208)
for v in struct.unpack_from(f"<{n}I", body, tab):
    uniq["global"].add(body[v:body.index(b"\0", v)].decode("cp932", "replace"))
count["global"] = n

print("kind     found  unique")
for k in ("global", "pointer", "record", "other"):
    print(f"{k:8s} {count[k]:6d} {len(uniq[k]):7d}")
real = uniq["global"] | uniq["pointer"] | uniq["record"]
print(f"unique text strings {len(real)}, characters {sum(len(s) for s in real)}")
