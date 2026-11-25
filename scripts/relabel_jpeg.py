#!/usr/bin/env python3
"""Replace label text in a JPEG and re-encode it to fit a byte budget.

Usage: relabel_jpeg.py SPEC.json IN.jpg OUT.jpg

SPEC.json:
  {"font": "/System/Library/Fonts/Supplemental/Georgia.ttf",
   "budget": 30000,
   "labels": [{"box": [x0, y0, x1, y1], "text": "Inn", "size": 14}, ...]}

For each box: pixels brighter than the threshold (default 120) are the old
text. Their bounding box, grown by 2 pixels, is filled with the median of the
darker pixels in the box, and the new text is drawn centred in the box with a
1-pixel black shadow. The output uses the highest JPEG quality whose size is
not more than the budget (docs/0012-images.md). Needs Pillow.
"""
import json
import statistics
import sys
from io import BytesIO

from PIL import Image, ImageDraw, ImageFont


def relabel(img, label, font_path, threshold):
    x0, y0, x1, y1 = label["box"]
    px = img.load()
    ink, bg = [], []
    for y in range(y0, y1 + 1):
        for x in range(x0, x1 + 1):
            r, g, b = px[x, y]
            (ink if (r + g + b) / 3 > threshold else bg).append((x, y, (r, g, b)))
    if ink:
        ix0 = max(x0, min(p[0] for p in ink) - 2)
        ix1 = min(x1, max(p[0] for p in ink) + 2)
        iy0 = max(y0, min(p[1] for p in ink) - 2)
        iy1 = min(y1, max(p[1] for p in ink) + 2)
        fill = tuple(int(statistics.median(p[2][i] for p in bg)) for i in range(3))
        ImageDraw.Draw(img).rectangle([ix0, iy0, ix1, iy1], fill=fill)
    font = ImageFont.truetype(font_path, label.get("size", 14))
    d = ImageDraw.Draw(img)
    l, t, r, b = d.textbbox((0, 0), label["text"], font=font)
    cx = (x0 + x1 - (r - l)) // 2 - l
    cy = (y0 + y1 - (b - t)) // 2 - t
    d.text((cx + 1, cy + 1), label["text"], font=font, fill=(0, 0, 0))
    d.text((cx, cy), label["text"], font=font, fill=label.get("color", (235, 228, 210)))


def main():
    spec_path, src, dst = sys.argv[1:4]
    with open(spec_path, encoding="utf-8") as f:
        spec = json.load(f)
    img = Image.open(src).convert("RGB")
    for label in spec["labels"]:
        relabel(img, label, spec["font"], spec.get("threshold", 120))
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
