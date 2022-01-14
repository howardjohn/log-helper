package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"os"
)

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
