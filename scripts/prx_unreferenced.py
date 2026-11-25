"""List functions that no relocation in a PSP PRX refers to.

Usage: python3 -I scripts/prx_unreferenced.py BOOT.BIN research/ghidra/<game>/functions.tsv [N]

A relocatable PRX records every absolute address in its relocation sections
(type 0x700000A0, Elf32_Rel). Calls are R_MIPS_26, address constants are
R_MIPS_HI16/LO16 pairs, and pointers in data are R_MIPS_32. A function whose
entry no relocation produces cannot be reached, so its bytes are free space.
Prints the N largest such functions and the total size.
"""
import struct
import sys

R_MIPS_32, R_MIPS_26, R_MIPS_HI16, R_MIPS_LO16 = 2, 4, 5, 6
boot = open(sys.argv[1], "rb").read()
u32 = lambda o: struct.unpack_from("<I", boot, o)[0]
shoff = u32(0x20)
shnum = struct.unpack_from("<H", boot, 0x30)[0]
sections = [struct.unpack_from("<10I", boot, shoff + i * 40) for i in range(shnum)]
# Segment 0 (code and data) loads at address 0; file offset = address + its
# offset in the file. All relocations in this executable use segment 0.
seg_file = min(s[4] for s in sections if s[1] == 1 and s[3] == 0 and s[5])

targets, hi = set(), []
for s in sections:
    if s[1] != 0x700000A0:
        continue
    for k in range(s[5] // 8):
        r_offset, r_info = struct.unpack_from("<II", boot, s[4] + k * 8)
        kind = r_info & 0xFF
        word = u32(seg_file + r_offset)
        if kind == R_MIPS_26:
            targets.add((word & 0x03FFFFFF) << 2)
        elif kind == R_MIPS_32:
            targets.add(word)
        elif kind == R_MIPS_HI16:
            hi.append(word & 0xFFFF)
        elif kind == R_MIPS_LO16:
            lo = word & 0xFFFF
            lo = lo - 0x10000 if lo & 0x8000 else lo
            for h in hi or [0]:
                targets.add(((h << 16) + lo) & 0xFFFFFFFF)
            hi = []

# PC-relative branches carry no relocation. Count their targets too, so a
# function entered by a branch from elsewhere is not reported as free.
text = next(s for s in sections if s[1] == 1 and s[3] == 0 and s[5])
for pc in range(0, text[5], 4):
    w = u32(seg_file + pc)
    op = w >> 26
    if op in (4, 5, 6, 7, 0x14, 0x15, 0x16, 0x17) or (op == 1 and ((w >> 16) & 0x1F) in (0, 1, 2, 3, 0x10, 0x11)):
        off = w & 0xFFFF
        off = off - 0x10000 if off & 0x8000 else off
        targets.add(pc + 4 + off * 4)

funcs = []
for line in open(sys.argv[2], encoding="utf-8").read().splitlines()[1:]:
    entry, name, size, _ = line.split("\t")
    funcs.append((int(entry, 16), name, int(size)))
free = sorted((f for f in funcs if f[0] not in targets), key=lambda f: -f[2])
n = int(sys.argv[3]) if len(sys.argv) > 3 else 20
print(f"functions {len(funcs)}, unreferenced {len(free)}, bytes {sum(f[2] for f in free)}")
for entry, name, size in free[:n]:
    print(f"{entry:08x} {size:6d} {name}")
