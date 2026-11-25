#!/usr/bin/env python3
"""Center English like its Japanese source in an index TSV.

Usage: python3 -I scripts/text_center.py EXPORT.json IN.tsv OUT.tsv

Some sources start with spaces that center the text in a fixed width. For
those, the English gets leading spaces so that its center falls on the same
column. Columns count Shift-JIS bytes: a full-width character is two columns,
a half-width space one. Other lines pass through.
"""
import json
import sys

export_path, in_path, out_path = sys.argv[1:4]
with open(export_path, encoding="utf-8") as f:
    export = json.load(f)


def cols(s):
    return sum(1 if ord(c) < 0x80 or 0xFF61 <= ord(c) <= 0xFF9F else 2 for c in s)


with open(in_path, encoding="utf-8") as f, open(out_path, "w", encoding="utf-8") as out:
    for line in f:
        line = line.rstrip("\n")
        if not line:
            continue
        idx, en = line.split("\t", 1)
        ja = export[int(idx)]["ja"]
        body = ja.lstrip(" 　")
        if body != ja and "\n" not in ja and not en.startswith(" "):
            center = cols(ja) - cols(body) + cols(body) / 2
            en = " " * max(0, round(center - len(en) / 2)) + en
        out.write(f"{idx}\t{en}\n")
