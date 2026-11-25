#!/usr/bin/env python3
"""Copy trailing line breaks from the source to an index TSV.

Usage: python3 -I scripts/tail_breaks.py EXPORT.json IN.tsv OUT.tsv

Many sources end in one or more line breaks; the engine may use them to
pause or page the message window. Each English line gets the same number
of trailing breaks (written as |) as its source, after its own are removed.
"""
import json
import sys

export_path, in_path, out_path = sys.argv[1:4]
export = json.load(open(export_path, encoding="utf-8"))
with open(in_path, encoding="utf-8") as f, open(out_path, "w", encoding="utf-8") as out:
    for line in f:
        line = line.rstrip("\n")
        if not line:
            continue
        idx, en = line.split("\t", 1)
        ja = export[int(idx)]["ja"]
        n = len(ja) - len(ja.rstrip("\n"))
        out.write(f"{idx}\t{en.rstrip('|')}{'|' * n}\n")
