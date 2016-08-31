// Package Mandelbrot displays Mandelbrot sets in RGBA color. I
//
// It does this in a progressive manner which can be changed through the ui.
// This is achieved by manipulating the DOM. Specifically, each complete image
// is a composite of several partial images (determined by ui.Ctx.Chunk, presented to the user as refr). We
// use transparency of PNG images to overlay the images.
//
// To speed up generation, the work is split amongst (gors) go routines, ideally a number close to the
// available coprocessors.
//
// The production and display are controlled by a go channel (between the ui package and main) and a succession
// of ajax GET calls from the user's web page
package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"mandelbrot/math"
	"mandelbrot/rgba"
	"mandelbrot/ui"
	"math/cmplx"
)

// N is the total number of pixels teh screen has
const N = ui.Width * ui.Height // number of pixels

const tookTooLong = 0          // flag failure, color Black
var (
	paint  func(d int) color.RGBA // we color a point by associated number d
	sigma  func(d int) int        // returns the application of our permutation on integer d
	px2cx  = ui.PixelToComplex    // converts a screen point (pixels) to a complex number
	pixels [N]ui.Point            // the array of all the screen points
)
var numPROCS = ui.Ctx.NumPROCS // number of parallel goroutines - here, I've used runtime.GOMAXPROCS(0)

// LastPiece tells main that the core is finished sending pieces of the partialImage
const LastPiece = -1

// PartialFrom commissions numProcs goroutines to generate images starting from position.
// It returns the image togeter with the next starting position for another image
func PartialFrom(position int) ([]byte, int) {
	partialImageChan := make(chan image.Image)

	// set the partial image go routines going
	for gor := 0; gor < numPROCS; gor++ {
		go buildImage(gor, position, partialImageChan)
	}

	var partialimage *image.RGBA

	// assemble the image
	partialimage = image.NewRGBA(image.Rect(0, 0, ui.Width, ui.Height))
	gor := 0
	op := draw.Src
	for img := range partialImageChan {
		draw.Draw(partialimage, partialimage.Bounds(), img, image.ZP, op)
		if op == draw.Src { // the first draw operation is the only .Src
			op = draw.Over // type - the rest are .Over
		}
		gor++
		if gor == numPROCS {
			close(partialImageChan)
		}
	}
	nextPosition := position + 1024*ui.Ctx.Chunk
	if nextPosition >= N {
		nextPosition = LastPiece //signal we are finished
	}
	// the assembly loop had blocked, and now we process result ....
	return encoded(partialimage), nextPosition
}

// encoded returns the byte slice of image after conversion to PNG then Base64 encoding
func encoded(image *image.RGBA) []byte {
	// generate PNG
	bufIn := new(bytes.Buffer)
	png.Encode(io.Writer(bufIn), image) // NOTE: ignoring errors, to an io.Writer

	// convert to Base64
	bufOut := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, io.Writer(bufOut)) // send to target w
	encoder.Write(bufIn.Bytes())
	encoder.Close()

	return bufOut.Bytes()
}

// buildImage generates a partial image of the Mandelbrot set and sends
// this to ch. This is called in a goroutine indexed by part <= numPROCS
// and draws a selection of the pixels - as given by the position parameter
func buildImage(part int, position int, ch chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, ui.Width, ui.Height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	endposition := position + 1024*ui.Ctx.Chunk
	if endposition > N {
		endposition = N
	}
	for k := position; k < endposition; k++ {
		if k%numPROCS != part { // choose our residue class
			continue
		}
		p := pixels[sigma(k)] // use our permutation
		z := px2cx(p)
		d := mandelBrot(z)
		color := paint(d) // use our color density
		img.Set(p.Right, p.Down, color)
	}
	ch <- img // send to teh channel
}

// mandelBrot performs the iteration from point z
// returning the number of iterations for an escape otherwise
// tookTooLong (=0)
func mandelBrot(z complex128) int {
	var v complex128
	for n := 0; n < ui.Ctx.Iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return n
		}
	}
	return tookTooLong
}

//========================================== initialize ======================================================

func init() {
	k := 0
	for down := 0; down < ui.Height; down++ {
		for right := 0; right < ui.Width; right++ {
			pixels[k] = ui.Point{right, down}
			k++
		}
	}

	sigma = math.RandPermFunc(N) // we walk through pixels with this

	wheel := rgba.ColorWheel()
	paint = func(d int) color.RGBA {
		if d == tookTooLong {
			return color.RGBA{0, 0, 0, 255}
		}
		d *= ui.Ctx.Density    // for greater color depth
		return wheel[d%1530] // len(wheel) = 255*6
	}

	fmt.Println("MandelBrot server started on  http://localhost:8000")
}
