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
	fmt.Printf("Permuation of size %d :\n%+v\n",size,M)
}
