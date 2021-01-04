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
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/howardjohn/log-helper/pkg/color"
	"github.com/mkmik/argsort"
)

type Config struct {
	Colors []string `json:"colors"`
}

func ReadConfig() (Config, error) {
	by, err := ioutil.ReadFile(".config.yaml")
	if os.IsNotExist(err) {
		return Config{
			Colors: []string{
				`#dc322f`,
				`#859900`,
				`#b58900`,
				`#268bd2`,
				`#d33682`,
				`#2aa198`,
			},
		}, nil
	}
	if err != nil {
		return Config{}, err
	}
	c := Config{}
	if err := yaml.Unmarshal(by, &c); err != nil {
		return c, err
	}
	return c, nil
}

type Matcher struct {
	r     *regexp.Regexp
	color color.Color
}

func (m Matcher) FindIndexes(s string) []IndexRange {
	res := m.r.FindAllStringIndex(s, -1)
	ret := make([]IndexRange, 0, len(res))
	for _, r := range res {
		ret = append(ret, IndexRange{r[0], r[1]})
	}
	return ret
}

type IndexRange struct {
	start, stop int
}
type ColoredIndexRange struct {
	IndexRange
	color color.Color
}

func FindAllMatches(ms []Matcher, s string) []ColoredIndexRange {
	current := []ColoredIndexRange{}
	for _, m := range ms {
		res := m.FindIndexes(s)
		for _, r := range res {
			overlap := false
			for _, c := range current {
				if c.start <= r.start && r.start < c.stop {
					overlap = true
					break
				}
				if c.start <= r.stop && r.stop < c.stop {
					overlap = true
					break
				}
			}
			if !overlap {
				current = append(current, ColoredIndexRange{
					IndexRange: r,
					color:      m.color,
				})
			}
		}
	}
	sort.Slice(current, func(i, j int) bool {
		return current[i].start < current[j].start
	})
	return current
}

var (
	colorTest = flag.Bool("test-colors", false, "test color support")
	runLogs   = flag.Bool("logs", false, "run log highlighter")

	// Log viewer
	filterUnmatched = flag.Bool("filter", false, "filter unmatched lines")
)

var (
	knownLogFormats = []*regexp.Regexp{
		regexp.MustCompile(`^20..-..-..T..:..:..\.......Z\t`),
	}
)

func ParseColors(s []string) []color.Color {
	res := []color.Color{}
	for _, cc := range s {
		res = append(res, color.Hex(cc))
	}
	return res
}

func ExtrapolateColorList(colors []color.Color, idx int, max int) color.Color {
	tints := max/len(colors) + 1
	tint := idx / len(colors)
	return color.Lighten(colors[idx%len(colors)], float64(tint)/float64(tints))
}

func main() {
	flag.Parse()
	cfg, err := ReadConfig()
	if err != nil {
		panic(err.Error())
	}
	if *colorTest {
		runColorTest()
		return
	}
	if *runLogs {
		all, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err.Error())
		}
		if err := logTimeBuffered(all); err != nil {
			panic(err.Error())
		}
		return
	}
	matchers := []Matcher{}
	args := flag.Args()
	for i, r := range args {
		rx := regexp.MustCompile(r)
		matchers = append(matchers, Matcher{
			r:     rx,
			color: ExtrapolateColorList(ParseColors(cfg.Colors), i, len(args)),
		})
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
		m := FindAllMatches(matchers, line)
		o := getLine(m, line)
		w.Write([]byte(o))
	}
}

func getLine(matches []ColoredIndexRange, line string) string {
	if len(matches) == 0 {
		return line
	}
	sb := strings.Builder{}
	prev := 0
	for _, match := range matches {
		sb.WriteString(line[prev:match.start])
		sb.WriteString(match.color.Sprint(line[match.start:match.stop]))
		prev = match.stop
	}
	sb.WriteString(line[prev:])
	return sb.String()

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
	timeLines := 0
	for i := range lines {
		p := times[i]
		if p == nil {
			continue
		}
		timeLines++
		p.t.Sub(lastTime)
		p.delta = p.t.Sub(lastTime)
		lastTime = p.t
	}

	// todo blanks steal spots
	ranks := argsort.Sort(TimeSlice(times))
	for i, r := range ranks {
		if times[r] != nil {
			times[r].rank = i - (len(ranks) - timeLines)
		}
	}
	for i := range lines {
		p := times[i]
		line := lines[i]
		if p == nil {
			if !*filterUnmatched {
				fmt.Println(string(line))
			}
			continue
		}

		fmt.Printf("%s%s\n", rankToColor(p.rank, len(ranks)).Sprint(string(line[:p.bits])), string(line[p.bits:]))
	}

	return nil
}

func rankToColor(rank int, total int) color.Color {
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
