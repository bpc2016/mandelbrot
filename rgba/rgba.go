// Package rgba holds the actual Mandelbrot routine
// this is responsible for coloring individual pixels in the image
package rgba

import (
	"image/color"
	"mandelbrot/math"
)


/*
var wheel = []color.Color{
	color.RGBA{255, 0, 0, 255},
	color.RGBA{255, 255, 0, 255},
	color.RGBA{0, 255, 0, 255},
	color.RGBA{0, 255, 255, 255},
	color.RGBA{0, 0, 255, 255},
	color.RGBA{255, 0, 255, 255},
}
*/

var diffArray = []int{
	0x100,    // {0, 1, 0},
	-0x10000, // {-1, 0, 0},
	0x1,      // {0, 0, 1},
	-0x100,   // {0, -1, 0},
	0x10000,  // {1, 0, 0},
	-0x1,     // {0, 0, -1},
}

// MakePalette contructs a palette
// of 255*6 colors starting at Red{255,0,0}
// it des this by walking  the sides of a square
func MakePalette() []color.RGBA {
	var a = [3]uint8{}
	v := 0xff0000 // <--> {255, 0, 0}
	r := []color.RGBA{}
	for i := 0; i < 6; i++ {
		for j := 0; j < 255; j++ {
			a = math.Base256(v)
			r = append(r, color.RGBA{a[0], a[1], a[2], 255})
			v += diffArray[i]
		}
	}
	return r
}




