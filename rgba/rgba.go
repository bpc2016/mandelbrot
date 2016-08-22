// Package rgba holds the actual Mandelbrot routine
// this is responsible for coloring individual pixels in the image
package rgba

import (
	//"fmt"
	"image/color"
	"math"
)

var iterations int
var f int
var col float64

// SetPalette simply passes us these two data
func SetPalette(num int, thecol float64) {
	iterations = num * 600 // TODO - get rid of this one , identify with iterations
	const M = 16777215     // 255,255,255 (white)
	f = int(math.Floor(float64(M) / float64(iterations)))
	// fmt.Println("FACTOR: ",f)
	col = thecol
}

// PxColor converts integer duration to a color
// according to our palette
func PxColor0(duration int) color.Color {
	if duration == 0 { // tookTooLong
		return color.Black
	}
	c := coloRatio(duration)
	w := getColors(c)
	return color.RGBA{w[0], w[1], w[2], 255}
}

/*
var wheel = []color.Color{
	color.RGBA{255, 0, 0, 255},
	color.RGBA{255, 255, 0, 255},
	color.RGBA{0, 255, 0, 255},
	color.RGBA{0, 255, 255, 255},
	color.RGBA{0, 0, 255, 255},
	color.RGBA{255, 0, 255, 255},
}

var wheelArray = []int{
	0xff0000, // {255, 0, 0},
	0xffff00, // {255, 255, 0},
	0x00ff00, // {0, 255, 0},
	0x00ffff, // {0, 255, 255},
	0x0000ff, // {0, 0, 255},
	0xff00ff, // {255, 0, 255},
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


func MakePalette() []color.RGBA {
	var a = [3]uint8{};
	v:= 0xff0000 	// <--> {255, 0, 0}
	r := []color.RGBA{}
	for i:=0;i<6;i++{
		for j:=0;j<255;j++{
			a = base256(v)
			r = append(r,color.RGBA{a[0],a[1],a[2],255})
			v += diffArray[i]
		}
	}
	return r
}


func PxColor3(duration int) color.Color {
	if duration == 0 { // tookTooLong
		return color.Black
	}
	// iterations = 4*600
	// const M = 16777215 // 255,255,255 (white)
	// f := int(math.Floor(float64(M)/float64(iterations)))
	// d := 2*600  // half
	w := base256(f * duration) // [23,11,18]
	// fmt.Println("reversed ",255-w[0],255-w[1],255-w[2])
	// fmt.Printf("\n%v, %T\n",w,f)
	// return color.RGBA{w[0], w[1], w[2], 255}
	return color.RGBA{255 - w[2], 255 - w[1], 255 - w[0], 255}
}

func PxColor2(duration int) color.Color {
	if duration == 0 { // tookTooLong
		return color.Black
	}
	m := uint8(duration % 256)
	// x := 1-float64(duration)/float64(iterations); //  0 < x < 1, closer to 0 for longer ones
	// m := uint8(math.Floor(255*x)+1)
	return color.RGBA{255, m, 0, 255}
	// return color.RGBA{1, 1, 1, 255}
}

func PxColor1(duration int) color.Color {
	if duration == 0 { // tookTooLong
		return color.Black
	}
	m := uint8(duration % 256)
	// x := 1-float64(duration)/float64(iterations); //  0 < x < 1, closer to 0 for longer ones
	// m := uint8(math.Floor(255*x)+1)
	return color.RGBA{m, m, m, 255}
	// return color.RGBA{1, 1, 1, 255}
}

const T = 8355771 //codeColors([3]int{127,127,127})

// coloRation uses the ratio of z to the total
// number of iterations modified by a quadratic function
func coloRatio(z int) uint64 {
	B := col
	max := iterations
	x := float64(z) / float64(max)
	y := (1 - B*x - (1-B*float64(max))*x*x) * float64(T)
	return uint64(math.Floor(y))
}

func codeColors(c [3]int) int {
	const B = 1 << 8
	return ((c[0]*B)+c[1])*B + c[2]
}

func base256(v int) [3]uint8 {
	const B = 1 << 8 // 256
	var ww [3]uint8
	ww[0] = uint8(v>>16)
	ww[1] = uint8((v>>8)%B)
	ww[2] = uint8(v%B)

	return ww
}

func getColors(v uint64) [3]uint8 {
	const B = 1 << 8
	var w [3]uint8
	for i := 0; i < 3; i++ {
		w[2-i] = uint8(v % B)
		v = v / B
	}
	return w
}
