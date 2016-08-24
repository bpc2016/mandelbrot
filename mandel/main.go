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
	"mandelbrot/html"
	"mandelbrot/math"
	"mandelbrot/rgba"
	"math/cmplx"
	"net/http"
	"strconv"
	"strings"
)

const (
	width, height = 1024, 512
	N             = width * height // number of pixels
	tookTooLong   = 0              // flag failure, color Black
)

type Point struct {
	Right int // right from TL corner (0,0)
	Down  int // down from ""
}

var (
	Color      func(d int) color.RGBA
	Sigma      func(d int) int
	Px2S       func(pr, pd int) (float64, float64) // pixel to Cartesian
	Px2C       func(p Point) complex128
	Pixel      [N]Point       // all the screen points
	position   = 0            // from which to respond to next request
	chunk      = 32           // refresh rate of images
	iterations = 3000         //
	numPROCS   = 4            // number of concurrent go routines - given by runtime.GOMAXPROCS(0)
	cx, cy     = -0.717, 0.23 // real/imaginary coordinates of central point pixel (width/2,height/2)
	hScale     = 0.02         // hScale = cx-0
	density    = 8            // color
)

func main() {
	http.HandleFunc("/", serveContext)
	http.HandleFunc("/image/", serveImage)
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("../html"))))
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// SendImage is the heart of this program. It renders the (partial)
// image to send to the http server, signaling completion
// by sending a banner
func SendImage(w io.Writer) {
	if position >= N { // complete we are done, send this banner to js
		io.WriteString(w, Banner())
		position = 0
		return
	}

	screenChan := make(chan image.Image)

	// set the partial image go routines going
	for gor := 0; gor < numPROCS; gor++ {
		go BuildImage(gor, screenChan)
	}

	var canvas *image.RGBA

	// assemble the image
	canvas = image.NewRGBA(image.Rect(0, 0, width, height))
	gor := 0
	op := draw.Src
	for img := range screenChan {
		draw.Draw(canvas, canvas.Bounds(), img, image.ZP, op)
		if op == draw.Src { // the first draw operation is the only .Src
			op = draw.Over // type - the rest are .Over
		}
		gor++
		if gor == numPROCS {
			close(screenChan)
		}
	}

	// the assembly loop had blocked, and now we process result ....
	Encode(w, canvas) // write the binary to writer w

	position += 1024 * chunk // update starting point for next partial
}

// Encode takes image encodes as PNG encodes then
// again as Base64 then sends that to writer w
func Encode(w io.Writer, image *image.RGBA) {
	// generate PNG
	buf := new(bytes.Buffer)
	png.Encode(io.Writer(buf), image) // NOTE: ignoring errors, to an io.Writer
	// convert to Base64
	encoder := base64.NewEncoder(base64.StdEncoding, w) // send to target w
	encoder.Write(buf.Bytes())
	encoder.Close()
}

// BuildImage generates a partial image of the Mandelbrot set and sends
// this to screenChan. This is called in a goroutine indexed by part <= numPROCS
// and draws a selection of the pixels - as given by var position
func BuildImage(part int, screenChan chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	endposition := position + 1024*chunk
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
		color := Color(density * d) // use our color density
		img.Set(p.Right, p.Down, color)
	}

	screenChan <- img
}

// mandelBrot performs the iteration from point z
// returning the number of iterations for an escape otherwise
// tookTooLong (=0)
func mandelBrot(z complex128) int {
	var v complex128
	for n := 0; n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return n
		}
	}
	return tookTooLong
}

func getTransforms() {
	Px2S = math.Transformation(cx, cy, hScale, width, height)
	Px2C = func(p Point) complex128 {
		re, im := Px2S(p.Right, p.Down)
		return complex(re, im)
	}
}

//========================================== initialize ======================================================

