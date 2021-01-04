package color

import "github.com/gookit/color"

type Color interface {
	Sprint(a ...interface{}) string
	Sprintf(format string, args ...interface{}) string
	Print(args ...interface{})
	Printf(format string, a ...interface{})
}

const (
	Red     = color.Red
	Cyan    = color.Cyan
	Gray    = color.Gray
	Blue    = color.Blue
	Black   = color.Black
	Green   = color.Green
	White   = color.White
	Yellow  = color.Yellow
	Magenta = color.Magenta
)

var StandardColors = []Color{Red, Cyan, Gray, Blue, Black, Green, White, Yellow, Magenta}

func RGB(r, g, b uint8) Color {
	return color.RGB(r, g, b)
}

func RGBBackground(r, g, b uint8) Color {
	return color.RGB(r, g, b, true)
}

func S256(fgAndBg ...uint8) Color {
	return color.S256(fgAndBg...)
}
