// Package main generates the executable.
// It consists of two loops handling channel communication between the core and ui
package main

import (
	"mandelbrot/core"
	"mandelbrot/ui"
)


func main() {
	go ui.StartServer()
	j := 0
	for {
		<-ui.RequestImage // wait for user 

		for j != core.LastPiece {
			image, nextJ := core.PartialFrom(j)
			ui.Base64Ready <- image 
			j = nextJ
		}
		banner := []byte(ui.Banner())
		ui.Base64Ready <- banner // banner indicates end of sending the image
		j = 0
		// <-ui.NextPlease          // wait for a request
	}
}
