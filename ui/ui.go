// Package ui handles and sets up the user interface.
// The natural navigation aides presented by the UI are
//
// (1) mouse click : this recenters the Mandelbrot calculation. The selected point becomes the new
// rectangle's center. Then the keyboard shortcuts ...
//
// (2) + : zoom in with the same center
//
// (3) - - : zoom out
//
// Finer control is provided by text fields at the top of the image which displays current settings
// but from which can be entered
//
// x  -   real part of the center point in C
//
// y  -   imaginary part of the center
//
// wd -   width of the rectangle in C (the height is scaled accordingly)
//
// itrs - number of  iterations
//
// refr - determines the size of each partial image, hence the refresh rate
//
// gors - coloring uses this number of parallel computations
//
// clr  - the larger this value the higher the color variation of the image
package ui

import (
	"log"
	"mandelbrot/math"
	"net/http"
	"strconv"
	"strings"
)

// The rectangle dimensions in pixels
const (
	Width, Height = 1024, 512
)

// View is the active view struct - the rectangle in the Complex plane where we are examining Mandelbrot
type View struct {
	X, Y   float64 // the center of rectangle in the Cplx plane
	Hwidth float64 // horizontal width of this rectangle
}

var view = View{X: -0.717, Y: 0.23, Hwidth: 0.02}

// Context is the active profile struct
type Context struct {
	Chunk      int // refresh rate of images
	Iterations int // the number of iterations
	NumPROCS   int // number of parallel goroutines for calculating the image (set to num of processors)
	Density    int // color density - the higher the more color is visible in the image
}

// Ctx gives the active profile: iterations, chunk size, number
// of processors and color density. These can be changed through the UI
var Ctx = Context{Chunk: 32, Iterations: 2000, NumPROCS: 4, Density: 8}

// Point on the screen - given by the Right and Down pixels from the Top Left corner (0,0)
type Point struct {
	Right int // right from TL corner (0,0)
	Down  int // down from ""
}

// PixelToComplex is the pixel --> Cmplx point translation, which changes with center and focus.
// It is exported as Px2C
var (
	PixelToComplex func(p Point) complex128
	px2cart        func(pr, pd int) (float64, float64) // pixel to Cartesian
)

var (
	Base64Ready = make(chan []byte)   // screen <--- base64
	NextPlease  = make(chan struct{}) // <--- user
)

// StartServer configures the http handlers and fires up the web server on port 8000
func StartServer() {
	http.HandleFunc("/", serveContext)
	http.Handle("/html/",
		http.StripPrefix("/html/", http.FileServer(http.Dir("../html"))))
	http.HandleFunc("/image/", serveImage)

	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// Banner writes the details of the last run to the top of the page
func Banner() string {
	sx := strconv.FormatFloat(view.X, 'f', -1, 64)
	sy := strconv.FormatFloat(view.Y, 'f', -1, 64)
	sr := strconv.FormatFloat(view.Hwidth, 'f', -1, 64)
	si := strconv.Itoa(Ctx.Iterations)
	st := strconv.Itoa(Ctx.Chunk)
	sm := strconv.Itoa(Ctx.NumPROCS)
	sc := strconv.Itoa(Ctx.Density)
	// the leading '_' below is a signal to the client that we are finished (see  mandel...js)
	return "_" + sx + "_" + sy + "_" + sr + "_" + si + "_" + st + "_" + sm + "_" + sc
}

//================================ private =======================================

var firstTime = true

func serveContext(w http.ResponseWriter, r *http.Request) {
	if !firstTime { // this is to accommodate refresh of the url after the start .. a headache!
		NextPlease <- struct{}{} 
	}
	if firstTime {
		firstTime = false
	}
	w.Write([]byte(indexHtml))
}

const indexHtml = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
   "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"
Cache-Control:No-Cache;>
<html>
  <head>
  <title>Mandelbrot</title>
    <script type="text/javascript" src="html/jquery-1.9.1.min.js"></script>
    <script type="text/javascript" src="html/mandelbrot.js"></script>
  </head>
  <body>
    <form id=data ">
      x: <input type="text" size=8 name="x">
      y: <input type="text" size=8 name="y">
      wd: <input type="text" size=5 name="w">
      itrs: <input type="text" size=3 name="num">
      refr: <input type="text" size=3 name="r">
      grs: <input type="text" size=3 name="m">
      clr: <input type="text" size=3 name="col">
      &nbsp;
      <button id="getForm" type="button">Submit</button>
    </form>
    <p>
    <div id="imgs" style="position:relative">
    </div>
  </body>
</html>
`

// return a Mandelbrot image
func serveImage(w http.ResponseWriter, r *http.Request) {
	if gotRequest(r, &view, &Ctx) {
		setTransforms()
		NextPlease <- struct{}{} // signal readiness for data
	}

	w.Write(<-Base64Ready)
}

func setTransforms() {
	px2cart = math.Transformation(view.X, view.Y, view.Hwidth, Width, Height)
	PixelToComplex = func(p Point) complex128 {
		re, im := px2cart(p.Right, p.Down)
		return complex(re, im)
	}
}

var count int

func gotRequest(r *http.Request, v *View, ctx *Context) bool {
	var (
		err        error
		newr, newd int
		n          int
		z          float64
	)

	checkF(r.ParseForm())

	for k, val := range r.Form {
		if k == "newpt" { 
			w := strings.Split(val[0], "|")
			newr, err = strconv.Atoi(w[0])
			checkF(err) // TODO - make this more forgiving: use prev values
			newd, err = strconv.Atoi(w[1])
			checkF(err)
			v.X, v.Y = px2cart(newr, newd)
		}
		if k == "in" { 
			v.Hwidth = v.Hwidth * 3 / 4
		}
		if k == "out" {
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
			v.X = z 
		case "y":
			v.Y = z 
		case "w":
			v.Hwidth = z
		case "num":
			ctx.Iterations = n 
		case "r":
			ctx.Chunk = n 
		case "m":
			ctx.NumPROCS = n 
		case "col":
			ctx.Density = n 
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
