package main

import (
	"math"
	"testing"
)

func TestDistance(t *testing.T) {
	desired := 10
	items := []float64{0}
	max := 1.0
	min := 0.0
	interval := 0.01
	maxDistance := float64(-1)
	best := float64(0)
	for i := 0; i < desired; i++ {
		for n := min; n <= max; n += interval {
			totalDistance := float64(0)
			for _, item := range items {
				t.Logf("%v to %v: %v", item, n, math.Abs(item-n)*math.Abs(item-n))
				totalDistance += math.Abs(item-n) * math.Abs(item-n)
			}
			if maxDistance == -1 || totalDistance > maxDistance {
				maxDistance = totalDistance
				best = n
			}
		}
		t.Logf("Best: %v with score %v", best, maxDistance)
		items = append(items, best)
	}
	t.Log(items)
}

func TestLinear(t *testing.T) {
	desired := 25
	items := []int{}
	shiftedItems := []float64{}
	interval := 100000
	cur := 0
	cap := interval

	// Iterate over all possible slots, with N interval, then N/2 interval, N/4, ...
	for iter := 0; iter < desired; iter++ {
		items = append(items, cur)
		shiftedItems = append(shiftedItems, 2*(float64(cur)/float64(cap)-0.5))
		for {
			cur += interval
			if cur > cap {
				cur = 0
				interval /= 2
			} else if (cur/interval)%2 == 1 {
				break
			}
		}
	}
	t.Log(items)
	t.Log(shiftedItems)
}
