#!/usr/bin/env python3
"""Convert between English-only translation files and full ones.

The repository keeps translations without Japanese source text
(translations/GAME/text.json). A text entry is found by a key: the first 12
hex digits of SHA-1 over kind, a zero byte and the Japanese, so a fresh
export from the disc gives the Japanese back. See docs/kfa1/0004-text.md.

Usage:
  python3 -I scripts/translation.py strip FULL.json OUT.json
      Write the translated entries of a full translation file (the format of
      kfa text export) as key, kind, where, max and en.
  python3 -I scripts/translation.py fill EXPORT.json PUBLIC.json OUT.json
      Copy the English of PUBLIC.json into a fresh export by key. Lists keys
      that the export does not have.
  python3 -I scripts/translation.py strip-fmg FULL.json OUT.json
      Drop the Japanese of an FMG translation file (kfa fmg import matches
      messages by file and ID, so the result imports as it is).
"""
import hashlib
import json
import sys


def key(kind, ja):
    return hashlib.sha1(kind.encode() + b"\0" + ja.encode("utf-8")).hexdigest()[:12]


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def save(path, data):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=1)
        f.write("\n")


cmd = sys.argv[1] if len(sys.argv) > 1 else ""
if cmd == "strip":
    out = [{"key": key(e["kind"], e["ja"]), "kind": e["kind"], "where": e["where"][:1],
            "max": e["max"], "en": e["en"]} for e in load(sys.argv[2]) if e["en"]]
    out.sort(key=lambda e: (e["where"], e["key"]))
    save(sys.argv[3], out)
    print(f"{sys.argv[3]}: {len(out)} entries")
elif cmd == "fill":
    export, public = load(sys.argv[2]), load(sys.argv[3])
    en = {e["key"]: e["en"] for e in public}
    used = set()
    for e in export:
        k = key(e["kind"], e["ja"])
        if k in en:
            e["en"] = en[k]
            used.add(k)
    save(sys.argv[4], export)
    for k in sorted(set(en) - used):
        print(f"not in export: {k}", file=sys.stderr)
    print(f"{sys.argv[4]}: {len(used)} of {len(en)} translations")
elif cmd == "strip-fmg":
    out = [{"file": e["file"], "id": e["id"], "en": e["en"]} for e in load(sys.argv[2]) if e["en"]]
    save(sys.argv[3], out)
    print(f"{sys.argv[3]}: {len(out)} messages")
else:
    sys.exit(__doc__)
