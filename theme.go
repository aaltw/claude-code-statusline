package main

import (
	"os"
	"strings"
)

// Theme controls separator glyphs and the color palette used for rendering.
type Theme struct {
	SepRight  string // separator between left-side segments (points right)
	SepLeft   string // separator between right-side segments (points left)
	InvertSep bool   // swap separator fg/bg — needed when the glyph's filled area faces the opposite direction (e.g. slantr)
	Colors    Palette
}

// Predefined themes. Add new ones here and wire them in activeTheme().
var (
	ThemeArrow = Theme{SepRight: "\ue0b0", SepLeft: "\ue0b2", Colors: CatppuccinMocha}
	ThemeRound = Theme{SepRight: "\ue0b4", SepLeft: "\ue0b6", Colors: CatppuccinMocha}
	ThemeSlant  = Theme{SepRight: "\ue0b8", SepLeft: "\ue0ba", Colors: CatppuccinMocha}
	ThemeSlantR = Theme{SepRight: "\ue0ba", SepLeft: "\ue0b8", InvertSep: true, Colors: CatppuccinMocha}
	ThemeFlame  = Theme{SepRight: "\ue0bc", SepLeft: "\ue0be", Colors: CatppuccinMocha}
	ThemePlain  = Theme{SepRight: " │", SepLeft: "│ ", Colors: CatppuccinMocha}
)

// activeTheme reads STATUSLINE_THEME and returns the matching theme.
// Defaults to ThemeArrow when the env var is unset or unrecognized.
func activeTheme() Theme {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STATUSLINE_THEME"))) {
	case "round":
		return ThemeRound
	case "slant":
		return ThemeSlant
	case "slantr":
		return ThemeSlantR
	case "flame":
		return ThemeFlame
	case "plain":
		return ThemePlain
	default:
		return ThemeArrow
	}
}
