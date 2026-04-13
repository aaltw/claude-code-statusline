package main

import "fmt"

// Color holds an RGB triple and emits true-color ANSI escape codes.
type Color struct{ R, G, B int }

func (c Color) FG() string { return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B) }
func (c Color) BG() string { return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B) }

const reset = "\033[0m"

// Palette maps semantic color roles to RGB values.
// A Theme carries one Palette, allowing different color schemes.
type Palette struct {
	Base      Color
	Surface0  Color
	Surface1  Color
	Blue      Color
	Green     Color
	Green80   Color
	GreenDim  Color
	Yellow    Color
	Yellow80  Color
	YellowDim Color
	Red       Color
	Red80     Color
	RedDim    Color
	Pink      Color
	Teal      Color
	Peach     Color
	Sapphire  Color
}

// CatppuccinMocha is the default palette.
var CatppuccinMocha = Palette{
	Base:      Color{30, 30, 46},
	Surface0:  Color{49, 50, 68},
	Surface1:  Color{69, 71, 90},
	Blue:      Color{137, 180, 250},
	Green:     Color{166, 227, 161},
	Green80:   Color{133, 182, 129},
	GreenDim:  Color{83, 113, 80},
	Yellow:    Color{249, 226, 175},
	Yellow80:  Color{199, 181, 140},
	YellowDim: Color{124, 113, 87},
	Red:       Color{243, 139, 168},
	Red80:     Color{194, 111, 134},
	RedDim:    Color{121, 69, 84},
	Pink:     Color{245, 194, 231},
	Teal:     Color{148, 226, 213},
	Peach:    Color{250, 179, 135},
	Sapphire: Color{116, 199, 236},
}
