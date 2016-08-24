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

func main() {
	http.HandleFunc("/", handler_init)
	http.HandleFunc("/image/", serveImage)
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("../html"))))
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// display the base html
func handler_init(w http.ResponseWriter, r *http.Request) {
	ReadQueryString(r) // handle the querystring
	// set the pixel to point mapping
	Px2S = math.Transformation(cx, cy, hScale, width, height)
	Px2C = func(p Point) complex128 {
		re, im := Px2S(p.Right, p.Down)
		return complex(re, im)
	}
	w.Write([]byte(html.UI))
}

// return a Mandelbrot image
func serveImage(w http.ResponseWriter, r *http.Request) {
	getImageReq(r)
	//ReadQueryString(r) // handle the querystring
	SendImage(w)       // fetch the base64 encoded png image
}

func SendImage(w io.Writer) {
	if position == N { // complete we are done, send this banner to js
		SendBanner(w)
		position = 0
		return
	}

	iterations = num * 600 // we scale the given iterations

	lastIndex := resetEnd(position, rate)

	screenChan := make(chan image.Image)

	// set the partial image go routines going
	for gor := 0; gor < numberOfroutines; gor++ {
		go BuildImage(gor, lastIndex, screenChan)
	}

	var canvas *image.RGBA

	// assemble the image
	canvas = image.NewRGBA(image.Rect(0, 0, width, height))
	count := 0
	op := draw.Src
	for img := range screenChan {
		draw.Draw(canvas, canvas.Bounds(), img, image.ZP, op)
		if op == draw.Src { // the first draw operation is the only .Src
			op = draw.Over // type - the rest are .Over
		}
		count++
		if count == numberOfroutines {
			close(screenChan)
		}
	}

	// the assembly loop had blocked, and now we process result ....
	Encode(w, canvas) // write the binary to writer w

	position = lastIndex // update starting point for next partial
}


var visited bool

func getImageReq(r *http.Request) {
	var err error
	if err = r.ParseForm(); err != nil {
		log.Print(err)
	}

	//set up our vars num=iterations
	for k, v := range r.Form {
		if options[k] != 3 {
			continue
		}
		if position == 0 {
			visited = false
		}

		if !visited {
			var x1, y1 int
			
			if k == "newpt" { // move center
				w := strings.Split(v[0], "|")
				x1, err = strconv.Atoi(w[0])
				if err != nil {
					log.Print(err)
					continue
				}
				y1, err = strconv.Atoi(w[1])
				if err != nil {
					log.Print(err)
					continue
				}
				// recenter
				cx,cy = Px2S(x1,y1)
			}

			if k == "in" { // move center
		 		hScale = hScale * 3 / 4
			}

			if k == "out" { // move center
		 		hScale = hScale * 2 
			}

			// ini all these cases, reset
			Px2S = math.Transformation(cx, cy, hScale, width, height)
			Px2C = func(p Point) complex128 {
				re, im := Px2S(p.Right, p.Down)
				return complex(re, im)
			}
			visited = true
		}
	}

}

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
	Color            func(d int) color.RGBA
	Sigma            func(d int) int
	Px2S             func(pr, pd int) (float64, float64) // pixel to Cartesian
	Px2C             func(p Point) complex128
	Pixel            [N]Point // all the screen points
	position         = 0
	rate             = 64          // how quickly we process - default if we dont  ?r=345 etc
	num              = 5           // multiples of 600
	iterations       = 3000        //  always 600 x  TODO kill one of these two
	numberOfroutines = 2           // number of concurrent go routines
	cx, cy           = -0.73, 0.23 // x, y coords of central point pixel (width/2,height/2)
	hScale           = 0.02        // hScale = cx-0
	col              = 1256.0      // color
	options          = map[string]int{
		"m": 1, "r": 1, "num": 1, // 1 = int, 2 = float
		"x": 2, "y": 2, "w": 2,
		"dpx": 1, "dpy": 1,
		"in": 3, "out": 3,
		"col": 2, "newpt": 3,
	}
)

