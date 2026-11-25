#!/usr/bin/env python3
"""Copy translations from an old translation file into a new export.

Usage: python3 -I scripts/text_merge.py OLD.json NEW.json OUT.json

Entries match by kind and source text. Entries of OLD with no match are
listed on stderr.
"""
import json
import sys

old_path, new_path, out_path = sys.argv[1:4]
with open(old_path, encoding="utf-8") as f:
    old = {(e["kind"], e["ja"]): e["en"] for e in json.load(f) if e["en"]}
with open(new_path, encoding="utf-8") as f:
    new = json.load(f)
used = set()
for e in new:
    k = (e["kind"], e["ja"])
    if k in old:
        e["en"] = old[k]
        used.add(k)
for k in sorted(set(old) - used):
    print(f"no match: {k[0]} {k[1]!r}", file=sys.stderr)
with open(out_path, "w", encoding="utf-8") as f:
    json.dump(new, f, ensure_ascii=False, indent=2)
    f.write("\n")
print(f"{out_path}: {len(used)} of {len(old)} translations carried over")
