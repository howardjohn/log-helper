package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

type flags struct {
	colorTest bool
	runLogs   bool

	caseInsensitive bool
	filterUnmatched bool
	kube            bool
	kubelight       bool

	preset    string
	colorMode string
}

var flagValues = flags{
	preset:    "default",
	colorMode: "on",
}

func init() {
	flag.BoolVar(&flagValues.colorTest, "test-colors", flagValues.colorTest, "test color support")
	flag.BoolVar(&flagValues.caseInsensitive, "i", flagValues.caseInsensitive, "case insensitive")
	flag.BoolVar(&flagValues.kube, "k", flagValues.kube, "replace Kubernetes IPs with names and highlight")
	flag.BoolVar(&flagValues.kubelight, "kk", flagValues.kubelight, "hightlight Kubernetes IPs with names")
	flag.BoolVar(&flagValues.runLogs, "logs", flagValues.runLogs, "run log highlighter")

	flag.StringVar(&flagValues.colorMode, "color", flagValues.colorMode, "whether color is used (on, off, auto)")
	flag.StringVar(&flagValues.preset, "preset", flagValues.preset, "preset configuration to use")
	flag.StringVar(&flagValues.preset, "p", flagValues.preset, "preset configuration to use (shorthand)")
	flag.BoolVar(&flagValues.filterUnmatched, "filter", flagValues.filterUnmatched, "filter unmatched lines")
}

func main() {
	flag.Parse()
	cfg, err := ReadConfig(flagValues.preset)
	if err != nil {
		panic(err.Error())
	}

	if flagValues.colorMode == "auto" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			flagValues.colorMode = "on"
		} else {
			flagValues.colorMode = "off"
		}
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
	staticMatch := cfg.GetMatchers(flag.Args())
	var matchers MatcherProvider = StaticMatchers{staticMatch}

	var replacer Replacer = strings.NewReplacer()
	if flagValues.kube || flagValues.kubelight {
		kr, err := NewKubeReplacer(!flagValues.kubelight)
		if err != nil {
			panic(err.Error())
		}
		replacer = kr
		matchers = NewKubeMatcher(staticMatch, kr, ParseColors(cfg.Colors))
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
		r := replacer.Replace(line)
		m := FindAllMatches(matchers.GetMatchers(), r)
		o := getLine(m, r)
		w.Write([]byte(o))
	}
}
