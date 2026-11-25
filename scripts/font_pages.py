"""Check CCM glyph pages against the font textures.

Usage: python3 -I scripts/font_pages.py FONTDIR [SAMPLES]
FONTDIR holds Ccm.ccm, Plt00.plt, Tex00.gim and Tex01.gim (unpacked font BND).
Prints glyphs per page and tests that sampled glyphs have ink in texture
page // 4, layer page % 4. See docs/0005-fonts.md.
"""
import collections
import random
import struct
import sys

W = 512
IMG_DATA = 0x480 + 0x10 + 0x40


def load(path):
    d = open(path, "rb").read()
    src, px, i = d[IMG_DATA:IMG_DATA + W * W], bytearray(W * W), 0
    for by in range(W // 8):
        for bx in range(W // 16):
            for r in range(8):
                y, x = by * 8 + r, bx * 16
                px[y * W + x:y * W + x + 16] = src[i:i + 16]
                i += 16
    return px


font = sys.argv[1]
samples = int(sys.argv[2]) if len(sys.argv) > 2 else 400
tex = [load(f"{font}/Tex00.gim"), load(f"{font}/Tex01.gim")]
plt = open(f"{font}/Plt00.plt", "rb").read()
alpha = [[plt[layer * 1024 + i * 4 + 3] for i in range(256)] for layer in range(4)]
ccm = open(f"{font}/Ccm.ccm", "rb").read()
count, table = struct.unpack_from("<H", ccm, 0x12)[0], struct.unpack_from("<I", ccm, 0x18)[0]
glyphs = [struct.unpack_from("<4f4h", ccm, table + g * 24) for g in range(count)]

pages = collections.Counter(g[7] for g in glyphs)
print("glyphs per page:", ", ".join(f"{p}: {n}" for p, n in sorted(pages.items())))


def ink(t, layer, x0, y0, x1, y1):
    return sum(1 for y in range(y0, y1) for x in range(x0, x1) if alpha[layer][tex[t][y * W + x]])


random.seed(0)
hit = blank = miss = 0
for g in random.sample(range(count), min(samples, count)):
    u0, v0, u1, v1, _, _, _, page = glyphs[g]
    box = (round(u0 * W), round(v0 * W), round(u1 * W), round(v1 * W))
    if ink(page // 4, page % 4, *box):
        hit += 1
    elif not any(ink(t, layer, *box) for t in (0, 1) for layer in range(4)):
        blank += 1
    else:
        miss += 1
print(f"sampled {min(samples, count)}: ink in predicted layer {hit}, miss {miss}, blank elsewhere {blank}")
