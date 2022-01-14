package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/howardjohn/log-helper/pkg/color"
	"sigs.k8s.io/yaml"
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

func ExtrapolateColorList(colors []color.Color, idx int, max int) color.Color {
	tints := max/len(colors) + 1
	tint := idx / len(colors)
	return color.Lighten(colors[idx%len(colors)], float64(tint)/float64(tints))
}

func ParseColors(s []string) []color.Color {
	res := []color.Color{}
	for _, cc := range s {
		res = append(res, color.Hex(cc))
	}
	return res
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

func compileRegex(regex string) *regexp.Regexp {
	if strings.HasSuffix(regex, "\\x") {
		base := strings.TrimSuffix(regex, "\\x")
		regex = fmt.Sprintf(`(?:\s|^)%s[:=]\S+`, base)
	}
	if flagValues.caseInsensitive {
		regex = `(?i)` + regex
	}
	return regexp.MustCompile(regex)
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
