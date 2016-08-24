package rgba

import (
	"fmt"
	"testing"
)

func TestInterPolate(t *testing.T) {
	Rainbow := MakePalette()
	fmt.Printf("test got %+v",Rainbow[1])
}
