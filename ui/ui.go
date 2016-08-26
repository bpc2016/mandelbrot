// Package ui is teh user interface part
package ui

import (
	"fmt"
	"log"
	"mandelbrot/math"
	"net/http"
	"strconv"
	"strings"
)

const (
	Width, Height = 1024, 512
)

// View is the active view struct - the rectangle
// in C we are examining Mandelbrot
type View struct {
	X, Y   float64 // the center of rectangle in the Cplx plane
	Hwidth float64 // horizontal width of this rectangle
}

var view = View{X: -0.717, Y: 0.23, Hwidth: 0.02}

// Run is the active profile struct
type Run struct {
	Chunk      int // refresh rate of images
	Iterations int // the number of iterations
	NumPROCS   int // number of parallel goroutines for calculating the image (set to num of processors)
	Density    int // color density - the higher the more color is visible in the image
}

// R is exported and gives the active run profile: iterations, chunk size, number
// of processors and color density. These can be changed through the UI
var R = Run{Chunk: 32, Iterations: 3000, NumPROCS: 4, Density: 8}

// Point is a pixel
type Point struct {
	Right int // right from TL corner (0,0)
	Down  int // down from ""
}

var (
	Px2S func(pr, pd int) (float64, float64) // pixel to Cartesian
	// PixelToComplex is the function that is exported to main and handles
	// the pixel --> Cmplx point translation, which changes with center and focus
	PixelToComplex func(p Point) complex128
)

var (
	ImageChan   = make(chan []byte)   // screen <--- base64 
	RequestChan = make(chan struct{}) // <--- user
)

// StartErver sets the two http handlers and fires up the web server on port 8000
func StartServer() {
	// ../html serves all static content, incls: index.html, mandelbrot.js
	http.Handle("/", http.FileServer(http.Dir("../html")))
	// the /image uri is for calling for image pieces from the js
	http.HandleFunc("/image/", serveImage)

	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// Banner writes the details of the last build
// at the top of the page
func Banner() string {
	sx := strconv.FormatFloat(view.X, 'f', -1, 64)
	sy := strconv.FormatFloat(view.Y, 'f', -1, 64)
	sr := strconv.FormatFloat(view.Hwidth, 'f', -1, 64)
	si := strconv.Itoa(R.Iterations)
	st := strconv.Itoa(R.Chunk)
	sm := strconv.Itoa(R.NumPROCS)
	sc := strconv.Itoa(R.Density)
	// the leading '_' below is a signal to the client that we are finished (see  mandel...js)
	return "_" + sx + "_" + sy + "_" + sr + "_" + si + "_" + st + "_" + sm + "_" + sc
}

//================================ private =======================================

// return a Mandelbrot image
func serveImage(w http.ResponseWriter, r *http.Request) {
	if getImageReq(r, &view) {
		fmt.Printf("Sending request: %+v\n%v\n", view, r.Form)
		// change center or focus
		setTransforms()
		RequestChan <- struct{}{} //
	}

	//fmt.Printf("...view, request: %+v\n%v\n", view, r.Form)

	w.Write(<-ImageChan) // binary
}

func setTransforms() {
	Px2S = math.Transformation(view.X, view.Y, view.Hwidth, Width, Height)
	PixelToComplex = func(p Point) complex128 {
		re, im := Px2S(p.Right, p.Down)
		return complex(re, im)
	}
}

var count int

func getImageReq(r *http.Request, v *View) bool {
	var (
		err        error
		newr, newd int
		n          int
		z          float64
	)

count++
fmt.Printf("count, request: %d\n%v\n", count, r.Form)

	checkF(r.ParseForm())

	for k, val := range r.Form {
		if !(k == "newpt" || k == "in" || k == "out") {
			continue
		}
		if k == "newpt" { // center data: pr|pd
			w := strings.Split(val[0], "|")
			newr, err = strconv.Atoi(w[0])
			checkF(err)
			newd, err = strconv.Atoi(w[1])
			checkF(err)
			v.X, v.Y = Px2S(newr, newd)
		}
		if k == "in" { // scale in
			v.Hwidth = v.Hwidth * 3 / 4
		}
		if k == "out" { // scale out
			v.Hwidth = 2 * v.Hwidth
		}
		if k == "num" || k == "r" || k == "m" || k == "col" { // int value
			n, err = strconv.Atoi(val[0])
			checkF(err)
		}
		if k == "x" || k == "y" || k == "w" {
			z, err = strconv.ParseFloat(val[0], 64)
			checkF(err)
		}
		switch k {
		case "x":
			v.X = z // global var center x coord
		case "y":
			v.Y = z // global var center y coord
		case "w":
			v.Hwidth = z // global var half a side
		case "num":
			R.Iterations = n // global var number of  iterations
		case "r":
			R.Chunk = n // global var chunk
		case "m":
			R.NumPROCS = n // global var number of goroutines
		case "col":
			R.Density = n // change the hue
		}
	}

	return len(r.Form) > 0 // we had some changes
}

func init() {
	setTransforms() // make sure that the base view is set up, main sets Px2C from this call
}

//======================= utility ================================

// abort on errors
func checkF(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
