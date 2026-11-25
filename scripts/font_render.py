"""Render the four glyph layers of a font BND's textures to a grayscale PNG.

Usage: python3 -I scripts/font_render.py FONTDIR ROWS OUT.png
FONTDIR holds Ccm.ccm, Plt00.plt, Tex00.gim and Tex01.gim (unpacked font BND).
ROWS is the number of texture rows to render per layer.
"""
import struct
import sys
import zlib

W = H = 512
IMG_DATA = 0x480 + 0x10 + 0x40  # image block + block header + image header


def load(path):
    d = open(path, "rb").read()
    src, px, i = d[IMG_DATA:IMG_DATA + W * H], bytearray(W * H), 0
    for by in range(H // 8):  # PSP swizzle: 16-byte x 8-row tiles
        for bx in range(W // 16):
            for r in range(8):
                y, x = by * 8 + r, bx * 16
                px[y * W + x:y * W + x + 16] = src[i:i + 16]
                i += 16
    return px


def png(path, w, rows):
    chunk = lambda t, b: struct.pack(">I", len(b)) + t + b + struct.pack(">I", zlib.crc32(t + b))
    ihdr = struct.pack(">IIBBBBB", w, len(rows), 8, 0, 0, 0, 0)
    raw = zlib.compress(b"".join(b"\0" + r for r in rows))
    open(path, "wb").write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", ihdr) + chunk(b"IDAT", raw) + chunk(b"IEND", b""))


font, n, out = sys.argv[1], int(sys.argv[2]), sys.argv[3]
plt = open(f"{font}/Plt00.plt", "rb").read()
rows = []
for tex in ("Tex00", "Tex01"):
    px = load(f"{font}/{tex}.gim")
    for layer in range(4):
        alpha = [plt[layer * 1024 + i * 4 + 3] for i in range(256)]
        rows += [bytes(255 - alpha[px[y * W + x]] for x in range(W)) for y in range(n)]
        rows.append(bytes([128]) * W)
png(out, W, rows)
