"""Walk the objects and command lists of every main.dat chunk.

Usage: python3 -I scripts/chunk_commands.py research/maindat/<game>
Input is the OUTDIR written by scripts/maindat.py. Prints object and command
counts, the opcode histogram, and the number of command words that equal the
address of a text string.
"""
import collections
import struct
import sys

root = sys.argv[1]
u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]
body = b"\0" * 0x130 + open(f"{root}/body.bin", "rb").read()
th = u32(body, 0x14C)
rows = [struct.unpack_from("<4I", body, u32(body, th + 4) + i * 16) for i in range(u32(body, th))]


def is_text(d, p):
    if not 0 < p < len(d) or d[p - 1] != 0:
        return False
    try:
        s = d[p:d.index(b"\0", p)].decode("cp932")
    except UnicodeDecodeError:
        return False
    if any(ord(c) < 0x20 and c not in "\r\n" for c in s):
        return False
    return any("぀" <= c <= "ヿ" or "一" <= c <= "鿿" for c in s)


ops = collections.Counter()
objects = commands = text_refs = bad = 0
for i, (_, ident, off, _) in enumerate(rows):
    d = open(f"{root}/chunks/{i:03d}_{ident:05d}.bin", "rb").read()
    count, table = u32(d, u32(d, 0x48) - off), u32(d, 0x54) - off
    for k in range(count):
        obj = u32(d, table + k * 4) - off
        n_cmd = struct.unpack_from("<H", d, obj + 0x3A)[0]
        p = u32(d, obj + 0x3C) - off
        end = p + u32(d, obj + 0x40)
        objects += 1
        for _ in range(n_cmd):
            op, length = d[p], d[p + 1]
            if length == 0 or length % 4:
                bad += 1
                break
            ops[op] += 1
            commands += 1
            text_refs += sum(is_text(d, u32(d, p + a) - off) for a in range(4, length, 4))
            p += length
        if p != end:
            bad += 1

print(f"objects {objects}, commands {commands}, walk errors {bad}, text pointers in commands {text_refs}")
print("opcode (hex): count")
print(" ".join(f"{op:02x}:{n}" for op, n in sorted(ops.items())))
