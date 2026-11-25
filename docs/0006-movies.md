# 0006: Movies

All movies have Japanese narration burned into the video. Staff credits and title cards are in English. `scripts/movie_sheets.sh` writes contact sheets to `research/movies`.

## Format

PSMF (`PSMF0014`). A 0x800-byte header precedes an MPEG program stream. Video is H.264, 480 × 272, 29.97 fps. ffmpeg decodes the stream after the header is cut off:

```sh
tail -c +2049 movie000.pmf | ffmpeg -f mpeg -i - out.mp4
```

## Files

| Game | File | Length | Contents |
|---|---|---|---|
| I | `movie000.pmf` | 2:21 | FromSoftware logo, opening narration |
| I | `movie001.pmf` | 2:16 | Opening narration with a closing line, no logo |
| I | `movie010.pmf` | 2:08 | Ending, title card, credits |
| I | `movie011.pmf` | 2:03 | Second ending, title card, credits |
| II | `movie000.pmf` | 1:58 | FromSoftware logo, opening narration |
| II | `movie201.pmf` | 1:54 | Opening narration, no logo |
| II | `movie202.pmf` | 1:03 | Labyrinth history; one sword still lost |
| II | `movie203.pmf` | 1:06 | Labyrinth history; both swords lost |
| II | `movie210.pmf` | 2:29 | Ending, title card, credits |
| II | `movie211.pmf` | 2:06 | Second ending, title card, credits |
| II | `movie212.pmf` | 2:36 | Third ending, title card, credits |

## Translation

The narration is part of the image. Translation needs new video with English text and a PSMF muxer for the result. The narration sits over moving art, so covering it needs a mask or a box behind the English text.
