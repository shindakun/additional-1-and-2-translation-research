# 0004: Translation and builds, Additional II

The text, FMG and build tools are in [kfa1/0004](../kfa1/0004-text.md), [kfa1/0013](../kfa1/0013-fmg.md) and [kfa1/0009](../kfa1/0009-test-builds.md).

## First build

`scripts/text_merge.py` carries 1,533 of the 2,385 Additional I translations into `research/text/kfaII-test.json`; the other 852 have no matching entry in Additional II. 180 FMG messages carry over by ID; the five that differ get `KFA II`. `scripts/build_english.sh kfaII` builds `research/builds/kfaII-english.iso` ([kfa1/0009](../kfa1/0009-test-builds.md)): 39,200 slots in place, 947 strings moved, 63 chunks relocated, nothing refused. With no subtitle tables, the build skips the subtitle patch and applies only the chunk base patch.

Headless PPSSPP: the town and the status overlay. Menu text from the carried-over English shows (`Status`, `Items`, `Magic`, `Camp`, `Short Sword`). With the seven edited images, the town map and the status overlay are English too.

## New entries

`scripts/gen_items.py` fills the found, used, priced and description entries from the item tables (165 entries, with 17 names and 15 description lines added to the tables). The new command, person and area names are done. The dialogue and message entries (1,298) go in batches of index-to-English TSV files, `research/text/kfaII-bN.tsv`. Many sources end in extra line breaks; `scripts/tail_breaks.py` copies the same number onto each English line. Batches 1 to 4 set 672 entries: town people, merchants, guards, the tavern, the casino rules (including Memory and Slots), the bard's songs and verses, the survey team, the scholar, the shop, and the first labyrinth scenes. 618 remain.

Test after batch 4: nothing refused. Headless PPSSPP shows the tavern owner's lines and the tavern menu (`Talk to the dealer`) in English.

## Open work

1. Translate the remaining dialogue and message entries.
2. Transcribe and subtitle the seven movies ([kfa1/0011](../kfa1/0011-movie-script.md)).
