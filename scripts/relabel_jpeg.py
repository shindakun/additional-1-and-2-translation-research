#!/usr/bin/env python3
"""Replace label text in a JPEG or PNG image.

Usage: relabel_jpeg.py SPEC.json IN OUT

A JPEG output is re-encoded to fit the spec "budget". A PNG output keeps the
alpha channel and has no budget (for GIM edits, docs/kfa1/0012-images.md).

SPEC.json:
  {"font": "/System/Library/Fonts/Supplemental/Georgia.ttf",
   "budget": 30000,
   "labels": [{"box": [x0, y0, x1, y1], "text": "Inn", "size": 14}, ...]}

Optional label fields: "color" [r, g, b], "threshold" (overrides the spec
"threshold"), "align" ("center" or "left"), "fill" ("inpaint", the default;
"flat"; "box": fill the whole box with its median color, for dark text on a
light box; "clear": make the box transparent), "shadow" (false: no shadow).
"text" can hold "\n" for two lines.

For each box: pixels brighter than the threshold (default 120) are the old
text. Those pixels, grown by 1 pixel, are filled from their neighbours,
which keeps the background texture ("flat" fills their bounding box with the
median of the darker pixels instead). The new text is drawn centred in the box with a
1-pixel black shadow. The output uses the highest JPEG quality whose size is
not more than the budget (docs/kfa1/0012-images.md). Needs Pillow.
"""
import json
import statistics
import sys
from io import BytesIO

from PIL import Image, ImageDraw, ImageFont


def inpaint(img, ink, box):
    """Replace the stroke pixels, grown by 1 pixel, with the mean of their
    unmasked neighbours, from the edges inward."""
    x0, y0, x1, y1 = box
    px = img.load()
    mask = set()
    for x, y, _ in ink:
        for dx in (-1, 0, 1):
            for dy in (-1, 0, 1):
                if x0 <= x + dx <= x1 and y0 <= y + dy <= y1:
                    mask.add((x + dx, y + dy))
    while mask:
        done = {}
        for x, y in mask:
            near = [px[x + dx, y + dy] for dx in (-2, -1, 0, 1, 2) for dy in (-2, -1, 0, 1, 2)
                    if (x + dx, y + dy) not in mask and 0 <= x + dx < img.width and 0 <= y + dy < img.height]
            if len(near) >= 3:
                done[(x, y)] = tuple(sum(c[i] for c in near) // len(near) for i in range(len(near[0])))
        if not done:
            break
        for (x, y), c in done.items():
            px[x, y] = c
        mask -= done.keys()


def relabel(img, label, font_path, threshold):
    threshold = label.get("threshold", threshold)
    x0, y0, x1, y1 = label["box"]
    px = img.load()
    if label.get("fill") == "clear":
        ImageDraw.Draw(img).rectangle([x0, y0, x1, y1], fill=(0, 0, 0, 0))
    if label.get("fill") == "box":
        allpx = [px[x, y] for y in range(y0, y1 + 1) for x in range(x0, x1 + 1)]
        fill = tuple(int(statistics.median(p[i] for p in allpx)) for i in range(len(allpx[0])))
        ImageDraw.Draw(img).rectangle([x0, y0, x1, y1], fill=fill)
    ink, bg = [], []
    for y in range(y0, y1 + 1):
        for x in range(x0, x1 + 1):
            r, g, b = px[x, y][:3]
            (ink if (r + g + b) / 3 > threshold else bg).append((x, y, px[x, y]))
    if ink and label.get("fill") not in ("clear", "box"):
        ix0 = max(x0, min(p[0] for p in ink) - 2)
        ix1 = min(x1, max(p[0] for p in ink) + 2)
        iy0 = max(y0, min(p[1] for p in ink) - 2)
        iy1 = min(y1, max(p[1] for p in ink) + 2)
        if label.get("fill") == "flat":
            fill = tuple(int(statistics.median(p[2][i] for p in bg)) for i in range(len(bg[0][2])))
            ImageDraw.Draw(img).rectangle([ix0, iy0, ix1, iy1], fill=fill)
        else:
            inpaint(img, ink, (x0, y0, x1, y1))
    font = ImageFont.truetype(font_path, label.get("size", 14))
    d = ImageDraw.Draw(img)
    l, t, r, b = d.multiline_textbbox((0, 0), label["text"], font=font, align="center")
    cx = (x0 + x1 - (r - l)) // 2 - l
    if label.get("align") == "left":
        cx = x0 - l
    cy = (y0 + y1 - (b - t)) // 2 - t
    color = tuple(label.get("color", (235, 228, 210)))
    if img.mode == "RGBA":
        color += (255,)
    if label.get("shadow", True):
        d.multiline_text((cx + 1, cy + 1), label["text"], font=font, align="center",
                         fill=(0, 0, 0) + ((255,) if img.mode == "RGBA" else ()))
    d.multiline_text((cx, cy), label["text"], font=font, fill=color, align="center")


def main():
    spec_path, src, dst = sys.argv[1:4]
    with open(spec_path, encoding="utf-8") as f:
        spec = json.load(f)
    png = dst.lower().endswith(".png")
    img = Image.open(src).convert("RGBA" if png else "RGB")
    for label in spec["labels"]:
        relabel(img, label, spec["font"], spec.get("threshold", 120))
    if png:
        img.save(dst)
        print(f"{dst}: written")
        return
    for q in range(95, 20, -1):
        buf = BytesIO()
        img.save(buf, "JPEG", quality=q, subsampling=2)
        if len(buf.getvalue()) <= spec["budget"]:
            with open(dst, "wb") as f:
                f.write(buf.getvalue())
            print(f"{dst}: quality {q}, {len(buf.getvalue())} of {spec['budget']} bytes")
            return
    sys.exit(f"{dst}: does not fit {spec['budget']} bytes")


if __name__ == "__main__":
    main()
