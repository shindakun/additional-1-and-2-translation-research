#!/usr/bin/env python3
"""Convert item and spell descriptions in global text to English.

Usage: python3 -I scripts/item_desc.py   (run from the repository root)

Reads research/text/global.tsv, research/text/items.tsv (names),
/tmp/desc.txt and research/text/desc-en.tsv (free-text lines). Writes
research/text/global3-en.tsv. Header lines get the English name padded to
the Japanese column; stat lines (starting with 斬) get two-letter codes:
Cu Hi St Fi Ea Wi Wa Li Da (docs/0004-text.md).
"""
import json,re
names=dict(l.rstrip('\n').split('\t') for l in open('research/text/items.tsv',encoding='utf-8'))
extra={'スペアキー':'Spare Key','傷薬':'Salve','兵士の腕章':"Soldier's Armband",'兵士の鍵':"Soldier's Key",'スコップ':'Shovel','兵士のネックレス':"Soldier's Necklace",'白珠の石版':'White Pearl Tablet','小さな指輪':'Small Ring','鉄格子の鍵':'Iron Gate Key','王家の紋章':'Royal Crest','リキストリアの鍵':'Rikistria Key','火の竜王石':'Fire Dragon Stone','水の竜王石':'Water Dragon Stone','風の竜王石':'Wind Dragon Stone','土の竜王石':'Earth Dragon Stone','光の鍵':'Key of Light','闇の鍵':'Key of Darkness','兵士の地図':"Soldier's Map",'リキストリアの地図':'Map of Rikistria','イシリウスの地図':'Map of Isilius','記憶の杖':'Staff of Memory','烙印の指輪':'Brand Ring','ストック':'Stock','トルネード':'Tornado','傭兵団のペンダント':'Mercenary Pendant','子供の指輪':"Child's Ring",'ショートソード':'Short Sword'}
names.update(extra)
desc_ja=[l.rstrip('\n').split('\t',1)[1] for l in open('research/text/desc-ja.tsv',encoding='utf-8')]
desc_en=dict(l.rstrip('\n').split('\t',1) for l in open('research/text/desc-en.tsv',encoding='utf-8'))
desc={d:desc_en[str(k)] for k,d in enumerate(desc_ja)}
def cols(s): return sum(1 if ord(c)<0x80 or 0xFF61<=ord(c)<=0xFF9F else 2 for c in s)
stat=dict(zip('斬殴刺火土風水光闇',['Cu','Hi','St','Fi','Ea','Wi','Wa','Li','Da']))
def head(line):
    m=re.match(r'^(#m\[[0-9a-f]+\])?(.*)$',line); icon=m.group(1) or ''; rest=m.group(2)
    key=re.search(r'(耐久度|矢本数|所持数|重量)',rest)
    nm_pad=rest[:key.start()] if key else rest
    nm=nm_pad.rstrip('　 ')
    if nm not in names: return None
    tail=rest[key.start():] if key else ''
    tail=tail.replace('耐久度','Dur').replace('％','').replace('矢本数','Arrows').replace('本','').replace('所持数','Owned').replace('重量','Wt').replace('　','  ')
    w=cols(nm_pad) if key else 0
    return icon+names[nm].ljust(w)+tail
def line2(line):
    if not re.match(r'^[　 ]*斬', line): return None
    out=line.replace('　','  ')
    for k,v in stat.items(): out=out.replace(k,v)
    return out
rows=[l.rstrip('\n').split('\t') for l in open('research/text/global.tsv',encoding='utf-8')][330:]
out=open('research/text/global3-en.tsv','w'); bad=[]
for i,mx,w,ja in rows:
    res=[]
    for ln in ja.split('|'):
        if ln=='': res.append(''); continue
        r=desc.get(ln) or line2(ln) or head(ln)
        if r is None: bad.append((i,ln)); break
        res.append(r)
    else:
        out.write(f"{i}\t{'|'.join(res)}\n")
out.close(); print('bad',len(bad)); [print(b) for b in bad[:20]]
