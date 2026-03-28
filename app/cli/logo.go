package cli

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/qeesung/image2ascii/convert"
	"image/jpeg"
	_ "image/jpeg"
)

var (
	//go:embed logo-hamburger.jpg
	HamburgerLogo []byte
)

func OutputHamburgerLogo() {
	// Create convert options
	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = 64
	convertOptions.FixedHeight = 24

	// Create the image converter
	converter := convert.NewImageConverter()

	img, err := jpeg.Decode(bytes.NewReader(HamburgerLogo))
	if err != nil {
		return
	}

	fmt.Print(converter.Image2ASCIIString(img, &convertOptions))
}
