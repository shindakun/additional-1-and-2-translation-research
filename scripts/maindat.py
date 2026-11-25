"""Verify and extract main.dat (ADVP container).

Usage: python3 -I scripts/maindat.py BOOT.BIN main.dat [OUTDIR]

Derives the AES key from BOOT.BIN, checks the MD5, validates the chunk table
and, when OUTDIR is given, writes every part decrypted. Needs openssl on PATH.
See docs/0003-maindat.md.
"""
import hashlib
import os
import struct
import subprocess
import sys

# Virtual addresses of the encrypted key table and its key, per BOOT.BIN SHA-1.
KEY_TABLE = {
    "e62c3559fc94e9c917081bdf5bbf8dd5713ef1ae": (0x12C85C, 0x12C918),  # Additional I
    "e832e575aef3cb4cf1e68ec3104c970a0191ab11": (0x12CA5C, 0x12CB18),  # Additional II
    "c37b28402bc0405c626be1b01329fd305838f20d": (0x123944, 0x123A10),  # Adventure Player
}
ELF_DATA_DELTA = 0x80


def aes(mode, key, data, iv=None):
    args = ["openssl", "enc", "-d", f"-aes-128-{mode}", "-nopad", "-K", key.hex()]
    if iv is not None:
        args += ["-iv", iv.hex()]
    return subprocess.run(args, input=data, capture_output=True, check=True).stdout


def keys(boot):
    table_va, key_va = KEY_TABLE[hashlib.sha1(boot).hexdigest()]
    at = lambda va, n: boot[va + ELF_DATA_DELTA:va + ELF_DATA_DELTA + n]
    rec = aes("ecb", at(key_va, 16), at(table_va, 0x80))
    col = lambda c: bytes(rec[i * 8 + c] for i in range(16))
    # Key index n: salt column n, key column 2+n, IV column 4+n.
    return [{"salt": col(n), "key": col(2 + n), "iv": col(4 + n)} for n in (0, 1)]


def main():
    boot = open(sys.argv[1], "rb").read()
    d = open(sys.argv[2], "rb").read()
    out = sys.argv[3] if len(sys.argv) > 3 else None
    ks = keys(boot)

    k = None
    for n, cand in enumerate(ks):
        hdr = aes("cbc", cand["key"], d[0x10:0x130], cand["iv"])
        if hdr[:4] == b"ADVP":
            k, idx = cand, n
            break
    if k is None:
        sys.exit("no key decrypts the ADVP header")
    dec = lambda off, n: aes("cbc", k["key"], d[off:off + (n + 15) // 16 * 16], k["iv"])[:n]
    u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]

    # Header fields, offsets relative to file start (header begins at 0x10).
    h = b"\0" * 0x10 + hdr
    version, size = u32(h, 0x14), u32(h, 0x18)
    body_end, body_start = u32(h, 0x58), u32(h, 0x5C)
    disc_id = h[0x34:h.index(b"\0", 0x34)].decode()
    print(f"key index {idx}, version {version:#x}, size {size:#x} (file {len(d):#x}), disc {disc_id}")

    md5 = hashlib.md5(k["salt"] + d[16:]).digest()
    print("md5", "ok" if md5 == d[:16] else f"MISMATCH {md5.hex()} != {d[:16].hex()}")

    body = b"\0" * body_start + dec(body_start, body_end - body_start)
    table_hdr = u32(body, 0x14C)
    if table_hdr == 0:
        print("no chunk table at body 0x14C; this body layout is not decoded")
        if out:
            os.makedirs(out, exist_ok=True)
            open(os.path.join(out, "header.bin"), "wb").write(h[0x10:0x130])
            open(os.path.join(out, "body.bin"), "wb").write(body[body_start:])
            print("wrote header.bin and body.bin to", out)
        return
    count, table = u32(body, table_hdr), u32(body, table_hdr + 4)
    rows = [struct.unpack_from("<4I", body, table + i * 16) for i in range(count)]
    good = sum(1 for (sz, _, off, _) in rows if u32(dec(off, 16), 0) == sz)
    print(f"chunks {count} at table {table:#x}, self-size ok {good}")

    sound_size, sound_base = u32(body, 0x1E0), u32(body, 0x1E4)
    bnds, o = [], sound_base
    while d[o:o + 4] == b"BND\0":
        bnds.append((o, u32(d, o + 0xC)))
        o = (o + u32(d, o + 0xC) + 0x7FF) & ~0x7FF
    print(f"sound BNDs {len(bnds)} from {sound_base:#x}, first size {bnds[0][1]:#x} (body says {sound_size:#x})")

    extra = [struct.unpack_from("<2I", h, 0x60 + i * 8) for i in range(4)]
    print("extra ranges", [(hex(a), hex(b)) for a, b in extra])
    bgm_off, bgm_len = extra[3]
    bgm_tab = dec(bgm_off, bgm_len)
    bgm = [struct.unpack_from("<4I", bgm_tab, 0x10 + i * 16) for i in range(u32(bgm_tab, 0))]
    end = bgm[-1][2] + bgm[-1][1]
    print(f"tail tracks {len(bgm)}, data end {end:#x}")

    if not out:
        return
    for sub in ("chunks", "sound", "bgm"):
        os.makedirs(os.path.join(out, sub), exist_ok=True)
    w = lambda name, data: open(os.path.join(out, name), "wb").write(data)
    w("header.bin", h[0x10:0x130])
    w("body.bin", body[body_start:])
    for i, (sz, ident, off, _) in enumerate(rows):
        w(f"chunks/{i:03d}_{ident:05d}.bin", dec(off, sz))
    for i, (off, sz) in enumerate(bnds):
        w(f"sound/{i:03d}_{off:08x}.bnd", d[off:off + sz])
    w("thumb.bmp", dec(*extra[1]))
    w("skin.skn", dec(*extra[2]))
    for ident, sz, off, _ in bgm:
        w(f"bgm/{ident}.at3", dec(off, sz))
    print("wrote", out)


main()
