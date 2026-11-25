# 0005: Movies, Additional II

The movie format, the subtitle patch and the method are in [kfa1/0006](../kfa1/0006-movies.md), [kfa1/0010](../kfa1/0010-subtitle-patch.md) and [kfa1/0011](../kfa1/0011-movie-script.md). The tables are in `research/subs/kfaII-movieNNN.tsv`; `scripts/build_english.sh kfaII` applies them.

| Movie | Length | Content |
|---|---|---|
| `movie000` | 1:58 | FromSoftware logo, then the opening narration; plays at boot |
| `movie201` | 1:54 | The opening narration without the logo; the `movie000` narration 5.0 s earlier. Not seen in a test run |
| `movie202`, `movie203` | 1:02, 1:06 | The labyrinth's history; the two differ in the last blocks. New Game plays `movie203` |
| `movie210`, `movie211`, `movie212` | 2:30, 2:06, 2:36 | Three endings, then the credits in English |

Times are seconds from the first frame. Each block starts 1 s before the Japanese and ends about 1 s after it. The opening times come from a 10 fps edge count ([kfa1/0011](../kfa1/0011-movie-script.md)); the others from one frame every 2 s, ±1 s. `/` marks a line break.

The English is a draft. Names follow [kfa1/0004](../kfa1/0004-text.md#proper-names).

## movie201 (opening)

| Start | End | English |
|---|---|---|
| 1.5 | 8 | Ten-odd years after the death of the king of Rikistria... |
| 11.1 | 23.7 | The disasters that struck again and again / were enough to sow new fear in a people / who had found brief relief at the king's death... |
| 26.6 | 34.7 | Traujito, the town of ruins in Rikistria, / once the scene of turmoil, / fell to a great earthquake. |
| 38.7 | 50.7 | The town lay buried in rubble. / Those who knew the turmoil feared the king had returned, / and some even began to say they had seen him. |
| 53.9 | 65.7 | The Republic's council took stock of the damage / and sent a survey team back into the labyrinth, / sealed for ten-odd years. |
| 67.1 | 77.8 | The survey leader had one family member: his son. / An envoy sent by the state came to the son. |
| 83.2 | 96.2 | The envoy said only that the survey leader / had died on duty in the labyrinth. / The young man pressed him for details, / but the envoy left without a word. |
| 100.5 | 109.8 | The next morning, the young man left home. / His steps led to Traujito, the Holy Land... |

`movie000` uses the same lines 5 s later.

## movie202

| Start | End | English |
|---|---|---|
| 1 | 11 | The labyrinth of Traujito, where the king of Rikistria, / once driven out by his people, hid himself. |
| 11 | 19 | Fearing the living king, Rikistria's central council / sent many mercenaries into the labyrinth. |
| 19 | 33 | Most mercenaries died in the labyrinth. But a nameless one / of a ruined band went in alone later and slew the king. |
| 33 | 43 | In the labyrinth slept two holy swords / stolen from the kingdom of Verdite. |
| 43 | 53 | The nameless mercenary found a holy sword here / and returned it to an envoy from Verdite. |
| 53 | 62 | One was sealed in the vault of Verdite, / but the other has not yet been found. |

## movie203

| Start | End | English |
|---|---|---|
| 1 | 11 | The labyrinth of Traujito, where the king of Rikistria, / once driven out by his people, hid himself. |
| 11 | 19 | Fearing the living king, Rikistria's central council / sent many mercenaries into the labyrinth. |
| 19 | 33 | Most mercenaries died in the labyrinth. But a nameless one / of a ruined band went in alone later and slew the king. |
| 33 | 45 | It was rumored that two holy swords / stolen from Verdite lay in the labyrinth. |
| 45 | 55 | After the king was slain, knights came often / to search the labyrinth, but none came back alive. |
| 55 | 65 | And neither holy sword has been found. |

## movie210

| Start | End | English |
|---|---|---|
| 4 | 17 | The earth fell still, and quiet returned to the labyrinth. / With Dark Slayer, its light gone, in his hand, / the young man left the labyrinth. |
| 21 | 37 | On the survey team's report, the Republic's council / blamed the disasters on magic stored in the labyrinth, / and reported that the situation was settling. |
| 39 | 51 | In Traujito the rebuilding was already under way. / The young man set out for his homeland. / People paused in their work to see him off. |
| 55 | 67 | And once again, / the labyrinth was sealed. |

## movie211

| Start | End | English |
|---|---|---|
| 5 | 15 | When all was over, / the place called the Holy Land / had become a land thick with foul air. |
| 15 | 23 | No living soul was left there. / People feared the curse of the king of Rikistria, / and the Republic gave up its claim to the land. |
| 25 | 37 | No one knows the truth of what happened here, / nor what caused it. |

## movie212

| Start | End | English |
|---|---|---|
| 1 | 13 | After a great rumbling, the labyrinth collapsed. / Earthquakes never struck the town of Traujito again. |
| 15 | 23 | The labyrinth was lost, / and the history of Rikistria came to its end. |
| 27 | 39 | On the survey team's report, the Republic's council / blamed the disasters on magic stored in the labyrinth, / and reported that the situation was settling. |
| 43 | 57 | Some time after the collapse, a young man pulled / from the rubble woke. The people were relieved / he was safe, and the town was filled with joy. |
| 59 | 68 | The two holy swords the young man held in the rubble / were then sealed in the vault of the Verdite royal house. |

## Test

The 96 lines of the seven tables do not fit the first code space of the patch. The text that does not fit goes to the second code space, after the chunk base stubs ([kfa1/0010](../kfa1/0010-subtitle-patch.md)).

Headless PPSSPP: `movie000` at boot shows every block in English clear of the Japanese, then the title. New Game plays `movie203` with its subtitles in place, then the town. `movie201`, `movie202` and the endings are not reached in a test run.
