# 0003: Image text, Additional II

The image formats and tools are in [kfa1/0012](../kfa1/0012-images.md).

91 of the 475 unique images have the same bytes as an Additional I image. The images edited for Additional I ([kfa1/0012](../kfa1/0012-images.md)) have counterparts with other bytes; a comparison of 32 × 32 grayscale thumbnails finds them:

| Additional I | Additional II | Mean difference (0–255) |
|---|---|---|
| `c000_20.jpg`, `c000_19.jpg` (town maps) | `c000_15.jpg`, `c000_14.jpg` | 9, 11: the town is in ruins; the labels are at the same places |
| `c009_04.jpg` (status screen) | `c011_04.jpg` | 0.9 |
| `c004_04.jpg`, `c004_06.jpg` (roulette) | `c004_04.jpg`, `c004_06.jpg` | 0 |
| `c000_00.gim` (shop panels) | `c000_00.gim` | 0.3 |
| `g_02aa00.gim` (HUD atlas) | `g_02afe0.gim` | 0.1 |

The Additional I specs apply with two changes on the full town map: the Inn label is 4 pixels further right and the Smithy label is 4 pixels wider (pixel diff of the two maps). `translations/kfaII/img` holds the specs; each budget is the size of the Additional II file. The Additional II JPEGs are more compressed: the edits fit at quality 78 to 85.

Test: in headless PPSSPP the town map, the status overlay and the HUD labels show English.

## New casino games

Memory (chunk 5) uses the card sheets `c002_00` to `c002_05` and the table `c005_06`. They have no text.

Slots (chunk 9) has Japanese on three images. The specs are in `translations/kfaII/img`:

| Image | Japanese | English | Spec |
|---|---|---|---|
| `c009_11.jpg` (screen) | `賭け金`, `所持金`, `配当表` | Bet, Gold, Payout | `slot-screen.json` |
| `c009_00.png` (side panel) | `賭け金`, `所持金` | Bet, Gold | `slot-panel.json` |
| `c009_01.png` (button strip) | `配当表`, vertical | PAYOUT, turned to read top to bottom like `EXIT` | `slot-buttons.json` |

The pay table `c009_12.jpg` is in English on the disc. A label with `rotate` is drawn on its own layer and turned ([kfa1/0012](../kfa1/0012-images.md)).

Headless PPSSPP at the start of a game: the tavern dealer lists Roulette, Poker, High & Low, Blackjack and Memory. Slots is not in the list, so the slot images are not seen in a test run.
