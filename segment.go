package main

import "strings"

// Segment is one visual block in the statusline.
type Segment struct {
	BG       Color
	Text     string // may contain ANSI codes; visibleWidth strips them for measurement
	NoSep    bool   // skip the separator before this segment (right-side only)
	SepColor *Color // override separator fg color; defaults to segment BG
}

func renderLeft(segs []Segment, theme Theme, b *strings.Builder) {
	for i, s := range segs {
		b.WriteString(s.BG.BG())
		b.WriteString(s.Text)
		if i+1 < len(segs) {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(segs[i+1].BG.BG())
			b.WriteString(theme.SepRight)
		} else {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(theme.SepRight)
			b.WriteString(reset)
		}
	}
}

func renderRight(segs []Segment, theme Theme, b *strings.Builder) {
	for i, s := range segs {
		if i == 0 {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(theme.SepLeft)
		} else if s.NoSep {
			// intentionally no separator
		} else {
			sepFG := s.BG
			if s.SepColor != nil {
				sepFG = *s.SepColor
			}
			b.WriteString(sepFG.FG())
			b.WriteString(segs[i-1].BG.BG())
			b.WriteString(theme.SepLeft)
		}
		b.WriteString(s.BG.BG())
		b.WriteString(s.Text)
	}
	b.WriteString(reset)
}

func segmentsWidth(segs []Segment) int {
	w := 0
	for _, s := range segs {
		w += visibleWidth(s.Text)
	}
	w += len(segs) // one separator per segment
	return w
}
