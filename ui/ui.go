// Package ui is teh user interface part
package ui

import (
	"fmt"
	"log"
	"mandelbrot/html"
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
	position   = 0            // from which to respond to next request
	chunk      = 32           // refresh rate of images
	iterations = 3000         //
	numPROCS   = 4            // number of concurrent go routines - given by runtime.GOMAXPROCS(0)
	cx, cy     = -0.717, 0.23 // real/imaginary coordinates of central point pixel (width/2,height/2)
	hScale     = 0.02         // hScale = cx-0
	density    = 8            // color
)

func StartServer() {
	http.HandleFunc("/", serveContext)
	http.HandleFunc("/image/", serveImage)
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("../html"))))
	log.Fatal(http.ListenAndServe("localhost:8000", nil))

}

// display the base html
func serveContext(w http.ResponseWriter, r *http.Request) {
	// ReadQueryString(r) // handle the querystring
	// set the pixel to point mapping
	// getTransforms()

	w.Write([]byte(html.UI))
}

var ImageChan = make(chan []byte)
var RequestChan = make(chan Request)
var NewContext = make(chan struct{})

type Direction int
const (
	none Direction = iota
	in
	out
)

type Request struct {
	Point
	focus Direction
}
var Z = Request{} // empty request

// return a Mandelbrot image
func serveImage(w http.ResponseWriter, r *http.Request) {
	R := getImageReq(r) // handle the querystring
	if R != Z {
		fmt.Printf("Sending request: \n%+v\n", R)
		RequestChan <- R
		fmt.Println("Past receiving request ")
	}
	binary := <-ImageChan
	w.Write(binary)
}

var seenr, seend int

func getImageReq(r *http.Request) Request {
	fmt.Println("request?")
	checkF(r.ParseForm())
	R := Request{}

	var err error
	var newr, newd int

	for k, v := range r.Form {
		if !(k == "newpt" || k == "in" || k == "out") {
			continue
		}
		if k == "newpt" { // center data: pr|pd
			w := strings.Split(v[0], "|")
			newr, err = strconv.Atoi(w[0])
			checkF(err)
			newd, err = strconv.Atoi(w[1])
			checkF(err)
			if newr == seenr && newd == seend {
				break // return Z
			}
			R.Right = newr
			R.Down = newd

			seenr = newr
			seend = newd
			// recenter
			//cx, cy = Px2S(newr, newd)
		}

		if k == "in" { // scale in
			//hScale = hScale * 3 / 4
			R.focus = in
		}

		if k == "out" { // scale out
			// hScale = hScale * 2
			R.focus = out
		}
	}

	return R
}

//var visited bool

// getImageReq handles http requests from
// user input: click and keyboard
// uses 'visited' above so as not to repeat itself
// func getImageReq0(r *http.Request) {

// 	fmt.Println("called getImageReq")

// 	checkF(r.ParseForm())

// 	fmt.Printf("No parse error, \n%+v\n", r.Form)

// 	var err error
// 	for k, v := range r.Form {
// 		if !(k == "newpt" || k == "in" || k == "out") {
// 			continue
// 		}

// 		fmt.Println("getImageReq ...position: ", position)

// 		visited = position != 0

// 		if !visited {
// 			var newr, newd int

// 			if k == "newpt" { // center data: pr|pd
// 				w := strings.Split(v[0], "|")
// 				newr, err = strconv.Atoi(w[0])
// 				if err != nil { // just use previous value
// 					log.Printf("format error for %v : %v", k, err)
// 					continue
// 				}
// 				newd, err = strconv.Atoi(w[1])
// 				if err != nil { // just use previous value
// 					log.Printf("format error for %v : %v", k, err)
// 					continue
// 				}
// 				// recenter
// 				cx, cy = Px2S(newr, newd)
// 			}

// 			if k == "in" { // scale in
// 				hScale = hScale * 3 / 4
// 			}

// 			if k == "out" { // scale out
// 				hScale = hScale * 2
// 			}
// 			// in all these cases, reset
// 			getTransforms()
// 			visited = true
// 		}
// 	}

// }

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
