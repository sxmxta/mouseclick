// ==============================================================================
// Application startup portal
// ==============================================================================

package main

import (
	"github.com/energye/lcl/lcl"
	"mouseclick/app"
	_ "mouseclick/resources"
)

func main() {
	// Global Initialization
	lcl.Init()
	// Start application message loop
	lcl.Run(app.Forms...)
}
