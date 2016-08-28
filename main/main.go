// Package main holds the executable
package main

import (
	"mandelbrot/ui"
	"mandelbrot/core"
)

const N = ui.Width * ui.Height // N is the total number of pixels in the screen display

func main() {
	go ui.StartServer()
	for {
		for j := 0; j < N; j += 1024 * ui.Ctx.Chunk {
			ui.ImageChan <- core.PartialFrom(j)
		}
		banner := []byte(ui.Banner()) 	  
		ui.ImageChan <- banner 			// banner indicates end of sending the image
		<-ui.RequestChan       			// wait for a request
	}
}
