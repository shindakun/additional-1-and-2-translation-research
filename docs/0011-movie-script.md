# 0011: Movie script

Narration text of the movies, with draft English and timing. The text is burned into the video ([0006](0006-movies.md)). The subtitle patch ([0010](0010-subtitle-patch.md)) draws the English.

## Method

- Text: one full-resolution frame per narration block, read by eye.
- Timing: the counter build ([0010](0010-subtitle-patch.md)) in headless PPSSPP, one screenshot each emulated second. The counter maps emulated time to video frames; a block starts at the first frame with its text and ends at the first frame without it. Times are seconds from the start of the video, ±1 s. An earlier pixel-count method gave times up to 6 s late for blocks 4 to 11. The subtitle tables in `research/subs` start each block 0.3 s before the times below.

The English is a draft. Names are not final: `ヴァータイト` is Verdite, `リキストリア` is Rikistria, `略奪王` is the Plunderer King.

## Additional I, movie001 (New Game opening)

`movie000` has the same narration after the FromSoftware logo (1.0 to 4.0 s); its times are later by the length of the logo part. The times below are for `movie001`, measured in game.

| # | Start | End | Japanese | English draft |
|---|---|---|---|---|
| 1 | 7.2 | 12.7 | ヴァータイトの騒乱から10年・・・ | Ten years after the turmoil in Verdite... |
| 2 | 16.7 | 22.7 | 人間という種を滅ぼそうとする強大な意志が / 世界から消えてなお / 人々の間から争いが消えることはなかった・・・ | The great will that sought to destroy humankind had left the world, / yet strife among people never ended... |
| 3 | 25.2 | 36.2 | 大陸の南方、リキストリアに、 / 聖地と名の付く迷宮がある。 / その迷宮には、かつて"略奪王"と呼ばれ、 / 民の手によって追われた王が身を隠しているという。 | In Rikistria, in the south of the continent, / lies a labyrinth called the Holy Land. / It is said that a king once called the Plunderer King, / driven out by his own people, hides inside it. |
| 4 | 43.7 | 52.2 | 王の失脚後、国威は衰え、 / 不満を覚えた民の間からは、 / 王の復権を望む声までが上がり始めていた。 | After the king's fall the nation weakened, / and among the discontented people / some began to call for his return. |
| 5 | 58.2 | 63.7 | 王の復権を恐れる者は、その死の証明を欲し / 多くの傭兵団を雇い聖地の探索を命じた。 | Those who feared his return wanted proof of his death. / They hired many mercenary bands to search the Holy Land. |
| 6 | 69.2 | 78.2 | ある傭兵団の一員に、一人の青年がいた。 / 戦争によって家族を失った青年にとって、 / 幼い自分を保護し育ててくれた傭兵団は、 / 第二の家族同然だった。 | One band had a young man among its members. / He had lost his family to war, / and the band that took him in as a child and raised him / was a second family to him. |
| 7 | 81.7 | 92.7 | 聖地に踏み込んだ傭兵団はあるとき、 / 迷宮の罠にかかった。 / 周囲を魔物に囲まれた傭兵団は / 混乱に陥りながらも撤退を図った。 | Deep in the Holy Land, the band / fell into a trap in the labyrinth. / Surrounded by monsters, / they tried to retreat in the chaos. |
| 8 | 84.7 | 92.7 | 何とか逃れ出た青年は、 / 深い傷の為に意識を失った。 | The young man barely escaped, / and his deep wounds left him unconscious. |
| 9 | 96.7 | 107.2 | 青年が再び意識を取り戻した時には、 / 1年もの歳月が経過していた。 / そして団が壊滅したこと、 / わずかに生き残った者も町を去ったことを知った。 | When he came to, / a whole year had passed. / He learned the band had been destroyed, / and the few survivors had left the town. |
| 10 | 111.7 | 119.2 | その後青年は町に出た際に、 / 傭兵団の生き残りとも亡霊ともつかない人物が / 聖地を彷徨っていたという噂を耳にした。 | Later, in town, he heard a rumor / of a figure wandering the Holy Land, / perhaps a survivor of the band, perhaps a ghost. |
| 11 | 123.7 | 131.2 | かつての仲間が未だ彷徨っているという噂の / 真偽を確かめる為に / 青年は、再び聖地へ踏み込むことを決意した。 | To learn whether his old comrades still wandered there, / the young man resolved / to enter the Holy Land once more. |

`/` marks a line break in the video. Block 8 appears while block 7 is still on screen.

## Other movies

Not yet transcribed: Additional I `movie001`, `movie010`, `movie011`; all seven Additional II movies.
