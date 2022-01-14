package main

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"github.com/howardjohn/log-helper/pkg/color"
	"github.com/mkmik/argsort"
)

var knownLogFormats = []*regexp.Regexp{
	regexp.MustCompile(`^20..-..-..T..:..:..\.......Z\s`),
}

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

type ParsedTime struct {
	t    time.Time
	bits int

	delta time.Duration
	rank  int
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

func rankToColor(rank int, total int) color.Color {
	position := 1 - float64(rank)/float64(total)
	if position <= 0.5 {
		return color.RGB(255, uint8(255*position*2), 0)
	}
	return color.RGB(255-uint8(255*(position-0.5)*2), 255, 0)
}
