# 0009: Test builds

Test builds are images for PPSSPP. They go to `research/builds`, which git ignores. Each build changes one thing so that a difference on screen has one cause.

Build `kfa` first:

```sh
go build -o kfa ./cmd/kfa
```

## VWF test

Question: does the engine use the CCM metrics ([0005](0005-fonts.md))?

```sh
scripts/build_vwf_test.sh kfaI
```

Output: `research/builds/kfaI-vwf-test.iso`. The script replaces 22 files in place: the 21 `Font/BitFont_GameNN.bnd` files and `Bnd/AdvMenu.bnd`, whose nested font has new metrics too. Only the CCM glyph entries change.

| Glyph | Original (left, width, advance) | Test build | Pen step |
|---|---|---|---|
| Space | 0, 8, 8 | 0, 8, 4 | 4 |
| `i` | 0, 8, 8 | −3, 8, 5 | 2 |
| `l` | 0, 8, 8 | −2, 8, 5 | 3 |
| Capitals, `m` | 0, 8, 8 | 0, 8, 8 | 8 |

Capitals keep their step because their ink fills the 8-pixel cell.

What to check:

1. The title menu: chunk 1 holds `NEW GAME` and `LOAD GAME` as text. If the menu draws them with these fonts, the gap between the words is narrower.
2. Any ASCII text with lowercase `i`, `l` or spaces: the letters sit closer together.
3. The game boots and Japanese text looks unchanged.

A narrower gap confirms the engine applies CCM metrics, so English can be proportional without code patches.
