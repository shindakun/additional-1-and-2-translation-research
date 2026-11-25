# 0014: Debug screens

The engine has debug screens that the released game does not show. Their text is not translated.

## Debug mode

The u16 at `main.dat` file offset `0x82` holds header flags ([0003](0003-maindat.md#header)). Bit 2 selects the key set. The disc has 0 there. Setting bit 0, bit 1 or bit 3 turns debug mode on: each of the three gives the same result.

To set a bit: decrypt the header unit, change the word, encrypt and sign the file (`internal/maindat`: `Unpack`, `Pack`), then put the file into a copy of the ISO with `kfa iso put`. `kfa dat verify` passes on the result.

Test, headless PPSSPP: with any of the three bits set, after a load the screen shows `現在のシーンID - 94` (current scene ID) at the top left. The original disc with the same input shows nothing there. The title, the system menu and the controls screen do not change.

## Screens

The FMG messages name the debug screens ([0013](0013-fmg.md)):

| IDs | Screen |
|---|---|
| 501140, 501141 | View menu: `表示内容変更` (change view), `アプリに戻る` (back to the app) |
| 501145–501147 | Action log, with page change |
| 501150–501156 | Flag and item value list: flag ID, current value, flag name; page change; change by 10 |
| 501160 | Current scene ID |
| 501165, 501166 | Debug screen layout: moves the action log on screen |
| 501200 and after | Action names for the action log (`SetFlag`, `PutTextToWnd`, `SfxFadeIn`) |
| 501320–501350 | Action log markers (`<TRG>`, `<SCENE IN>`, `Cmd(%d)`) |

The flag list shows the 1,234 flag names of `main.dat` ([0004](0004-text.md)). `FUN_0006F90C` builds the view menu (FMG 501140, 501141).

Only the scene ID has been seen on screen. The input that opens the other screens is not known: in debug mode, Select, L, R, L+R, Triangle and Square in town and Start in the system menu open nothing new.

## Translation state

Not translated: the debug FMG messages above (136 messages) and the flag names. A build for play does not need them: debug mode is off on the disc.
