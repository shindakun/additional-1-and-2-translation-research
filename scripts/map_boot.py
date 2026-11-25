#!/usr/bin/env python3
"""Map BOOT.BIN addresses from one game to another by code patterns.

Usage: python3 -I scripts/map_boot.py FROM.BIN TO.BIN ADDR...

Each ADDR is a code or data address in FROM (hex). For a code address the
script searches TO for the same 16 words around it, with jal targets and
16-bit immediates masked, and prints the unique match. For a data address
(not inside a matched code pattern) it finds an instruction pair in FROM that
builds the address (lui, then an addiu/ori/lw/sw/lb/lbu on the same register)
and reads the same pair at the matching place in TO. See docs/kfa2/0001.
"""
import struct
import sys


def words(path):
    d = open(path, "rb").read()
    ph = struct.unpack_from("<I", d, 0x1C)[0]
    off = struct.unpack_from("<I", d, ph + 4)[0]
    size = struct.unpack_from("<I", d, ph + 16)[0]
    return [struct.unpack_from("<I", d, off + a)[0] for a in range(0, size, 4)]


def mask(w):
    op = w >> 26
    if op in (2, 3):  # j, jal
        return w & 0xFC000000
    if op in (8, 9, 10, 11, 12, 13, 14, 15) or op >= 0x20:  # immediates
        return w & 0xFFFF0000
    return w


A, B = words(sys.argv[1]), words(sys.argv[2])
MA, MB = [mask(w) for w in A], [mask(w) for w in B]
index = {}
for j in range(len(MB) - 16):
    index.setdefault(tuple(MB[j:j + 16]), []).append(j)


def match(i):
    """Index in B of the code at word i of A, or None."""
    for lo in (i - 8, i, i - 15):
        if lo < 0:
            continue
        hits = index.get(tuple(MA[lo:lo + 16]), [])
        if len(hits) == 1:
            return hits[0] + (i - lo)
    return None


def sx(v):
    return v - 0x10000 if v & 0x8000 else v


def data_site(addr):
    """Word index in A of a lui whose pair builds addr, and the pair index."""
    hi = (addr + 0x8000) >> 16
    for i, w in enumerate(A):
        if w >> 26 == 0xF and (w & 0xFFFF) == hi:
            rt = (w >> 16) & 31
            for k in range(i + 1, min(i + 8, len(A))):
                x = A[k]
                if x >> 26 in (8, 9, 0xD, 0x20, 0x21, 0x23, 0x24, 0x25, 0x28, 0x29, 0x2B) and (x >> 21) & 31 == rt:
                    lo = x & 0xFFFF
                    v = (hi << 16) + (lo if x >> 26 == 0xD else sx(lo))
                    if v & 0xFFFFFFFF == addr:
                        yield i, k


for arg in sys.argv[3:]:
    a = int(arg, 16)
    j = match(a // 4) if a // 4 < len(A) else None
    if j is not None and MA[a // 4] == MB[j]:
        print(f"{arg}\tcode\t{j * 4:#x}")
        continue
    found = None
    for i, k in data_site(a):
        bi, bk = match(i), match(k)
        if bi is None or bk is None or bk - bi != k - i:
            continue
        hi, x = B[bi] & 0xFFFF, B[bk]
        lo = x & 0xFFFF
        found = (hi << 16) + (lo if x >> 26 == 0xD else sx(lo))
        break
    print(f"{arg}\tdata\t{found:#x}" if found is not None else f"{arg}\tnot found")
