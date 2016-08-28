// Package rgba returns a palette of 255*6 = 1530 RGB colors.
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


// This is the array of (unit) 'differences' between sucessive colors around the RGB wheel
var circle = []int{
	0x100,    // {0, 1, 0},
	-0x10000, // {-1, 0, 0},
	0x1,      // {0, 0, 1},
	-0x100,   // {0, -1, 0},
	0x10000,  // {1, 0, 0},
	-0x1,     // {0, 0, -1},
}

// ColorWheel contructs a palette of 255*6 colors starting at Red {255,0,0}
// by walking a circuit around a cube, interpolating a single dimension at a time.
// The palette itself consists of the RGB coordinates of each color.
func ColorWheel() []color.RGBA {
	var a = [3]uint8{}
	v := 0xff0000 // <--> {255, 0, 0} : RED
	r := []color.RGBA{}
	for i := 0; i < 6; i++ {
		for j := 0; j < 255; j++ {
			a = math.Base256(v)
			r = append(r, color.RGBA{a[0], a[1], a[2], 255})
			v += circle[i]
		}
	}
	return r
}

var line = []int{
	-0x10000, // {-1, 0, 0},
	-0x100,   // {0, -1, 0},
	0x10000,  // {1, 0, 0},
	-0x1,     // {0, 0, -1},
	0x100,    // {0, 1, 0},
	-0x10000, // {-1, 0, 0},
	-0x100,   // {0, 1, 0},
}

// ColorPath starts at WHITE {255,255,255} and traces a path
// along seven edges of the cube to visit all vertices and end at
// BLACK+ {0,0,1}, interpolating along the way
func ColorPath() []color.RGBA {
	var a = [3]uint8{}
	v := 0xffffff // <--> {255, 255, 255} : WHITE
	r := []color.RGBA{}
	for i := 0; i < 7; i++ {
		for j := 0; j < 255; j++ {
			a = math.Base256(v)
			r = append(r, color.RGBA{a[0], a[1], a[2], 255})
			v += line[i]
		}
	}
	return r
}





