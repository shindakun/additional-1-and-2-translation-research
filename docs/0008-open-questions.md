# 0008: Open questions

| # | Question | State |
|---|---|---|
| 1 | Layout of each action struct | Text fields known for `0x29` and `0x64` ([0004](0004-text.md)). Other actions: names known ([0007](0007-adventure-player.md)), fields not decoded. |
| 2 | Meaning of `#e[n]` | Answered: sound effect with parameters ([0004](0004-text.md)). The callback target is not traced. |
| 3 | Structure that holds the 0x20-byte name records | Answered: command and flag arrays ([0004](0004-text.md)). |
| 4 | Which flag names reach the screen | Answered: none in normal play; only debug screens ([0004](0004-text.md)). |
| 5 | Engine use of the CCM glyph metrics | Answered in code: offset, width and advance drive layout ([0005](0005-fonts.md)). Not yet seen on screen. |
| 6 | How a glyph selects its texture and layer | Answered: page field at `+0x16` ([0005](0005-fonts.md)). |
| 7 | PSMF muxing for re-encoded movies | Stream layout known; remux by video replacement is possible. PSMF header tables not decoded ([0006](0006-movies.md)). |
| 8 | Text limits in the King's Field build | Lower bound measured: 56 half-width units per line, 7 lines ([0004](0004-text.md)). Engine limit not known. |
