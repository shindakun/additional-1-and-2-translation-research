# 0006: Movies

All movies have Japanese narration burned into the video. Staff credits and title cards are in English. `scripts/movie_sheets.sh` writes contact sheets to `research/movies`.

## Format

PSMF (`PSMF0014`). The 0x800-byte header holds the stream offset (`0x08`, big-endian) and size (`0x0C`), then stream and entry-point information. An MPEG program stream follows.

| Stream ID | Contents |
|---|---|
| `0xE0` | H.264 video: Main profile, level 2.1, 480 × 272, 29.97 fps, no B-frames |
| `0xBD`, substream `0x00` | ATRAC3plus audio |
| `0xBB` | System header |
| `0xBF` | Private stream 2 |
| `0xBE` | Padding |

`scripts/pmf_streams.py` counts the packets. All 11 movies parse to the last byte with this stream set.

ffmpeg decodes the video after the header is cut off:

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

The narration is part of the image. Translation needs new video with English text. The narration sits over moving art, so covering it needs a mask or a box behind the English text.

No open-source tool writes PSMF; ffmpeg reads it but does not write it. The audio sits in its own stream, so a remuxer can copy the `0xBD` packets and replace the `0xE0` payload with a new encode at the same profile and level. The header's stream and entry-point tables must then be rebuilt; their layout is not yet decoded.

## Subtitles

A code patch can add English subtitles over the existing video. [0010](0010-subtitle-patch.md) describes it.
