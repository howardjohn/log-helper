package main

import (
	"sort"
	"strings"

	"github.com/howardjohn/log-helper/pkg/color"
)

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
		if flagValues.colorMode == "off" {
			sb.WriteString(line[match.start:match.stop])
		} else {
			sb.WriteString(match.color.Sprint(line[match.start:match.stop]))
		}
		prev = match.stop
	}
	sb.WriteString(line[prev:])
	return sb.String()
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
