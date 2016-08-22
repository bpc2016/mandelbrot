// Package Mandelbrot draws mandelbrot sets in RGBA color
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
	"math/cmplx"
	"mandelbrot/html"
	"mandelbrot/rgba"
	//"mark/rgb"
)

func main() {
	http.HandleFunc("/", handler_init)        
	http.HandleFunc("/image/", handler_image)
	http.Handle("/static/",
	 	http.StripPrefix("/static/", http.FileServer(http.Dir("."))))
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// display the base html
func handler_init(w http.ResponseWriter, r *http.Request) {
	//InitPoints()
	ReadQueryString(r) // handle the querystring
	w.Write([]byte(html.UI))
}

// return a Mandelbrot image
func handler_image(w http.ResponseWriter, r *http.Request) {
	ReadQueryString(r) // handle the querystring
	SendImage(w)       // fetch the base64 encoded png image
}

const (
	width, height = 1024, 512
	N             = width * height // number of pixels
	tookTooLong   = 0 			   // flag failure, color Black
)

type Point struct {
	Right int // right from TL corner (0,0)
	Down  int // down from ""
}

var (
	perm             [N]int
	Pixel            [N]Point // all the screen points
	position         = 0
	rate             = 64          // how quickly we process - default if we dont  ?r=345 etc
	num              = 5           // multiples of 600
	iterations		 = 3000		   //  always 600 x  TODO kill one of these two
	numberOfroutines = 2           // number of concurrent go routines
	cx, cy           = -0.73, 0.23 // x, y coords of central point pixel (width/2,height/2)
	hScale           = 0.02        // hScale = cx-0
	col              = 1256.0      // color
	options          = map[string]int{
		"m": 1, "r": 1, "num": 1, // 1 = int, 2 = float
		"x": 2, "y": 2, "w": 2,
		"dpx": 1, "dpy": 1,
		"in": 1, "out": 1,
		"col": 2,
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
		case "dpx":
			if position == 0 {
				cx = cx + hScale*float64(2*n-width)/float64(width)
				fmt.Println("new cx = ", cx)
			}
		case "dpy":
			if position == 0 {
				vScale := hScale * (float64(height) / float64(width))
				cy = cy + vScale*float64(height-2*n)/float64(height)
				fmt.Println("new cy = ", cy)
			}
		case "in":
			if position == 0 {
				hScale = hScale * 3 / 4
				fmt.Println("new hScale = ", hScale)
			}
		case "out":
			if position == 0 {
				hScale = 2 * hScale
				fmt.Println("new hScale = ", hScale)
			}
		case "col":
			col = z // change the hue
		}
	}
}

func SendImage(w io.Writer) {
	if position == N { // complete we are done, send this banner to js
		SendBanner(w)
		position = 0
		return
	}

	iterations = num * 600           	// we scale the given iterations
	rgba.SetPalette(num,col)			// set range and hue palette
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
		if op == draw.Src { 	// the first draw operation is the only .Src
			op = draw.Over		// type - the rest are .Over
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
	// fmt.Print("Rate = ", refresh, ", position = ", pos, "\n")
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
		p := NextPoint(k)
		z := Point2C(p)
		d := mandelBrot(z)
		img.Set(p.Right, height-p.Down, rgba.PxColor(d))
	}

	screenChan <- img
}

// func mandelColor(n int) string {
// 	if n == tookTooLong {
// 		return CSSblack
// 	}
// 	return CSSRainbow[n%len(CSSRainbow)]
// }

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
	io.WriteString(w, "_"+sx+"_"+sy+"_"+sr+"_"+si+"_"+st+"_"+sm+"_"+sc)
}

// NextPoint returns the point on the screen
// parametrized by k, using our randomization
func NextPoint(k int) Point {
	m := perm[k] // randomize
	return Pixel[m]
}

// P2C converts a pixel point Q to
// a complex number
func Point2C(p Point) complex128 {
	pr := float64(p.Right)
	pd := float64(p.Down)

	wid := float64(width)
	rx := cx + (pr-wid/2)*(hScale/wid)

	het := float64(height)
	vScale := hScale * (het / wid)
	ry := cy + (pd-het/2)*(vScale/het)

	return complex(rx, ry)
}



func randPermutation() [N]int {
	t := time.Now().Nanosecond()
	rand.Seed(int64(t))

	var v [N]int
	for i, _ := range v {
		v[i] = i // we will start from 0, of course
	}
	for i := 0; i < N-1; i++ {
		j := rand.Intn(N-i) + i // now i <= j <= N-1
		h := v[j]
		v[j] = v[i]
		v[i] = h //swap v_i,v_j
	}
	return v
}

// var CSSRainbow = make([]string, 0)
// var CSSwhite = "#ffffff"
// var CSSblack = "#000000"

func init() {
	k := 0
	for down := 0; down < height; down++ {
		for right := 0; right < width; right++ {
			Pixel[k] = Point{right, down}
			k++
		}
	}
	// for _, c := range rgb.Rainbow {
	// 	CSSRainbow = append(CSSRainbow, fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
	// }
	perm = randPermutation()
	position = 0
	//fmt.Println("INIT: wheel size:", len(CSSRainbow))
}
/*
	localhost:8000
	generates a *color* 1024x1024 mandelbrot progressively
	numberOfroutinesify
	refresh rate (r),
	iterations (num),
	square size (w),
	number of goroutines (m),
	color distribution (col)
	- you may wish to set r to a power of 2
   // http://localhost:8000/?x=-0.727650183203125&y=0.2136008193359375&w=0.0000003375&num=160&r=64&m=7&col=1.2
  //http://localhost:8000/?x=-0.7221093749999999&y=0.24281250000000001&w=0.02&num=1&r=128&m=4&col=-120
*/
	