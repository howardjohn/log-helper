package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/gookit/color"
)

type Matcher struct {
	r *regexp.Regexp
}

var (
	colorTest = flag.Bool("test-colors", false, "test color support")
)

func main() {
	flag.Parse()
	if *colorTest {
		runColorTest()
		return
	}
	matchers := []Matcher{}
	for _, r := range flag.Args() {
		rx := regexp.MustCompile(r)
		matchers = append(matchers, Matcher{r: rx})
	}
	w := io.MultiWriter(os.Stdout)
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err.Error())
		}
		out := match(matchers, line)
		w.Write([]byte(out))
	}
}

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
	for i := 0; i < 256; i += 2 {
		color.RGB(uint8(i), 0, 0, true).Printf(" ")
	}
	fmt.Println()
	for i := 0; i < 256; i += 2 {
		color.RGB(0, uint8(i), 0, true).Printf(" ")
	}
	fmt.Println()
	for i := 0; i < 256; i += 2 {
		color.RGB(0, 0, uint8(i), true).Printf(" ")
	}
	fmt.Println()
}

func match(ms []Matcher, line string) string {
	for _, m := range ms {
		if m.r.MatchString(line) {
			return line
		}
	}
	return ""
}

func count(l []byte, c byte) int {
	i := 0
	for _, gc := range l {
		if gc == c {
			i++
		}
	}
	return i
}
