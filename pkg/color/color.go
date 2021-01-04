package color

import (
	"fmt"

	"github.com/gookit/color"
)

type Color interface {
	Sprint(a ...interface{}) string
	Sprintf(format string, args ...interface{}) string
	Print(args ...interface{})
	Printf(format string, a ...interface{})
}

const (
	Red       = color.Red
	LightRed  = color.LightRed
	Cyan      = color.Cyan
	LightCyan = color.LightCyan
	Blue      = color.Blue
	LightBlue = color.LightBlue

	Green      = color.Green
	LightGreen = color.LightGreen

	Yellow       = color.Yellow
	LightYellow  = color.LightYellow
	Magenta      = color.Magenta
	LightMagenta = color.LightMagenta

	Black      = color.Black
	Gray       = color.Gray
	White      = color.White
	LightWhite = color.LightWhite
)

var StandardColors = []Color{
	LightRed, LightGreen, LightYellow, LightBlue, LightMagenta, LightCyan,
	Red, Green, Yellow, Blue, Magenta, Cyan,
}

type Gradiant struct {
	colors []color.RGBColor
}

func (g Gradiant) For(n float64) Color {
	if n < 0 || n > 1 {
		panic(fmt.Sprintf("must be [0,1], got %v", n))
	}
	if n == 1 {
		return g.colors[len(g.colors)-1]
	}
	if n == 0 {
		return g.colors[0]
	}
	base := int(n * float64(len(g.colors)-1))

	segments := float64(len(g.colors) - 1)
	scale := segments
	startPercentage := n - (float64(base) / segments)
	distanceBetweenColors := startPercentage * scale
	basec := g.colors[base]
	nextc := g.colors[base+1]
	return color.RGB(
		uint8(float64(basec[0])+(float64(nextc[0])-float64(basec[0]))*(distanceBetweenColors)),
		uint8(float64(basec[1])+(float64(nextc[1])-float64(basec[1]))*(distanceBetweenColors)),
		uint8(float64(basec[2])+(float64(nextc[2])-float64(basec[2]))*(distanceBetweenColors)),
		!g.colors[0].IsEmpty(),
	)
}

func NewGradiant(colors ...Color) Gradiant {
	res := make([]color.RGBColor, 0, len(colors))
	for _, c := range colors {
		switch t := c.(type) {
		case color.RGBColor:
			res = append(res, t)
		case color.Color256:
			res = append(res, t.RGBColor())
		default:
			panic("unsupported")
		}
	}
	return Gradiant{res}
}

func RGB(r, g, b uint8) Color {
	return color.RGB(r, g, b)
}

func RGBBackground(r, g, b uint8) Color {
	return color.RGB(r, g, b, true)
}

func S256(fgAndBg ...uint8) Color {
	return color.S256(fgAndBg...)
}
