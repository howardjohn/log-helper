package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/howardjohn/log-helper/pkg/color"
	"github.com/mkmik/argsort"
)

type ConfigFile struct {
	Presets map[string]Config `json:"presets"`
}

type ConfigMatcher struct {
	Regex string `json:"regex"`
}

type Config struct {
	Colors   []string        `json:"colors"`
	Matchers []ConfigMatcher `json:"matchers"`
}

func (c Config) GetMatchers(extra []string) []*Matcher {
	matchers := c.Matchers
	for _, m := range extra {
		matchers = append(matchers, ConfigMatcher{Regex: m})
	}
	colors := ParseColors(c.Colors)
	resp := []*Matcher{}
	for i, r := range matchers {
		rx := compileRegex(r.Regex)
		resp = append(resp, &Matcher{
			r:        rx,
			variants: map[string]int{},
			color:    ExtrapolateColorList(colors, i, len(matchers)),
		})
	}
	return resp
}

func compileRegex(regex string) *regexp.Regexp {
	trim := false
	if strings.HasSuffix(regex, "\\x") {
		base := strings.TrimSuffix(regex, "\\x")
		regex = fmt.Sprintf(`(?:\s|^)%s[:=]\S+`, base)
		trim = true
		_ = base
	}
	if flagValues.caseInsensitive {
		regex = `(?i)` + regex
	}
	_ = trim
	return regexp.MustCompile(regex)
}

func ReadConfig(preset string) (Config, error) {
	defaultConfig := Config{
		Colors: []string{
			`#cb4b16`,
			`#a2ba00`,
			`#e1ab00`,
			`#0096ff`,
			`#6c71c4`,
			`#31bbb0`,
		},
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return Config{}, err
	}
	by, err := ioutil.ReadFile(filepath.Join(base, "log-helper/config.yaml"))
	if os.IsNotExist(err) {
		return defaultConfig, nil
	}
	if err != nil {
		return Config{}, err
	}
	c := ConfigFile{}
	if err := yaml.Unmarshal(by, &c); err != nil {
		return Config{}, err
	}
	cfg, f := c.Presets[preset]
	if !f {
		if preset == "default" {
			return defaultConfig, nil
		}
		return Config{}, fmt.Errorf("preset %q not defined", preset)
	}
	return cfg, nil
}

type Matcher struct {
	r        *regexp.Regexp
	last     int
	variants map[string]int
	color    color.Color
}

var variants = []float64{0, 0.5, -0.25, 0.25, 0.75, -0.375, -0.125}

func (m *Matcher) ColorFor(data string) color.Color {
	iter, f := m.variants[data]
	if !f {
		m.variants[data] = m.last
		iter = m.last
		m.last++
	}
	return color.Adjust(m.color, variants[iter%len(variants)])
}

func (m Matcher) FindIndexes(s string) []IndexRange {
	res := m.r.FindAllStringIndex(s, -1)
	ret := make([]IndexRange, 0, len(res))
	idx := m.r.SubexpIndex("primary")
	if idx == -1 {
		idx = 0
	}
	for _, r := range m.r.FindAllStringSubmatchIndex(s, -1) {
		ret = append(ret, IndexRange{r[2*idx], r[2*idx+1]})
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func overlaps(current []ColoredIndexRange, r IndexRange) bool {
	for _, c := range current {
		if max(c.start, r.start) < min(c.stop, r.stop) {
			return true
		}
	}
	return false
}

func FindAllMatches(ms []*Matcher, s string) []ColoredIndexRange {
	current := []ColoredIndexRange{}
	for _, m := range ms {
		res := m.FindIndexes(s)
		for _, r := range res {
			if !overlaps(current, r) {
				current = append(current, ColoredIndexRange{
					IndexRange: r,
					color:      m.ColorFor(s[r.start:r.stop]),
				})
			}
		}
	}
	sort.Slice(current, func(i, j int) bool {
		return current[i].start < current[j].start
	})
	return current
}

type flags struct {
	colorTest bool
	runLogs   bool

	caseInsensitive bool
	filterUnmatched bool

	preset string
}

var flagValues = flags{
	preset: "default",
}

func init() {
	flag.BoolVar(&flagValues.colorTest, "test-colors", flagValues.colorTest, "test color support")
	flag.BoolVar(&flagValues.caseInsensitive, "i", flagValues.caseInsensitive, "case insensitive")
	flag.BoolVar(&flagValues.runLogs, "logs", flagValues.runLogs, "run log highlighter")

	flag.StringVar(&flagValues.preset, "preset", flagValues.preset, "preset configuration to use")
	flag.StringVar(&flagValues.preset, "p", flagValues.preset, "preset configuration to use (shorthand)")
	flag.BoolVar(&flagValues.filterUnmatched, "filter", flagValues.filterUnmatched, "filter unmatched lines")
}

var knownLogFormats = []*regexp.Regexp{
	regexp.MustCompile(`^20..-..-..T..:..:..\.......Z\s`),
}

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
	cfg, err := ReadConfig(flagValues.preset)
	if err != nil {
		panic(err.Error())
	}

	if flagValues.colorTest {
		runColorTest()
		return
	}
	if flagValues.runLogs {
		all, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err.Error())
		}
		if err := logTimeBuffered(all); err != nil {
			panic(err.Error())
		}
		return
	}
	matchers := cfg.GetMatchers(flag.Args())

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
		if prev > match.start {
			continue
		}
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
			if !flagValues.filterUnmatched {
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
