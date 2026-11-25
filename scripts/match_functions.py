"""Carry function names from a symbol-bearing BOOT.BIN to an unnamed one.

Usage: python3 -I scripts/match_functions.py NAMED_BOOT NAMED_TSV TARGET_BOOT TARGET_TSV OUT.tsv
TSV files are functions.tsv from scripts/ghidra/DumpProgram.java.

Each function is fingerprinted by its instruction sequence with immediates,
jump targets and register-independent constants masked. A fingerprint that is
unique in both binaries is a match. Matches then spread to callees: a call at
the same position in two matched functions links the two callees.
"""
import collections
import hashlib
import struct
import sys

TEXT_OFF = 0x80


def load(boot, tsv):
    code = open(boot, "rb").read()
    funcs = []
    for line in open(tsv, encoding="utf-8").read().splitlines()[1:]:
        entry, name, size, _ = line.split("\t")
        funcs.append((int(entry, 16), name, int(size)))
    return code, funcs


def mask(w):
    op = w >> 26
    if op in (2, 3):  # j, jal
        return w & 0xFC000000
    if op == 0:  # SPECIAL: keep all but shift amount
        return w & 0xFFFFF83F
    if op == 0x0F:  # lui: drop immediate
        return w & 0xFFFF0000
    if op in (0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E) or op >= 0x20:
        return w & 0xFFFF0000  # ALU immediates and loads/stores: drop offset
    if op in (4, 5, 6, 7, 1):  # branches: drop offset
        return w & 0xFFFF0000
    return w


def fingerprint(code, entry, size):
    words = struct.unpack_from(f"<{size // 4}I", code, TEXT_OFF + entry)
    return hashlib.sha1(b"".join(struct.pack("<I", mask(w)) for w in words)).hexdigest()


def calls(code, entry, size):
    out = []
    for i in range(size // 4):
        w = struct.unpack_from("<I", code, TEXT_OFF + entry + i * 4)[0]
        if w >> 26 == 3:
            out.append((w & 0x03FFFFFF) << 2)
    return out


a_code, a_funcs = load(sys.argv[1], sys.argv[2])
b_code, b_funcs = load(sys.argv[3], sys.argv[4])
a_name = {e: n for e, n, _ in a_funcs}
a_size = {e: s for e, _, s in a_funcs}
b_size = {e: s for e, _, s in b_funcs}

fa = collections.defaultdict(list)
fb = collections.defaultdict(list)
for e, _, s in a_funcs:
    if s >= 16:
        fa[fingerprint(a_code, e, s)].append(e)
for e, _, s in b_funcs:
    if s >= 16:
        fb[fingerprint(b_code, e, s)].append(e)

match = {}
for fp, ea in fa.items():
    eb = fb.get(fp)
    if eb and len(ea) == 1 and len(eb) == 1:
        match[eb[0]] = ea[0]
exact = len(match)

changed = True
while changed:
    changed = False
    for b_e, a_e in list(match.items()):
        ca, cb = calls(a_code, a_e, a_size[a_e]), calls(b_code, b_e, b_size[b_e])
        if len(ca) != len(cb):
            continue
        for x, y in zip(ca, cb):
            if y not in match and x in a_name and y in b_size:
                match[y] = x
                changed = True

with open(sys.argv[5], "w", encoding="utf-8") as out:
    out.write("target\tsource\tname\n")
    for b_e in sorted(match):
        out.write(f"{b_e:08x}\t{match[b_e]:08x}\t{a_name[match[b_e]]}\n")
print(f"target functions {len(b_funcs)}, exact matches {exact}, after call propagation {len(match)}")