func ReadQueryString(r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Print(err)
	}

	//set up our vars num=iterations
	for k, v := range r.Form {
		if options[k] == 0 {
			continue
		}
		if options[k] == 3 {
			continue
		}
		var (
			n   int
			z   float64
			err error
		)
		if options[k] == 1 { // int value
			n, err = strconv.Atoi(v[0])
			if err != nil {
				log.Print(err)
				continue
			}
		} else {
			z, err = strconv.ParseFloat(v[0], 64)
			if err != nil {
				log.Print(err)
				continue
			}
		}
		switch k {
		case "num":
			num = n // global var number of  iterations
		case "r":
			rate = n // global var rate
		case "m":
			numberOfroutines = n // global var number of goroutines
		case "x":
			cx = z // global var center x coord
		case "y":
			cy = z // global var center y coord
		case "w":
			hScale = z // global var half a side
		case "col":
			col = z // change the hue
		}
	}
}


// Encode take image, first PNG encodes it then
// writes its Base64 encoding to writer w
func Encode(w io.Writer, image *image.RGBA) {
	// generate PNG
	buf := new(bytes.Buffer)
	png.Encode(io.Writer(buf), image) // NOTE: ignoring errors, to an io.Writer
	// convert to Base64
	encoder := base64.NewEncoder(base64.StdEncoding, w) // send to target w
	encoder.Write(buf.Bytes())
	encoder.Close()
}

// resetEnd returns the last index for points given
// the 'rate' refresh (which decides refresh) and
// current index pos along points list
func resetEnd(pos, refresh int) int {
	if refresh == 0 {
		refresh = 256
	}
	lastInd := pos + 1024*refresh
	if lastInd > N {
		lastInd = N
	}
	return lastInd
}

// BuildImage generates a partial image of the Mandelbrot set and sends
// this to screenChan. This is called in a goroutine indexed by part <= numberOfgoroutines
// and draws a selection of the pixels. lastIndex is the point where we stop
// thereby animating the progression
func BuildImage(part int, lastIndex int, screenChan chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	for k := position; k < lastIndex; k++ {
		if k%numberOfroutines != part { // choose our residue class
			continue
		}
		p := Pixel[Sigma(k)] // use our permutation
		z := Px2C(p)         //Point2C(p)
		d := mandelBrot(z)
		img.Set(p.Right, p.Down, Color(18*d)) // no switch??
	}

	screenChan <- img
}

// mandelBrot performs the iteration from point z
// returning the number of iterations for an escape otherwise
// tookTooLong (=0)
func mandelBrot(z complex128) int {
	var v complex128
	for n := 0; n < iterations; n++ { // iterations comes from global 'num' !!
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return n
		}
	}
	return tookTooLong
}

// SendBanner writes the details of the last build
// at the top of the page
func SendBanner(w io.Writer) {
	sx := strconv.FormatFloat(cx, 'f', -1, 64)
	sy := strconv.FormatFloat(cy, 'f', -1, 64)
	sr := strconv.FormatFloat(hScale, 'f', -1, 64)
	si := strconv.Itoa(num)
	st := strconv.Itoa(rate)
	sm := strconv.Itoa(numberOfroutines)
	sc := strconv.FormatFloat(col, 'f', -1, 64)
	// the leading '_' below is a signal to the client that we are finished (see html.UI, js)
	io.WriteString(w, "_"+sx+"_"+sy+"_"+sr+"_"+si+"_"+st+"_"+sm+"_"+sc)
}


func init() {
	k := 0
	for down := 0; down < height; down++ {
		for right := 0; right < width; right++ {
			Pixel[k] = Point{right, down}
			k++
		}
	}

	permutation := math.RandPermutation(N)
	Sigma = func(d int) int {
		if d < 0 || d >= N {
			panic(fmt.Sprintf("can't handle % in permutation", d))
		}
		return permutation[d]
	}

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
