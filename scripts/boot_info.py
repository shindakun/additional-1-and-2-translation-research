"""Print BOOT.BIN section headers and the file offset of the AES Te0 table.

Usage: python3 -I scripts/boot_info.py BOOT.BIN
"""
import struct
import sys

with open(sys.argv[1], "rb") as f:
    d = f.read()
shoff = struct.unpack_from("<I", d, 0x20)[0]
shnum, shstrndx = struct.unpack_from("<HH", d, 0x30)
sh = [struct.unpack_from("<10I", d, shoff + i * 40) for i in range(shnum)]
strtab = sh[shstrndx][4]
for s in sh:
    name = d[strtab + s[0]:d.index(b"\0", strtab + s[0])].decode()
    print(f"{name:34s} type={s[1]:#x} addr={s[3]:#08x} off={s[4]:#08x} size={s[5]:#08x}")
te0 = d.find(struct.pack("<I", 0xA56363C6))
print("AES Te0 file offset:", hex(te0) if te0 >= 0 else "not found")
