// Package sjis converts between Shift-JIS and UTF-8 with the game's own
// tables: jis2ucs.bin and ucs2jis.bin from AdvSys.bnd map JIS X 0208 codes
// to UCS-2 and back. Shift-JIS converts to JIS with a fixed formula.
package sjis

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Codec holds the two tables.
type Codec struct {
	jis2ucs, ucs2jis [65536]uint16
}

// New builds a codec from the two 0x20000-byte tables.
func New(jis2ucs, ucs2jis []byte) (*Codec, error) {
	if len(jis2ucs) != 0x20000 || len(ucs2jis) != 0x20000 {
		return nil, errors.New("sjis: tables must be 0x20000 bytes")
	}
	c := &Codec{}
	for i := range 65536 {
		c.jis2ucs[i] = binary.LittleEndian.Uint16(jis2ucs[i*2:])
		c.ucs2jis[i] = binary.LittleEndian.Uint16(ucs2jis[i*2:])
	}
	return c, nil
}

func lead(b byte) bool { return (b >= 0x81 && b <= 0x9F) || (b >= 0xE0 && b <= 0xEF) }

func toJIS(s1, s2 byte) uint16 {
	h := uint16(s1)
	if h <= 0x9F {
		h -= 0x70
	} else {
		h -= 0xB0
	}
	h <<= 1
	l := uint16(s2)
	if l < 0x9F {
		h--
		if l < 0x7F {
			l -= 0x1F
		} else {
			l -= 0x20
		}
	} else {
		l -= 0x7E
	}
	return h<<8 | l
}

func fromJIS(j uint16) (byte, byte) {
	h, l := j>>8, j&0xFF
	var s1 uint16
	if h <= 0x5E {
		s1 = (h+1)/2 + 0x70
	} else {
		s1 = (h+1)/2 + 0xB0
	}
	var s2 uint16
	if h%2 == 1 {
		s2 = l + 0x1F
		if s2 >= 0x7F {
			s2++
		}
	} else {
		s2 = l + 0x7E
	}
	return byte(s1), byte(s2)
}

// Decode converts Shift-JIS bytes to a string. Half-width katakana
// (0xA1-0xDF) map to U+FF61-U+FF9F.
func (c *Codec) Decode(b []byte) (string, error) {
	var sb strings.Builder
	for i := 0; i < len(b); i++ {
		switch x := b[i]; {
		case x < 0x80:
			sb.WriteByte(x)
		case x >= 0xA1 && x <= 0xDF:
			sb.WriteRune(rune(0xFF61 + int(x) - 0xA1))
		case lead(x) && i+1 < len(b):
			u := c.jis2ucs[toJIS(x, b[i+1])]
			if u == 0 {
				return "", fmt.Errorf("sjis: no character for %02X%02X", x, b[i+1])
			}
			sb.WriteRune(rune(u))
			i++
		default:
			return "", fmt.Errorf("sjis: bad byte %02X at %d", x, i)
		}
	}
	return sb.String(), nil
}

// Encode converts a string to Shift-JIS.
func (c *Codec) Encode(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for i, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case r >= 0xFF61 && r <= 0xFF9F:
			out = append(out, byte(0xA1+r-0xFF61))
		case r < 0x10000 && c.ucs2jis[r] != 0:
			s1, s2 := fromJIS(c.ucs2jis[r])
			out = append(out, s1, s2)
		case r == utf8.RuneError:
			return nil, fmt.Errorf("sjis: invalid UTF-8 at %d", i)
		default:
			return nil, fmt.Errorf("sjis: %q has no Shift-JIS code", r)
		}
	}
	return out, nil
}
