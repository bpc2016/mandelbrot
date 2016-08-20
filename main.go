/*
	localhost:8000
	generates a *color* 1024x1024 mandelbrot progressively
	modify
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
	setCoords()
	getPars(r) // handle the querystring
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

// return a mandelbrot image
func handler_image(w http.ResponseWriter, r *http.Request) {
	getPars(r)  // handle the querystring
	getImage(w) // fetch the base64 encoded png image
}

const (
	width, height = 1024, 1024
	N             = 1024 * 1024 // number of pixels
)

var (
	x           [N]int
	y           [N]int
	perm        [N]int
	position    = 0
	rate        = 128               // how quickly we process - default if we dont  ?r=345 etc
	num         = 1                 // multiples of 600
	Mod         = 4                 // number of concurrent go routines
	cx, cy, rad = -0.73, 0.23, 0.02 // x0=cx-rad, y0=cy-rad
	col         = 1256.0            // color
	options     = map[string]int{
		"m": 1, "r": 1, "num": 1, // 1 = int, 2 = float
		"x": 2, "y": 2, "w": 2,
		"dpx": 1, "dpy": 1,
		"in": 1, "out": 1,
		"col": 2,
	}
)

func setCoords() {
	//fill our coordinate arrays
	k := 0
	for iy := 0; iy < height; iy++ {
		for ix := 0; ix < width; ix++ {
			x[k] = ix
			y[k] = iy
			k++
		}
	}
	perm = randPermutation() // set up the permutation
	position = 0
}

func getPars(r *http.Request) {
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
			Mod = n // global var Mod - number of goroutines
		case "x":
			cx = z // global var center x coord
		case "y":
			cy = z // global var center y coord
		case "w":
			rad = z // global var half a side
		case "dpx":
			if position == 0 {
				cx = cx + rad*float64(n-512)/float64(512)
				fmt.Println("new cx = ", cx)
			}
		case "dpy":
			if position == 0 {
				cy = cy + rad*float64(512-n)/float64(512)
				fmt.Println("new cy = ", cy)
			}
		case "in":
			if position == 0 {
				rad = rad * 3 / 4
				fmt.Println("new rad = ", rad)
			}
		case "out":
			if position == 0 {
				rad = 2 * rad
				fmt.Println("new rad = ", rad)
			}
		case "col":
			col = z // change the hue
		}
	}
}

func getImage(w io.Writer) {
	if position == N { // complete we are done, send this indicator to js
		sx := strconv.FormatFloat(cx, 'f', -1, 64)
		sy := strconv.FormatFloat(cy, 'f', -1, 64)
		sr := strconv.FormatFloat(rad, 'f', -1, 64)
		si := strconv.Itoa(num)
		st := strconv.Itoa(rate)
		sm := strconv.Itoa(Mod)
		sc := strconv.FormatFloat(col, 'f', -1, 64)
		io.WriteString(w, "_"+sx+"_"+sy+"_"+sr+"_"+si+"_"+st+"_"+sm+"_"+sc)
		position = 0
		return
	}
	if rate == 0 {
		rate = 256
	}
	lim := position + 1024*rate
	fmt.Print("Rate = ", rate, ", position = ", position, "\n")
	//fmt.Print("cx = ",cx,", cy = ",cy,", rad = ",rad,"\n")
	if lim > N {
		lim = N
	}

	buf := new(bytes.Buffer)
	c := make(chan image.Image)

	img0 := image.NewRGBA(image.Rect(0, 0, width, height))

	for i := 0; i < Mod; i++ {
		go partialImage(lim, Mod, i, c)
	}

	count := 0
	op := draw.Src
	for img := range c {
		draw.Draw(img0, img0.Bounds(), img, image.ZP, op)
		if op == draw.Src {
			op = draw.Over
		} // swicth the one time
		count++
		if count == Mod {
			close(c)
		}
	}

	png.Encode(io.Writer(buf), img0) // NOTE: ignoring errors, to an io.Writer
	position = lim                   // update

	// convert to base64
	encoder := base64.NewEncoder(base64.StdEncoding, w) // send to target w
	encoder.Write(buf.Bytes())
	encoder.Close()
}

func partialImage(lim int, base int, part int, c chan image.Image) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.ZP, draw.Src)

	var (
		px, py, r_k  int     //pixel positions
		rx, ry       float64 //real, im parts
		x0, y0, side float64 // -0.75, 0.21, 0.04
		z            complex128
	)

	x0 = cx - rad
	y0 = cy - rad
	side = 2 * rad

	for k := position; k < lim; k++ {
		if k%base != part {
			continue
		} // choose our residue class
		r_k = perm[k] // randomize
		px = x[r_k]
		py = y[r_k]
		rx = x0 + float64(px)/width*side
		ry = y0 + float64(py)/height*side
		z = complex(rx, ry)
		img.Set(px, height-py, mandelbrot(z)) //orientation: '-'
	}
	c <- img
}

func mandelbrot(z complex128) color.Color {
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
