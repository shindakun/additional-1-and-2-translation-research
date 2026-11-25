package font

// Proportional sets left offset and advance for the printable ASCII glyphs
// (0x21-0x7E) from their ink, so the pen moves by ink width plus gap pixels.
// The drawn width and texture coordinates are unchanged, so glyphs are not
// scaled. Space (0x20) gets spaceAdvance. Glyphs whose page is not
// covered by the given layers keep their metrics. It returns the number of
// glyphs changed.
func (c *CCM) Proportional(tex [2]*Texture, plt []byte, gap, spaceAdvance int16) int {
	var alpha [4][256]byte
	for l := range alpha {
		alpha[l] = LayerAlpha(plt, l)
	}
	n := 0
	for code := uint32(0x20); code <= 0x7E; code++ {
		i, ok := c.Index(code)
		if !ok || i >= len(c.Glyphs) {
			continue
		}
		g := &c.Glyphs[i]
		if code == 0x20 {
			g.Left, g.Advance = 0, spaceAdvance
			n++
			continue
		}
		if g.Page < 0 || g.Page > 7 {
			continue
		}
		first, last, ok := InkColumns(tex[g.Page/4], alpha[g.Page%4], *g)
		if !ok {
			continue
		}
		// FontDrawChar draws at pen+Left and leaves the pen at pen+Left+Advance.
		g.Left = -int16(first)
		g.Advance = int16(last+1) + gap
		n++
	}
	return n
}
