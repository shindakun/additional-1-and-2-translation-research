"""List the MPEG program stream packets in a PSMF (.pmf) movie.

Usage: python3 -I scripts/pmf_streams.py FILE.pmf [...]
Prints the stream offset and size from the PSMF header, the pack count, and
the PES packet count per stream ID (and per substream of private stream 0xBD).
"""
import collections
import struct
import sys

for path in sys.argv[1:]:
    d = open(path, "rb").read()
    start, size = struct.unpack_from(">II", d, 8)
    o, packs = start, 0
    streams, subs = collections.Counter(), collections.Counter()
    while o + 4 <= len(d):
        if d[o:o + 4] == b"\x00\x00\x01\xba":
            packs += 1
            o += 14 + (d[o + 13] & 7)
            continue
        if d[o:o + 3] != b"\x00\x00\x01":
            break
        sid, length = d[o + 3], struct.unpack_from(">H", d, o + 4)[0]
        streams[sid] += 1
        if sid == 0xBD:
            subs[d[o + 9 + d[o + 8]]] += 1
        o += 6 + length
    print(f"{path}: magic {d[:8].decode()}, stream at {start:#x} size {size:#x}, packs {packs}, "
          f"parsed to {o:#x} of {len(d):#x}")
    print("  streams " + ", ".join(f"{k:#04x}: {v}" for k, v in sorted(streams.items()))
          + "; 0xbd substreams " + ", ".join(f"{k:#04x}: {v}" for k, v in sorted(subs.items())))
