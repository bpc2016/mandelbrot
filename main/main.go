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
package main

import (
	"mandelbrot/ui"
	"mandelbrot/mandel"
)

const N = ui.Width * ui.Height // N is the total number of pixels in the screen display

func main() {
	go ui.StartServer()
	for {
		for j := 0; j < N; j += 1024 * ui.Ctx.Chunk {
			ui.ImageChan <- mandel.PartialFrom(j)
		}
		banner := []byte(ui.Banner()) 	  
		ui.ImageChan <- banner 			// banner indicates end of sending the image
		<-ui.RequestChan       			// wait for a request
	}
}
