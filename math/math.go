// Package math contains aditionals to teh math packages
// in time, these can all be collected into a single package.
package math

import (
	"math/rand"
	"time"
)

// RandPermutation is Knuth's algorithm for producing
// a random permuation on symbols 0 .. N-1
func RandPermutation(N int) []int {
	t := time.Now().Nanosecond()
	rand.Seed(int64(t))

	var v []int

	// start with the identity
	for i := 0; i < N; i++ {
		v = append(v, i)
	}
	// perform random swaps
	for i := 0; i < N-1; i++ {
		j := rand.Intn(N-i) + i // now i <= j <= N-1
		h := v[j]
		v[j] = v[i]
		v[i] = h //swap v_i,v_j
	}
	return v
}

// Transformation returns a function that converts a pixel (pr,pd)
// with right and down coordinates from TL corner, into its Cartseian
// coordinates wrt to origin (width/2, height/2), with the usual orientation
func Transformation(cx, cy, hscale float64, width, height int) func(pr, pd int) (float64,float64) {
	if hscale <= 0 || width <= 0 || height <= 0 {
		panic("not allowed negative values there!")
	}
	vscale := float64(height) / float64(width) * hscale
	// setup the retunred function value
	return func(pr, pd int) (float64,float64) {
		if ! (0 <= pr && pr <= width) {
			panic("pixel horizontal coordinate is out of bounds!")
		}
		if ! (0 <= pd && pd <= height) {
			panic("pixel vertical coordinate is out of bounds!")
		}
		re := cx + float64(2*pr-width)/float64(2*width)*hscale
		im := cy + float64(height-2*pd)/float64(2*height)*vscale
		return  re, im 
	}
}
