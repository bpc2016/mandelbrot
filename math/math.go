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
