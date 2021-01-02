package main

import (
	"bufio"
	"flag"
	"io"
	"os"
	"regexp"
)

type Matcher struct {
	r *regexp.Regexp
}

func main() {
	flag.Parse()
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
