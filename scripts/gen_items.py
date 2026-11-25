#!/usr/bin/env python3
"""Fill item-based entries of a translation file from the name tables.

Usage: python3 -I scripts/gen_items.py TARGET.json

Reads research/text/items.tsv (item names, JA<TAB>EN) and
research/text/desc-ja.tsv with research/text/desc-en.tsv (description
lines by number). Fills empty entries of these forms in TARGET.json:

  XをJ手に入れた。        Got the X.
  Xを見つけた。\\nこれ以上は持てない。   Found the X.\\nYou can't carry any more.
  Xを使った。              Used the X.
  X    1200 (shop name)    the English name, padded, with the price
  item descriptions        header (name, Dur, Arrows, Owned, Wt), stat codes, text

Prints the item names and description lines that the tables lack.
"""
import json
import re
import sys

T = "research/text/"
names = dict(l.rstrip("\n").split("\t") for l in open(T + "items.tsv", encoding="utf-8") if "\t" in l)
dja = [l.rstrip("\n").split("\t", 1)[1] for l in open(T + "desc-ja.tsv", encoding="utf-8")]
den = dict(l.rstrip("\n").split("\t", 1) for l in open(T + "desc-en.tsv", encoding="utf-8") if "\t" in l)
desc = {d: den[str(k)] for k, d in enumerate(dja) if str(k) in den}
stat = dict(zip("斬殴刺火土風水光闇", ["Cu", "Hi", "St", "Fi", "Ea", "Wi", "Wa", "Li", "Da"]))
miss_names, miss_desc = set(), set()


def cols(s):
    return sum(1 if ord(c) < 0x80 or 0xFF61 <= ord(c) <= 0xFF9F else 2 for c in s)


def name(n):
    if n in names:
        return names[n]
    miss_names.add(n)
    return None


def head(line):
    m = re.match(r"^(#m\[[0-9a-f]+\])?(.*)$", line)
    icon, rest = m.group(1) or "", m.group(2)
    key = re.search(r"(耐久度|矢本数|所持数|重量)", rest)
    pad = rest[:key.start()] if key else rest
    en = name(pad.rstrip("　 "))
    if en is None:
        return None
    tail = rest[key.start():] if key else ""
    for a, b in (("耐久度", "Dur"), ("％", ""), ("矢本数", "Arrows"), ("本", ""), ("所持数", "Owned"), ("重量", "Wt"), ("　", "  ")):
        tail = tail.replace(a, b)
    return icon + en.ljust(cols(pad) if key else 0) + tail


def statline(line):
    if not re.match(r"^[　 ]*斬", line):
        return None
    out = line.replace("　", "  ")
    for k, v in stat.items():
        out = out.replace(k, v)
    return out


def describe(ja):
    res = []
    for ln in ja.split("\n"):
        if ln == "":
            res.append("")
            continue
        r = desc.get(ln) or statline(ln) or head(ln)
        if r is None:
            if not re.match(r"^(#m\[|[　 ]*斬)", ln):
                miss_desc.add(ln)
            return None
        res.append(r)
    return "\n".join(res)


def fill(x):
    ja = x["ja"]
    m = re.match(r"^(.+?)を手に入れた。(#e\[[0-9a-f]+\])?$", ja)
    if m:
        n = name(m.group(1))
        return n and f"Got the {n}.{m.group(2) or ''}"
    m = re.match(r"^(.+?)を見つけた。\nこれ以上は持てない。$", ja)
    if m:
        n = name(m.group(1))
        return n and f"Found the {n}.\nYou can't carry any more."
    m = re.match(r"^(.+?)を使った。$", ja)
    if m:
        n = name(m.group(1))
        return n and f"Used the {n}."
    if x["kind"] == "name":
        m = re.match(r"^(.*?)[ 　]+(\d+)$", ja)
        if m:
            n = name(m.group(1))
            if n:
                price = m.group(2)
                return n[:23 - len(price) - 1].ljust(23 - len(price)) + price
            return None
        return None
    if re.search(r"重量|耐久度|所持数", ja):
        return describe(ja)
    return None


path = sys.argv[1]
e = json.load(open(path, encoding="utf-8"))
n = 0
for x in e:
    if x["en"] or x["kind"] not in ("text", "name"):
        continue
    en = fill(x)
    if en:
        x["en"] = en
        n += 1
json.dump(e, open(path, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
print(f"{path}: {n} entries filled")
for m in sorted(miss_names):
    print("name\t" + m)
for m in sorted(miss_desc):
    print("desc\t" + m)
