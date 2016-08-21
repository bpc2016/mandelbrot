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
	"math"
	"math/cmplx"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/", handler_init)        // each request calls handler
	http.HandleFunc("/image/", handler_image) // each request calls handler
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("."))))
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
	fmt.Println("server started on port 8000 ...")
}

// display the base html
func handler_init(w http.ResponseWriter, r *http.Request) {
	InitPoints()
	ReadQueryString(r) // handle the querystring
	// note the use of backtick!!
	w.Write([]byte(`
<html>
<head>
<title>Mandelbrot</title>
<script type="text/javascript" src="static/jquery-1.9.1.min.js"></script>
<script type="text/javascript" src="static/mandelbrot.js"></script>
</head>
<body>
<form>
x: <input type="text" size=8 name="x">
y: <input type="text" size=8 name="y">
wd: <input type="text" size=5 name="w">
itrs: <input type="text" size=3 name="num">
refr: <input type="text" size=3 name="r">
grs: <input type="text" size=3 name="m">
clr: <input type="text" size=3 name="col">
&nbsp;
<input type="submit" value="Submit">
</form>
<p>
<div id="imgs" style="position:relative">
</div>
</body>
</html>`))
}

// return a Mandelbrot image
func handler_image(w http.ResponseWriter, r *http.Request) {
	ReadQueryString(r) // handle the querystring
	SendImage(w)       // fetch the base64 encoded png image
}

const (
	width, height = 1024, 512
	N             = width * height // number of pixels
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

// InitPoints fills the screen pixels array Pixel
// and generates permutation perm
func InitPoints() {
	k := 0
	for down := 0; down < height; down++ {
		for right := 0; right < width; right++ {
			Pixel[k] = Point{right, down}
			k++
		}
	}

	perm = randPermutation()
	position = 0
}

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

	lastIndex := stopAt(rate, position)

	screenChan := make(chan image.Image)

	// set the partial image go routines going
	for gor := 0; gor < numberOfroutines; gor++ {
		go BuildImage(lastIndex, gor, screenChan)
	}

	var canvas *image.RGBA

	// assemble the image
	canvas = image.NewRGBA(image.Rect(0, 0, width, height))
	count := 0
	op := draw.Src
	for img := range screenChan {
		draw.Draw(canvas, canvas.Bounds(), img, image.ZP, op)
		if op == draw.Src { // switch the drawing operation one time
			op = draw.Over
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
	buf := new(bytes.Buffer)
	png.Encode(io.Writer(buf), image) // NOTE: ignoring errors, to an io.Writer
	// convert to Base64
	encoder := base64.NewEncoder(base64.StdEncoding, w) // send to target w
	encoder.Write(buf.Bytes())
	encoder.Close()
}

// stopAt returns the last index for points given
// the 'rate' factor (which decides refresh) and
// current index pos along points list
func stopAt(factor, pos int) int {
	if factor == 0 {
		factor = 256
	}
	lastInd := pos + 1024*factor
	// fmt.Print("Rate = ", factor, ", position = ", pos, "\n")
	if lastInd > N {
		lastInd = N
	}
	return lastInd
}

// BuildImage generates a partial image of the Mandelbrot set and sends
// this to screenChan
func BuildImage(lastIndex int, part int, screenChan chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	for k := position; k < lastIndex; k++ {
		if k%numberOfroutines != part { // choose our residue class
			continue
		}
		P := NextPoint(k)
		z := Point2C(P)
		img.Set(P.Right, height-P.Down, GetColor(z))
	}

	screenChan <- img
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
	// hside := 2 * hScale
	// vside := 2 * vScale
	// rx := cx - hScale + float64(pr)/width*hside
	// ry := cy - vScale + float64(pd)/height*vside
	return complex(rx, ry)
}

// GetColor runs the Mandelbrot iteration from point z
// it returns a color for the point dependent on how many
// iterations are required to escape. If this is more than
// our bound we assign color.Black
func GetColor(z complex128) color.Color {
	var (
		c uint64
		v complex128
		w [3]uint8
	)
	iterations := num * 600           // we scale the given iterations
	for n := 0; n < iterations; n++ { // iterations comes from global 'num' !!
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			c = coloRatio(n, iterations)
			w = getColors(c)
			return color.RGBA{w[0], w[1], w[2], 255}
		}
	}
	return color.Black
}

func coloRatio(z, max int) uint64 {
	const T = 8355771 //codeColors([3]int{127,127,127})
	var B = col
	x := float64(z) / float64(max)
	y := (1 - B*x - (1-B*float64(max))*x*x) * float64(T)
	return uint64(math.Floor(y))
}

func codeColors(c [3]int) int {
	const B = 1 << 8
	return ((c[0]*B)+c[1])*B + c[2]
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
