"""List text through the engine's own structures: global text and scene text.

Usage: python3 -I scripts/scene_text.py research/maindat/<game> [OUT.tsv]
Input is the OUTDIR written by scripts/maindat.py. OUT.tsv gets one row per
text pointer: source, chunk, scene ID, text index, pointer site, text.
See docs/0004-text.md for the structures.
"""
import re
import struct
import sys

root = sys.argv[1]
u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]
NONE = 0xFFFFFFFF
body = b"\0" * 0x130 + open(f"{root}/body.bin", "rb").read()
th = u32(body, 0x14C)
rows = [struct.unpack_from("<4I", body, u32(body, th + 4) + i * 16) for i in range(u32(body, th))]


def cstr(d, p):
    return d[p:d.index(b"\0", p)].decode("cp932", "replace")


out = []
# Global text: game data (file 0x130) +0x14 -> text header {count, offset table}.
hdr = u32(body, 0x130 + 0x14)
count, table = u32(body, hdr), u32(body, hdr + 4)
for k in range(count):
    p = u32(body, table + 4 * k)
    if p != NONE:
        out.append(("global", "", "", k, table + 4 * k, cstr(body, p)))

scenes = 0
for i, (_, ident, off, _) in enumerate(rows):
    d = open(f"{root}/chunks/{i:03d}_{ident:05d}.bin", "rb").read()
    n, lst = struct.unpack_from("<H", d, 0x0C)[0], u32(d, 0x14) - off
    for s in range(n):
        sp = u32(d, lst + 4 * s)
        if sp == NONE:
            continue
        scene = sp - off
        scenes += 1
        hdr = u32(d, scene + 0x18)
        if hdr == NONE:
            continue
        count, table = u32(d, hdr - off), u32(d, hdr - off + 4)
        if table == NONE:
            continue
        for k in range(count):
            site = table - off + 4 * k
            p = u32(d, site)
            if p != NONE:
                out.append(("scene", i, u32(d, scene + 4), k, site, cstr(d, p - off)))

texts = [r for r in out if r[5]]
print(f"scenes {scenes}, text pointers {len(out)}, non-empty {len(texts)}, unique {len({r[5] for r in texts})}")
if len(sys.argv) > 2:
    with open(sys.argv[2], "w", encoding="utf-8") as f:
        f.write("source\tchunk\tscene\tindex\tsite\ttext\n")
        for src, chunk, scene, k, site, text in out:
            f.write(f"{src}\t{chunk}\t{scene}\t{k}\t{site:#x}\t{text.replace(chr(13), '').replace(chr(10), '\\n')}\n")


def width(line):
    line = re.sub(r"#[a-z](\[[^\]]*\])?", "", line)
    return sum(1 if ord(c) < 0x80 or 0xFF61 <= ord(c) <= 0xFF9F else 2 for c in line)


widths, line_counts, sizes = [], [], []
for r in texts:
    lines = r[5].split("\r\n")
    line_counts.append(len(lines))
    widths += [width(x) for x in lines]
    sizes.append(len(r[5].encode("cp932", "replace")))
widths.sort()
print(f"line width (half-width units): max {widths[-1]}, p99 {widths[int(0.99 * len(widths))]}; "
      f"most lines {max(line_counts)}; longest text {max(sizes)} bytes")
