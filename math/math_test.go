// Package math contains aditionals to teh math packages
// in time, these can all be collected into a single package.
package math

import (
	"fmt"
	"testing"
)

func TestRandPermuation(t *testing.T) {
	size := 25
	M := RandPermutation(size)
	if len(M) != size {
		t.Errorf("wrong array size! expected %s but got %s", size, len(M))
	}
	fmt.Printf("Permuation of size %d :\n%+v\n", size, M)
}

func TestTransformation(t *testing.T) {

	test_cx := 1.1
	test_cy := 3.3
	test_hscale := 0.2
	test_width := 20
	test_height := 10

	f := Transformation(test_cx, test_cy, test_hscale, test_width, test_height)

	pixels := [][2]int{
		{20, 0},
		{20, 10},
		{10, 5}, // <--- should return cx,cy
		{0, 5},	
		{20, 10},	
		{20, 12},
	}
	for _, P := range pixels {
		fmt.Printf("pr=%d, pd=%d =>\t  %+v\n", P[0], P[1], f(P[0], P[1]))
	}
}
