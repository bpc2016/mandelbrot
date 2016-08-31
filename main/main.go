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
		ui.Base64Ready <- banner 
		j = 0
	}
}
// TODO - convert the jquery material to pure js, so that this a well as
// mandelbrot.js can be subsumed in the constant indexHTML - then the entire
// program is independent of location - NO need for static html

// TODO - furtehr study of the use of a linear color scheme - clearly wants a prior run
// to determine the extent of the possible 'n' values <---> colors 