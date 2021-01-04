package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/gookit/color"
	"github.com/mkmik/argsort"
)

type Matcher struct {
	r *regexp.Regexp
}

var (
	colorTest = flag.Bool("test-colors", false, "test color support")
)

var (
	knownLogFormats = []*regexp.Regexp{
		regexp.MustCompile(`^20..-..-..T..:..:..\.......Z\t`),
	}
)

func main() {
	flag.Parse()
	if *colorTest {
		runColorTest()
		return
	}
	all, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err.Error())
	}
	if err := logTimeBuffered(all); err != nil {
		panic(err.Error())
	}
	return
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

type ParsedTime struct {
	t    time.Time
	bits int

	delta time.Duration
	rank  int
}

func matchTime(rs []*regexp.Regexp, data []byte) (*ParsedTime, error) {
	for _, r := range rs {
		if d := r.Find(data); d != nil {
			t1, err := time.Parse(
				`2006-01-02T15:04:05.999999`,
				string(d)[:len(d)-2])
			if err != nil {
				return nil, err
			}
			return &ParsedTime{t: t1, bits: len(d) - 1}, nil
		}
	}
	return nil, nil
}

type TimeSlice []*ParsedTime

func (p TimeSlice) Len() int { return len(p) }
func (p TimeSlice) Less(i, j int) bool {
	if p[i] == nil {
		return true
	}
	if p[j] == nil {
		return false
	}
	return p[i].delta < p[j].delta
}
func (p TimeSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func logTimeBuffered(data []byte) error {
	lines := bytes.Split(data, []byte("\n"))
	times := make([]*ParsedTime, len(lines))
	for i, line := range lines {
		p, err := matchTime(knownLogFormats, line)
		if err != nil {
			return err
		}
		times[i] = p
	}
	lastTime := time.Time{}
	for i := range lines {
		p := times[i]
		if p == nil {
			continue
		}
		p.t.Sub(lastTime)
		p.delta = p.t.Sub(lastTime)
		lastTime = p.t
	}

	// todo blanks steal spots
	ranks := argsort.Sort(TimeSlice(times))
	for i, r := range ranks {

		if times[r] != nil {
			times[r].rank = i
		}
	}
	for i := range lines {
		p := times[i]
		line := lines[i]
		if p == nil {
			fmt.Println(string(line))
			continue
		}

		fmt.Printf("%s%s\n", rankToColor(p.rank, len(ranks)).Sprint(string(line[:p.bits])), string(line[p.bits:]))
	}

	return nil
}

func rankToColor(rank int, total int) color.RGBColor {
	position := 1 - float64(rank)/float64(total)
	if position <= 0.5 {
		return color.RGB(255, uint8(255*position*2), 0)
	}
	return color.RGB(255-uint8(255*(position-0.5)*2), 255, 0)
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
		color.RGB(255, uint8(i), 0, true).Printf(" ")
	}
	fmt.Println()
	for i := 255; i >= 0; i -= 2 {
		color.RGB(255-uint8(i), 255, 0, true).Printf(" ")
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
