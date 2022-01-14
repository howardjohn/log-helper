package main

import (
	"fmt"

	"github.com/howardjohn/log-helper/pkg/color"
)

func runColorTest() {
	fmt.Printf("%-22sStandard Color %-42sExtended Color \n", " ", " ")
	for i := range []int{7: 0} {
		color.S256(255, uint8(i)).Printf("   %-4d", i)
	}
	fmt.Print("    ")
	for i := range []int{7: 0} {
		i += 8
		color.S256(0, uint8(i)).Printf("   %-4d", i)
	}

	dark := true
	fmt.Printf("\n%-50s216 Color\n", " ")
	for i := range []int{215: 0} {
		v := i + 16

		if i != 0 {
			if i%18 == 0 && dark {
				dark = false
				fmt.Println()
			}

			if i%36 == 0 {
				dark = true
			}
		}

		if dark {
			color.S256(255, uint8(v)).Printf("  %-4d", v)
		}
	}
	dark = true
	for i := range []int{215: 0} {
		v := i + 16

		if i != 0 {
			if i%18 == 0 && dark {
				dark = false
				fmt.Println()
			}

			if i%36 == 0 {
				dark = true
			}
		}

		if !dark {
			color.S256(0, uint8(v)).Printf("  %-4d", v)
		}
	}

	fmt.Printf("\n%-50sGrayscale Color\n", " ")
	fg := 255
	for i := range []int{23: 0} {
		if i < 12 {
			fg = 255
		} else {
			fg = 0
		}

		i += 232
		color.S256(uint8(fg), uint8(i)).Printf(" %-4d", i)
	}

	fmt.Printf("\n%-50s24-bit Color\n", " ")
	grad := color.NewGradiant(
		color.RGBBackground(0, 0, 0),
		color.RGBBackground(255, 0, 0),
		color.RGBBackground(255, 255, 0),
		color.RGBBackground(0, 255, 0),
		color.RGBBackground(0, 255, 255),
		color.RGBBackground(0, 0, 255),
		color.RGBBackground(255, 128, 255),
		color.RGBBackground(255, 255, 255),
	)
	for i := 0; i <= 128; i += 1 {
		grad.For(float64(i) / 128).Printf(" ")
	}
	fmt.Println()
}
