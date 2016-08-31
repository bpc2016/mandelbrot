// Package main generates the executable.
// It consists of two loops handling channel communication between the core and ui
package main

import (
	"mandelbrot/core"
	"mandelbrot/ui"
)

const N = ui.Width * ui.Height // N is the total number of pixels in the screen display

func main() {
	go ui.StartServer()
	for {
		<-ui.RequestImage // wait for this
		 
		for j := 0; j < N; j += 1024 * ui.Ctx.Chunk {
			ui.Base64Ready <- core.PartialFrom(j)
		}
		banner := []byte(ui.Banner())
		ui.Base64Ready <- banner // banner indicates end of sending the image
		// <-ui.NextPlease          // wait for a request
	}
}
