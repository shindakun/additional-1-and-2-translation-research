#!/usr/bin/env python3
"""Set translations from a TSV into a translation file.

Usage: python3 -I scripts/text_apply_tsv.py EXPORT.json IN.tsv TARGET.json

Each TSV line is INDEX<TAB>ENGLISH. INDEX is the entry's position in
EXPORT.json (the output of kfa text export); "|" in ENGLISH is a line break.
The entry is found in TARGET.json by kind and source text, so TARGET.json can
be an older or merged file.
"""
import json
import sys

export_path, tsv_path, target_path = sys.argv[1:4]
with open(export_path, encoding="utf-8") as f:
    export = json.load(f)
with open(target_path, encoding="utf-8") as f:
    target = json.load(f)
pos = {(e["kind"], e["ja"]): i for i, e in enumerate(target)}
n = 0
with open(tsv_path, encoding="utf-8") as f:
    for line in f:
        line = line.rstrip("\n")
        if not line:
            continue
        idx, en = line.split("\t", 1)
        src = export[int(idx)]
        i = pos.get((src["kind"], src["ja"]))
        if i is None:
            sys.exit(f"{idx}: {src['ja']!r} not in {target_path}")
        target[i]["en"] = en.replace("|", "\n")
        n += 1
with open(target_path, "w", encoding="utf-8") as f:
    json.dump(target, f, ensure_ascii=False, indent=2)
    f.write("\n")
print(f"{target_path}: {n} translations set")
