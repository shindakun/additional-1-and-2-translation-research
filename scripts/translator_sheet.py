#!/usr/bin/env python3
"""Write a translator sheet (CSV) for a game, and read the edited sheet back.

Usage:
  python3 -I scripts/translator_sheet.py export GAME OUT.csv
  python3 -I scripts/translator_sheet.py import GAME IN.csv

GAME is kfaI or kfaII. Both need the extracted disc in
research/extract/GAME/PSP_GAME and bin/kfa (make kfa).

export writes one row per line of text a player can see: main.dat text,
names and menus in scene order, then the FMG system messages. Columns:

  id        Do not change. Finds the line again on import.
  where     Chunk, scene and line (c012/s00345/t0006), g/ for global text,
            or the FMG file and message ID.
  kind      text, name, menu, block or fmg.
  max       Bytes in the original slot. Longer English still fits (the
            build moves it), except for kind name, menu and block.
  japanese  The source line from the disc.
  english   The translation. Edit only this column.

The file is UTF-8 with a byte order mark, so Excel and Google Sheets show
the Japanese. Flag names and placeholder text are left out. The sheet holds
Japanese from the disc: do not commit it.

import reads the english column into translations/GAME/text.json and
fmg.json. An empty cell removes the translation. A cell that lost its
trailing line breaks gets back as many as the old English had (or the
Japanese, for a new line). It warns about dialogue lines over 52 characters,
and about English with more than 3 lines and more lines than the Japanese
(docs/kfa1/0004-text.md). It also lists changed lines the build will not
write: a character with no Shift-JIS code (for example é), or a
name, menu or block over its max bytes.
"""
import csv
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Dev and unused strings that no player sees.
PLACEHOLDER = re.compile(
    r"^(装飾品\d+|貴重品\d+|テキスト\d*|削除|\d+G?|コマンド0|Dummy|（未使用）|エラーテキスト|"
    r"テスト|（予備）|選択肢|NEW GAME|LOAD GAME|[○×./]|/12|)$|^(↓|◆|削除|ULJS)")
COLUMNS = ["id", "where", "kind", "max", "japanese", "english"]


def key(kind, ja):

    return hashlib.sha1(kind.encode() + b"\0" + ja.encode("utf-8")).hexdigest()[:12]


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def save(path, data):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=1)
        f.write("\n")


def exports(game, tmp):
    """Export the main.dat text and the FMG messages from the disc."""
    kfa = os.path.join(ROOT, "bin", "kfa")
    gamedir = os.path.join(ROOT, "research", "extract", game, "PSP_GAME")
    text, fmg = os.path.join(tmp, "text.json"), os.path.join(tmp, "fmg.json")
    for cmd, out in (("text", text), ("fmg", fmg)):
        subprocess.run([kfa, cmd, "export", gamedir, out], check=True, stdout=subprocess.DEVNULL)
    return load(text), load(fmg)


def trailing(s):
    return len(s) - len(s.rstrip("\n"))


def visible(e):
    return e["kind"] != "flag" and not PLACEHOLDER.match(e["ja"].strip(" 　\n"))


def export(game, out):
    tr = os.path.join(ROOT, "translations", game)
    with tempfile.TemporaryDirectory() as tmp:
        text, fmg = exports(game, tmp)
    en = {e["key"]: e["en"] for e in load(os.path.join(tr, "text.json"))}
    fen = {(e["file"], e["id"]): e["en"] for e in load(os.path.join(tr, "fmg.json"))}
    rows = []
    for e in text:
        if visible(e):
            k = key(e["kind"], e["ja"])
            rows.append([k, " ".join(e["where"][:3]), e["kind"], e["max"], e["ja"], en.get(k, "")])
    rows.sort(key=lambda r: r[1])
    for e in fmg:
        rows.append([f"fmg:{e['file']}:{e['id']}", f"{e['file']} {e['id']}", "fmg", "",
                     e["ja"], fen.get((e["file"], e["id"]), "")])
    with open(out, "w", encoding="utf-8-sig", newline="") as f:
        w = csv.writer(f)
        w.writerow(COLUMNS)
        w.writerows(rows)
    done = sum(1 for r in rows if r[5])
    print(f"{out}: {len(rows)} rows, {done} translated")


def warn(row, en, ja):
    try:
        size = len(en.replace("\n", "\r\n").encode("cp932"))
    except UnicodeEncodeError as err:
        print(f"not written {row['where']}: {en[err.start]!r} has no Shift-JIS code", file=sys.stderr)
        return
    if row["kind"] != "text":
        if row["max"] and size > int(row["max"]):
            print(f"not written {row['where']}: {size} bytes, room for {row['max']}", file=sys.stderr)
        return
    lines = [ln for ln in en.split("\n") if ln.strip()]
    jlines = [ln for ln in ja.split("\n") if ln.strip()]
    long = [ln for ln in lines if len(ln) > 52 and not ln.startswith("#")]
    if long or len(lines) > max(3, len(jlines)):
        print(f"check {row['where']}: {len(lines)} lines, longest {max(map(len, lines))}",
              file=sys.stderr)


def import_(game, src):
    tr = os.path.join(ROOT, "translations", game)
    with tempfile.TemporaryDirectory() as tmp:
        text, fmg = exports(game, tmp)
    with open(src, encoding="utf-8-sig", newline="") as f:
        rows = list(csv.DictReader(f))
    if not rows or set(COLUMNS) - set(rows[0]):
        sys.exit(f"{src}: needs the columns {', '.join(COLUMNS)}")

    old = {e["key"]: e for e in load(os.path.join(tr, "text.json"))}
    oldf = {(e["file"], e["id"]): e["en"] for e in load(os.path.join(tr, "fmg.json"))}
    byk = {}
    for e in text:
        byk.setdefault(key(e["kind"], e["ja"]), e)
    fja = {(e["file"], e["id"]): e["ja"] for e in fmg}
    changed = 0
    for r in rows:
        en = r["english"]
        if r["id"].startswith("fmg:"):
            _, file, mid = r["id"].split(":")
            k = (file, int(mid))
            if k not in fja:
                print(f"unknown id {r['id']}", file=sys.stderr)
                continue
            if en != oldf.get(k, ""):
                changed += 1
            if en:
                oldf[k] = en
            else:
                oldf.pop(k, None)
            continue
        e = byk.get(r["id"])
        if e is None:
            print(f"unknown id {r['id']}", file=sys.stderr)
            continue
        prev = old.get(r["id"], {}).get("en", "")
        if en and not trailing(en):
            en += "\n" * trailing(prev if prev else e["ja"])
        if en != prev:
            changed += 1
            if en:
                warn(r, en, e["ja"])
        if en:
            old[r["id"]] = {"key": r["id"], "kind": e["kind"], "where": e["where"][:1],
                            "max": e["max"], "en": en}
        else:
            old.pop(r["id"], None)

    out = sorted(old.values(), key=lambda e: (e["where"], e["key"]))
    save(os.path.join(tr, "text.json"), out)
    order = {(e["file"], e["id"]): i for i, e in enumerate(fmg)}
    fout = [{"file": f, "id": i, "en": t} for (f, i), t in sorted(oldf.items(), key=lambda kv: order.get(kv[0], 1 << 30))]
    save(os.path.join(tr, "fmg.json"), fout)
    print(f"{src}: {changed} lines changed; translations/{game} has {len(out)} text and {len(fout)} FMG translations")


if len(sys.argv) != 4 or sys.argv[1] not in ("export", "import"):
    sys.exit(__doc__)
(export if sys.argv[1] == "export" else import_)(sys.argv[2], sys.argv[3])
