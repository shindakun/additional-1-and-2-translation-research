"""Map main.dat: entropy per MiB, segment starts, plaintext BNDs, known magics.

Usage: python3 -I scripts/maindat_survey.py main.dat
A segment start is a 0x800-aligned offset preceded by 16 zero bytes.
"""
import collections
import math
import re
import sys

with open(sys.argv[1], "rb") as f:
    d = f.read()

MIB = 1 << 20
profile = ""
for o in range(0, len(d), MIB):
    c = collections.Counter(d[o:o + MIB])
    n = sum(c.values())
    e = -sum(v / n * math.log2(v / n) for v in c.values())
    profile += "#" if e > 7.9 else ("+" if e > 6 else ".")
print("entropy per MiB (# > 7.9 bits, + > 6, . lower):")
print(profile)

starts = [0] + [s for s in range(0x800, len(d), 0x800)
                if d[s - 16:s] == bytes(16) and d[s:s + 16] != bytes(16)]
bnds = [m.start() for m in re.finditer(b"BND\x00\x33\x06", d)]
print(f"size {len(d):#x}, segments {len(starts)}")
if bnds:
    print(f"version 0x633 BNDs: {len(bnds)}, first {bnds[0]:#x}, last {bnds[-1]:#x}")
    print("segments before first BND:", sum(1 for s in starts if s < bnds[0]))
    print("segments after last BND:", [hex(s) for s in starts if s > bnds[-1]])
for magic in [b"BND\0", b"HVG  ", b"IECSsreV", b"MIG.00.1", b"PSMF", b"RIFF", b"\x89PNG"]:
    print(f"{magic!r}: {d.count(magic)}")
