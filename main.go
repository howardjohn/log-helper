package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	flag.Parse()
	regexs := flag.Args()
	_ = regexs
	w := io.MultiWriter(os.Stdout)
	r := bufio.NewReader(os.Stdin)
	buf := make([]byte, 0, 4*1024)
	r
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err.Error())
		}
		w.Write([]byte(line))
	}

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
