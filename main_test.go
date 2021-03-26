package main

import "testing"

func TestOverlaps(t *testing.T) {
	tests := []struct {
		name    string
		current []ColoredIndexRange
		r       ColoredIndexRange
		want    bool
	}{
		{
			"overlap exact",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{0, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 2}},
			true,
		},
		{
			"overlap right short",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{0, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 1}},
			true,
		},
		{
			"overlap left short",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{0, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{1, 2}},
			true,
		},
		{
			"overlap both short",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{0, 3}},
			},
			ColoredIndexRange{IndexRange: IndexRange{1, 2}},
			true,
		},
		{
			"overlap right long",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{0, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 3}},
			true,
		},
		{
			"overlap left long",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{1, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 2}},
			true,
		},
		{
			"overlap both long",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{1, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 3}},
			true,
		},
		{
			"no overlap",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{1, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{3, 4}},
			false,
		},
		{
			"no overlap left match",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{1, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{2, 4}},
			false,
		},
		{
			"no overlap right match",
			[]ColoredIndexRange{
				{IndexRange: IndexRange{1, 2}},
			},
			ColoredIndexRange{IndexRange: IndexRange{0, 1}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overlaps(tt.current, tt.r); got != tt.want {
				t.Errorf("overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}
