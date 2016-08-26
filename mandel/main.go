// Package Mandelbrot draws mandelbrot sets in RGBA color
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"mandelbrot/math"
	"mandelbrot/rgba"
	"mandelbrot/ui"
	"math/cmplx"
)

const (
	N           = ui.Width * ui.Height // number of pixels
	tookTooLong = 0                    // flag failure, color Black
)

var (
	Color    func(d int) color.RGBA
	Sigma    func(d int) int
	Px2C     = ui.PixelToComplex
	Pixel    [N]ui.Point     // all the screen points
	numPROCS = ui.R.NumPROCS // number of parallel goroutines - here, I've used runtime.GOMAXPROCS(0)
)

func main() {
	go ui.StartServer()
	for {
		fmt.Println("started producing images")
		for pos := 0; pos < N; pos += 1024 * ui.R.Chunk {
			ui.ImageChan <- PartialFrom(pos)
			fmt.Println("image", pos)
		}
		banner := []byte(ui.Banner())
		ui.ImageChan <- banner //  banner indicates end of sending the image
		<-ui.RequestChan       // wait for a request
	}
}

// PartialFrom returns base64 encoded image from position.
// it is partial because the limiting position is given by ui.R.Chunk
func PartialFrom(position int) []byte {

	screenChan := make(chan image.Image)

	// set the partial image go routines going
	for gor := 0; gor < numPROCS; gor++ {
		go BuildImage(gor, position, screenChan)
	}

	var partialimage *image.RGBA

	// assemble the image
	partialimage = image.NewRGBA(image.Rect(0, 0, ui.Width, ui.Height))
	gor := 0
	op := draw.Src
	for img := range screenChan {
		draw.Draw(partialimage, partialimage.Bounds(), img, image.ZP, op)
		if op == draw.Src { // the first draw operation is the only .Src
			op = draw.Over // type - the rest are .Over
		}
		gor++
		if gor == numPROCS {
			close(screenChan)
		}
	}
	// the assembly loop had blocked, and now we process result ....
	return Encoded(partialimage)
}

// Encoded returns the byte slice of image after
// conversion to PNG followed by base64 encoding
func Encoded(image *image.RGBA) []byte {
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

// BuildImage generates a partial image of the Mandelbrot set and sends
// this to screenChan. This is called in a goroutine indexed by part <= numPROCS
// and draws a selection of the pixels - as given by var position
func BuildImage(part int, position int, screenChan chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, ui.Width, ui.Height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	endposition := position + 1024*ui.R.Chunk
	if endposition > N {
		endposition = N
	}
	for k := position; k < endposition; k++ {
		if k%numPROCS != part { // choose our residue class
			continue
		}
		p := Pixel[Sigma(k)] // use our permutation
		z := Px2C(p)
		d := mandelBrot(z)
		color := Color(ui.R.Density * d) // use our color density
		img.Set(p.Right, p.Down, color)
	}
	screenChan <- img
}

// mandelBrot performs the iteration from point z
// returning the number of iterations for an escape otherwise
// tookTooLong (=0)
func mandelBrot(z complex128) int {
	var v complex128
	for n := 0; n < ui.R.Iterations; n++ {
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
			Pixel[k] = ui.Point{right, down}
			k++
		}
	}

	Sigma = math.RandPermFunc(N) // we walk through pixels with this

	Rainbow := rgba.MakePalette()
	Color = func(d int) color.RGBA {
		if d == tookTooLong {
			return color.RGBA{0, 0, 0, 255}
		}
		return Rainbow[d%1530] // len(Rainbow) = 255*6
	}

	fmt.Println("MandelBrot server started on  http://localhost:8000")
}

//======================= utility ================================

// abort on errors
func checkF(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