func init() {
	k := 0
	for down := 0; down < height; down++ {
		for right := 0; right < width; right++ {
			Pixel[k] = Point{right, down}
			k++
		}
	}

	Sigma = math.RandPermFunc(N) // we walk through pixels with this

	position = 0 // for partial image creation

	Rainbow := rgba.MakePalette()
	Color = func(d int) color.RGBA {
		if d == tookTooLong {
			return color.RGBA{0, 0, 0, 255}
		}
		return Rainbow[d%1530] // len(Rainbow) = 255*6
	}

	fmt.Println("MandelBrot server started on  http://localhost:8000")
}

//======================================= ui stuff ================================================

// display the base html
func serveContext(w http.ResponseWriter, r *http.Request) {
	ReadQueryString(r) // handle the querystring
	// set the pixel to point mapping
	getTransforms()
	w.Write([]byte(html.UI))
}

// return a Mandelbrot image
func serveImage(w http.ResponseWriter, r *http.Request) {
	getImageReq(r) // handle the querystring
	SendImage(w)   // fetch the base64 encoded png image
}

var visited bool

// getImageReq handles http requests from
// user input: click and keyboard
// uses 'visited' above so as not to repeat itself
func getImageReq(r *http.Request) {
	checkF(r.ParseForm())

	var err error
	for k, v := range r.Form {
		if !(k == "newpt" || k == "in" || k == "out") {
			continue
		}

		visited = position != 0

		if !visited {
			var newr, newd int

			if k == "newpt" { // center data: pr|pd
				w := strings.Split(v[0], "|")
				newr, err = strconv.Atoi(w[0])
				if err != nil { // just use previous value
					log.Printf("format error for %v : %v", k, err)
					continue
				}
				newd, err = strconv.Atoi(w[1])
				if err != nil { // just use previous value
					log.Printf("format error for %v : %v", k, err)
					continue
				}
				// recenter
				cx, cy = Px2S(newr, newd)
			}

			if k == "in" { // scale in
				hScale = hScale * 3 / 4
			}

			if k == "out" { // scale out
				hScale = hScale * 2
			}
			// in all these cases, reset
			getTransforms()
			visited = true
		}
	}

}

var numberType = map[string]int{
	"m": 1, "r": 1, "num": 1, // 1 = int, 2 = float
	"x": 2, "y": 2, "w": 2,
	"dpx": 1, "dpy": 1,
	"col": 1,
}

func ReadQueryString(r *http.Request) {
	// read the form
	checkF(r.ParseForm())

	//set up our vars
	for k, v := range r.Form {
		if numberType[k] == 0 {
			continue
		}
		var (
			n   int
			z   float64
			err error
		)
		if numberType[k] == 1 { // int value
			n, err = strconv.Atoi(v[0])
		} else {
			z, err = strconv.ParseFloat(v[0], 64)
		}
		if err != nil { // just use previous value
			log.Printf("format error for %v : %v", k, err)
			continue
		}
		switch k {
		case "num":
			iterations = n // global var number of  iterations
		case "r":
			chunk = n // global var chunk
		case "m":
			numPROCS = n // global var number of goroutines
		case "x":
			cx = z // global var center x coord
		case "y":
			cy = z // global var center y coord
		case "w":
			hScale = z // global var half a side
		case "col":
			density = n // change the hue
		}
	}
}

// Banner writes the details of the last build
// at the top of the page
func Banner() string {
	sx := strconv.FormatFloat(cx, 'f', -1, 64)
	sy := strconv.FormatFloat(cy, 'f', -1, 64)
	sr := strconv.FormatFloat(hScale, 'f', -1, 64)
	si := strconv.Itoa(iterations)
	st := strconv.Itoa(chunk)
	sm := strconv.Itoa(numPROCS)
	sc := strconv.Itoa(density)
	// the leading '_' below is a signal to the client that we are finished (see html.UI, js)
	return "_" + sx + "_" + sy + "_" + sr + "_" + si + "_" + st + "_" + sm + "_" + sc
}

//======================= utility ================================

// abort on errors
func checkF(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
